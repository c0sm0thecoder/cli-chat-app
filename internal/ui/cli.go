package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
)

const (
	baseURL = "http://localhost:8080/api/v1"
	wsURL   = "ws://localhost:8080/ws"
)

// Global session state and UI components
var (
	authToken  string
	username   string
	roomCode   string
	wsConn     *websocket.Conn
	app        *tview.Application
	pages      *tview.Pages
	msgView    *tview.TextView
	statusBar  *tview.TextView
	httpClient = &http.Client{}

	// Global references for setting focus later.
	loginForm      *tview.Form
	roomForm       *tview.Form
	chatInputField *tview.InputField
)

// StartCLI initializes and runs the CLI application
func StartCLI() error {
	app = tview.NewApplication()
	pages = tview.NewPages()

	// Create status bar for notifications
	statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	statusBar.SetBackgroundColor(tcell.ColorBlack)

	// Initialize screens
	createLoginScreen()
	createRoomScreen()
	createChatScreen()

	// Main layout with status bar at bottom
	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true).
		AddItem(statusBar, 1, 0, false)

	// Set pages as root and start with login
	pages.SwitchToPage("login")
	return app.SetRoot(mainLayout, true).Run()
}

// showStatus displays a temporary status message without blocking
func showStatus(message string, isError bool) {
	app.QueueUpdateDraw(func() {
		color := "green"
		if isError {
			color = "red"
		}
		statusBar.Clear()
		fmt.Fprintf(statusBar, "[%s]%s[-]", color, message)
		// Auto-clear after 3 seconds
		go func() {
			time.Sleep(3 * time.Second)
			app.QueueUpdateDraw(func() {
				statusBar.Clear()
			})
		}()
	})
}

func createLoginScreen() {
	form := tview.NewForm()
	form.AddInputField("Username", "", 20, nil, func(text string) {
		username = text
	})
	form.AddPasswordField("Password", "", 20, '*', nil)
	form.AddButton("Login", func() {
		pass := form.GetFormItem(1).(*tview.InputField).GetText()
		if username == "" || pass == "" {
			showStatus("Please enter both username and password", true)
			return
		}
		handleAuth("/login", pass, form)
	})
	form.AddButton("Signup", func() {
		pass := form.GetFormItem(1).(*tview.InputField).GetText()
		if username == "" || pass == "" {
			showStatus("Please enter both username and password", true)
			return
		}
		handleAuth("/signup", pass, form)
	})
	form.AddButton("Quit", func() {
		app.Stop()
	})
	form.SetBorder(true).SetTitle(" Login/Signup ").SetTitleAlign(tview.AlignCenter)
	// Save reference for later focus setting.
	loginForm = form
	pages.AddPage("login", form, true, false)
}

func handleAuth(endpoint string, password string, form *tview.Form) {
	log.Printf("[DEBUG] Pre-goroutine")
	log.Println("Username:", username)
	log.Println("Password:", password)

	// Create request body before goroutine
	reqBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		log.Printf("[DEBUG] JSON marshal error: %v", err)
		showStatus("Invalid input", true)
		return
	}

	// Start network request in goroutine
	go func() {
		log.Printf("[DEBUG] Inside goroutine")
		defer log.Printf("[DEBUG] Goroutine exit")
		req, err := http.NewRequest("POST", baseURL+endpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			log.Printf("[DEBUG] Request creation failed: %v", err)
			app.QueueUpdateDraw(func() {
				showStatus("Request failed: "+err.Error(), true)
			})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		log.Printf("[DEBUG] Response: %+v, Error: %v", resp, err)
		if err != nil {
			app.QueueUpdateDraw(func() {
				showStatus("Connection failed: "+err.Error(), true)
				form.SetTitle(" Login/Signup ")
			})
			return
		}
		defer resp.Body.Close()
		// Process response in UI thread
		app.QueueUpdateDraw(func() {
			form.SetTitle(" Login/Signup ")
			handleAuthResponse(endpoint, resp, form)
		})
	}()
	log.Printf("[DEBUG] Post-goroutine launch")
}

// handleAuthResponse processes the authentication response
func handleAuthResponse(endpoint string, resp *http.Response, form *tview.Form) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		showStatus(fmt.Sprintf("Authentication failed (Status: %d)", resp.StatusCode), true)
		return
	}
	if endpoint == "/login" {
		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			showStatus("Invalid response: "+err.Error(), true)
			return
		}
		authToken = result["token"]
		app.QueueUpdateDraw(func() {
			form.GetFormItem(1).(*tview.InputField).SetText("")
			pages.SwitchToPage("room")
			// Set focus to the room screen (stored globally)
			app.SetFocus(roomForm)
			showStatus("Login successful", false)
		})
	} else {
		app.QueueUpdateDraw(func() {
			form.GetFormItem(0).(*tview.InputField).SetText("")
			form.GetFormItem(1).(*tview.InputField).SetText("")
			username = ""
			app.SetFocus(loginForm)
			showStatus("Signup successful! Please login", false)
		})
	}
}

func createRoomScreen() {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(" Room Selection ").
		SetTitleAlign(tview.AlignCenter)
	// Add input fields
	form.AddInputField("Room Name", "", 20, nil, nil)
	form.AddInputField("Room Code", "", 12, nil, nil)
	// Add buttons
	form.AddButton("Create Room", func() {
		roomName := form.GetFormItem(0).(*tview.InputField).GetText()
		if roomName == "" {
			showStatus("Please enter a room name", true)
			return
		}
		handleCreateRoom(roomName)
	})
	form.AddButton("Join Room", func() {
		code := form.GetFormItem(1).(*tview.InputField).GetText()
		if code == "" {
			showStatus("Please enter a room code", true)
			return
		}
		handleJoinRoom(code)
	})
	form.AddButton("Logout", func() {
		authToken = ""
		app.QueueUpdateDraw(func() {
			pages.SwitchToPage("login")
			// Set focus back to the login form.
			app.SetFocus(loginForm)
			showStatus("Logged out", false)
		})
	})
	// Save reference for later focus setting.
	roomForm = form
	pages.AddPage("room", form, true, false)
}

func handleCreateRoom(roomName string) {
	showStatus("Creating room...", false)
	go func() {
		reqBody, _ := json.Marshal(map[string]string{"name": roomName})
		req, _ := http.NewRequest("POST", baseURL+"/rooms", bytes.NewBuffer(reqBody))
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			showStatus("Failed to create room: "+err.Error(), true)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			showStatus(fmt.Sprintf("Failed to create room (Status: %d)", resp.StatusCode), true)
			return
		}
		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			showStatus("Invalid server response: "+err.Error(), true)
			return
		}
		roomCode = result["code"]
		connectToChat()
	}()
}

func handleJoinRoom(code string) {
	showStatus("Joining room...", false)
	roomCode = code
	connectToChat()
}

func createChatScreen() {
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	// Messages area
	msgView = tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	msgView.SetBorder(true).SetTitle(" Messages ").SetTitleAlign(tview.AlignCenter)
	// Input area
	inputField := tview.NewInputField()
	inputField.
		SetLabel("> ").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				text := inputField.GetText()
				if text != "" {
					sendMessage(text)
					inputField.SetText("")
				}
			}
		})
	// Save reference for later focus setting.
	chatInputField = inputField

	// Button area
	buttonBar := tview.NewFlex().SetDirection(tview.FlexColumn)
	exitButton := tview.NewButton("Exit Chat").SetSelectedFunc(func() {
		if wsConn != nil {
			wsConn.Close()
			wsConn = nil
		}
		pages.SwitchToPage("room")
		showStatus("Left chat room", false)
	})
	buttonBar.AddItem(nil, 0, 1, false).
		AddItem(exitButton, 10, 0, true).
		AddItem(nil, 0, 1, false)

	// Combine all elements
	mainFlex.AddItem(msgView, 0, 1, false).
		AddItem(inputField, 1, 0, true).
		AddItem(buttonBar, 1, 0, false)

	// Handle keyboard shortcuts
	mainFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.SetFocus(exitButton)
			return nil
		}
		if event.Key() == tcell.KeyTab {
			if inputField.HasFocus() {
				app.SetFocus(exitButton)
			} else {
				app.SetFocus(inputField)
			}
			return nil
		}
		return event
	})
	pages.AddPage("chat", mainFlex, true, false)
}

func connectToChat() {
	// Establish WebSocket connection
	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Add("Authorization", "Bearer "+authToken)
	conn, _, err := dialer.Dial(wsURL+"?room="+roomCode, header)
	if err != nil {
		showStatus("Failed to connect to chat: "+err.Error(), true)
		return
	}
	wsConn = conn
	app.QueueUpdateDraw(func() {
		// Clear previous messages
		msgView.Clear()
		msgView.SetTitle(fmt.Sprintf(" Chat Room: %s ", roomCode))
		// Add welcome message
		fmt.Fprintf(msgView, "[green]Connected to room %s as %s[white]\n", roomCode, username)
		pages.SwitchToPage("chat")
		// Set focus to the chat input field.
		app.SetFocus(chatInputField)
		showStatus("Connected to chat room", false)
		app.Draw()
	})
	// Start listening for messages
	go func() {
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				if !strings.Contains(err.Error(), "normal closure") {
					showStatus("Connection lost: "+err.Error(), true)
					app.QueueUpdateDraw(func() {
						pages.SwitchToPage("room")
					})
				}
				return
			}
			app.QueueUpdateDraw(func() {
				fmt.Fprintf(msgView, "%s\n", message)
			})
		}
	}()
}

func sendMessage(text string) {
	if wsConn == nil {
		showStatus("Not connected to chat", true)
		return
	}
	if err := wsConn.WriteMessage(websocket.TextMessage, []byte(text)); err != nil {
		showStatus("Failed to send message: "+err.Error(), true)
	}
}
