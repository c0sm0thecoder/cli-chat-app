package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
)

var (
	app             *tview.Application
	pages           *tview.Pages
	apiBaseURL      string
	wsBaseURL       string  // WebSocket URL
	authToken       string
	username        string
	currentRoomID   string
	currentRoomCode string
	chatDisplay     *tview.TextView
	messageInput    *tview.InputField
	roomCodeInput   *tview.InputField
	wsConn          *websocket.Conn // WebSocket connection
	stopWebsocket   chan struct{}   // Channel to signal stopping the WebSocket
)

// Define global form variables
var (
	loginForm  *tview.Form
	signupForm *tview.Form
)

// StartCLI initializes and starts the CLI application
func StartCLI(serverURL string) {
	// Make sure API URL has the correct format
	if serverURL == "" {
		serverURL = "http://localhost:8080/api/v1"
	} else if !strings.HasSuffix(serverURL, "/api/v1") {
		// Check if we need to append /api/v1
		if strings.HasSuffix(serverURL, "/") {
			serverURL = serverURL + "api/v1"
		} else {
			serverURL = serverURL + "/api/v1"
		}
	}
	
	apiBaseURL = serverURL
	
	// Create application and pages
	app = tview.NewApplication()
	pages = tview.NewPages()

	// Setup login form
	setupLoginForm()
	
	// Setup signup form
	setupSignupForm()

	// Add login page as the default
	pages.AddPage("login", createLoginPage(), true, true)
	pages.AddPage("signup", createSignupPage(), true, false)

	// Global key handling
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlQ {
			app.Stop()
			return nil
		} else if event.Key() == tcell.KeyEsc {
			// ESC key returns to login from any page
			if pages.HasPage("modal") {
				pages.RemovePage("modal")
			} else if name, _ := pages.GetFrontPage(); name != "login" {
				pages.SwitchToPage("login")
			}
			return nil
		}
		return event
	})

	// Start the application
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("Error running application: %v", err)
	}

	// Clean up when app exits
	if wsConn != nil {
		wsConn.Close()
		if stopWebsocket != nil {
			close(stopWebsocket)
		}
	}
}

// setupLoginForm creates the login form
func setupLoginForm() {
	loginForm = tview.NewForm()
	loginForm.AddInputField("Username", "", 30, nil, nil)
	loginForm.AddPasswordField("Password", "", 30, '*', nil)
	
	loginForm.AddButton("Login", func() {
		usernameInput := loginForm.GetFormItem(0).(*tview.InputField).GetText()
		password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
		
		// Validate input
		if usernameInput == "" || password == "" {
			showInfoModal("Error", "Username and password cannot be empty")
			return
		}
		
		// Attempt login
		token, err := login(usernameInput, password)
		if err != nil {
			showInfoModal("Login Failed", err.Error())
			return
		}
		
		// Store credentials
		authToken = token
		username = usernameInput
		
		// Navigate to rooms page
		setupRoomsPage()
		pages.SwitchToPage("rooms")
	})
	
	loginForm.AddButton("Sign Up", func() {
		// Switch to signup page
		pages.SwitchToPage("signup")
	})
	
	loginForm.AddButton("Quit", func() {
		app.Stop()
	})
}

// setupSignupForm creates the signup form
func setupSignupForm() {
	signupForm = tview.NewForm()
	signupForm.AddInputField("Username", "", 30, nil, nil)
	signupForm.AddPasswordField("Password", "", 30, '*', nil)
	signupForm.AddPasswordField("Confirm Password", "", 30, '*', nil)
	
	signupForm.AddButton("Register", func() {
		usernameInput := signupForm.GetFormItem(0).(*tview.InputField).GetText()
		password := signupForm.GetFormItem(1).(*tview.InputField).GetText()
		confirmPassword := signupForm.GetFormItem(2).(*tview.InputField).GetText()
		
		// Validate input
		if usernameInput == "" || password == "" {
			showInfoModal("Error", "Username and password cannot be empty")
			return
		}
		
		if password != confirmPassword {
			showInfoModal("Error", "Passwords do not match")
			return
		}
		
		// Attempt registration
		err := registerUser(usernameInput, password)
		if err != nil {
			showInfoModal("Registration Failed", err.Error())
			return
		}
		
		// Show success and return to login
		showInfoModal("Registration Successful", "You can now log in with your credentials")
		
		// Clear the form
		for i := 0; i < signupForm.GetFormItemCount()-3; i++ {
			if field, ok := signupForm.GetFormItem(i).(*tview.InputField); ok {
				field.SetText("")
			}
		}
		
		// Switch back to login page after successful registration
		pages.SwitchToPage("login")
	})
	
	signupForm.AddButton("Back to Login", func() {
		pages.SwitchToPage("login")
	})
}

// createLoginPage creates a page containing the login form
func createLoginPage() tview.Primitive {
	// Title
	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("CLI Chat Application")
	
	// Footer with help text
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Ctrl+Q to quit | ESC to go back")
	
	// Layout with title, form, and footer
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 3, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(loginForm, 0, 3, true).
			AddItem(nil, 0, 1, false),
		0, 1, true).
		AddItem(footer, 1, 1, false)
	
	// Set borders and titles
	loginForm.SetBorder(true).
		SetTitle(" Login ").
		SetTitleAlign(tview.AlignCenter)
	
	return flex
}

// createSignupPage creates a page containing the signup form
func createSignupPage() tview.Primitive {
	// Title
	title := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Register New Account")
	
	// Footer with help text
	footer := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Ctrl+Q to quit | ESC to go back")
	
	// Layout with title, form, and footer
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(title, 3, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(signupForm, 0, 3, true).
			AddItem(nil, 0, 1, false),
		0, 1, true).
		AddItem(footer, 1, 1, false)
	
	// Set borders and titles
	signupForm.SetBorder(true).
		SetTitle(" Sign Up ").
		SetTitleAlign(tview.AlignCenter)
	
	return flex
}

// registerUser registers a new user with the server
func registerUser(username, password string) error {
	// Prepare request data
	reqData := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to prepare request: %v", err)
	}

	// Create HTTP request - make sure you're using the full endpoint path
	req, err := http.NewRequest("POST", apiBaseURL+"/register", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection error: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	return nil
}

// setupRoomsPage creates the rooms page with the current username
func setupRoomsPage() {
	roomsPage := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("Welcome to Chat App, " + username), 3, 1, false).
		AddItem(tview.NewList().
			AddItem("Create Room", "Create a new chat room", 'c', func() {
				showCreateRoomModal()
			}).
			AddItem("Join Room", "Join an existing room", 'j', func() {
				showJoinRoomModal()
			}).
			AddItem("Logout", "Return to login screen", 'l', func() {
				authToken = ""
				username = ""
				pages.SwitchToPage("login")
			}), 0, 1, true)

	roomsPage.SetBorder(true).
		SetTitle(" Chat Rooms ").
		SetTitleAlign(tview.AlignCenter)

	pages.AddPage("rooms", roomsPage, true, false)
}

// login sends a login request to the API
func login(username, password string) (string, error) {
	// Prepare request data
	reqData := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to prepare request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiBaseURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection error: %v", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("invalid username or password")
		}
		return "", fmt.Errorf("server error (status %d)", resp.StatusCode)
	}

	// Parse response
	var respData struct {
		Token string `json:"token"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if respData.Token == "" {
		return "", fmt.Errorf("no token received from server")
	}

	return respData.Token, nil
}

// showInfoModal displays an information modal with a message
func showInfoModal(title, message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			pages.RemovePage("modal")
		})
	
	modal.SetBorder(true).
		SetTitle(" " + title + " ").
		SetTitleAlign(tview.AlignCenter)
	
	pages.AddPage("modal", modal, false, true)
}

// showCreateRoomModal displays a modal for creating a new room
func showCreateRoomModal() {
	form := tview.NewForm()
	form.AddInputField("Room Name", "", 30, nil, nil)
	form.AddButton("Create", func() {
		roomName := form.GetFormItem(0).(*tview.InputField).GetText()
		if roomName == "" {
			showInfoModal("Error", "Room name cannot be empty")
			return
		}
		
		// Create room via API
		room, err := createRoom(roomName)
		if err != nil {
			showInfoModal("Error", "Failed to create room: "+err.Error())
			return
		}
		
		// Successfully created room - show room code
		currentRoomID = room.ID
		currentRoomCode = room.Code
		
		// Show success message with room code
		showInfoModal("Room Created", fmt.Sprintf("Room '%s' created successfully!\nRoom Code: %s", 
			room.Name, room.Code))
		
		// Setup the chat room
		setupChatRoom(room)
		
		// Remove the modal and switch to chat page
		pages.RemovePage("createRoomModal")
		pages.SwitchToPage("chat")
	})
	form.AddButton("Cancel", func() {
		pages.RemovePage("createRoomModal")
	})
	
	form.SetBorder(true).
		SetTitle(" Create New Room ").
		SetTitleAlign(tview.AlignCenter)
	
	pages.AddPage("createRoomModal", tview.NewGrid().
		SetColumns(0, 40, 0).
		SetRows(0, 10, 0).
		AddItem(form, 1, 1, 1, 1, 0, 0, true), true, true)
}

// showJoinRoomModal displays a modal for joining an existing room
func showJoinRoomModal() {
	// Create input field for room code
	roomCodeInput = tview.NewInputField().
		SetLabel("Room Code: ").
		SetFieldWidth(20).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				code := roomCodeInput.GetText()
				if code != "" {
					pages.RemovePage("joinRoomModal")
					joinRoom(code)
				}
			}
		})

	// Create the form with the input field and buttons
	form := tview.NewForm().
		AddFormItem(roomCodeInput).
		AddButton("Join", func() {
			code := roomCodeInput.GetText()
			if code != "" {
				pages.RemovePage("joinRoomModal")
				joinRoom(code)
			}
		}).
		AddButton("Cancel", func() {
			pages.RemovePage("joinRoomModal")
		})

	form.SetBorder(true).
		SetTitle(" Join Room ").
		SetTitleAlign(tview.AlignCenter)

	// Create modal layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 40, 1, true).
			AddItem(nil, 0, 1, false),
			10, 1, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("joinRoomModal", flex, true, true)
	app.SetFocus(roomCodeInput) // Set focus on the input field
}

// createRoom sends a request to create a new room
func createRoom(roomName string) (*models.Room, error) {
	// Prepare request data
	reqData := map[string]string{
		"name": roomName,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiBaseURL+"/rooms", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection error: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Handle different status codes
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized: invalid or expired token")
	} else if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("server error (status %d)", resp.StatusCode)
	}

	// Parse response
	var roomResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Code string `json:"code"`
	}
	
	if err := json.Unmarshal(respBody, &roomResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Create a room object from the response
	room := &models.Room{
		ID:   roomResp.ID,
		Name: roomResp.Name,
		Code: roomResp.Code,
	}

	return room, nil
}

// setupChatRoom creates the chat interface for a specific room
func setupChatRoom(room *models.Room) {
	// Close existing WebSocket connection if any
	if wsConn != nil {
		wsConn.Close()
		if stopWebsocket != nil {
			close(stopWebsocket)
		}
	}

	// Store current room info
	currentRoomID = room.ID
	currentRoomCode = room.Code

	// Display room information
	roomTitle := fmt.Sprintf(" Room: %s (Code: %s) ", room.Name, room.Code)

	// Setup chat display
	chatDisplay = tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	chatDisplay.SetBorder(true).SetTitle(roomTitle)

	// Create message input field
	messageInput = tview.NewInputField().
		SetLabel("Message: ").
		SetFieldWidth(0).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				content := messageInput.GetText()
				if content != "" {
					sendMessage(currentRoomID, content)
					messageInput.SetText("")
				}
			}
		})

	// Create layout
	chatFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatDisplay, 0, 1, false).
		AddItem(messageInput, 3, 1, true)

	// Add keybindings
	chatFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.SwitchToPage("rooms")
			return nil
		}
		return event
	})

	// Add page and display
	pages.AddPage("chat", chatFlex, true, false)
	
	// Load existing messages
	fetchMessages(currentRoomID)
	
	// Connect to WebSocket for real-time updates
	connectWebSocket(room.ID)
}

// connectWebSocket establishes a WebSocket connection for real-time messages
func connectWebSocket(roomID string) {
	// Parse the API base URL to create WebSocket URL
	apiURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return
	}
	
	// Change scheme http -> ws, https -> wss
	wsScheme := "ws"
	if apiURL.Scheme == "https" {
		wsScheme = "wss"
	}
	
	// Create WebSocket URL
	wsURL := url.URL{
		Scheme: wsScheme,
		Host:   apiURL.Host,
		Path:   strings.TrimSuffix(apiURL.Path, "/api/v1") + "/api/v1/ws",
	}
	
	// Add query parameters for room and auth
	q := wsURL.Query()
	q.Set("room_id", roomID)
	q.Set("token", authToken)
	wsURL.RawQuery = q.Encode()
	
	// Connect to WebSocket server
	header := http.Header{}
	header.Add("Authorization", "Bearer "+authToken)
	wsConnection, _, err := websocket.DefaultDialer.Dial(wsURL.String(), header)
	if err != nil {
		return
	}
	
	wsConn = wsConnection
	stopWebsocket = make(chan struct{})
	
	// Start goroutine to handle incoming WebSocket messages
	go func() {
		defer wsConn.Close()
		
		for {
			select {
			case <-stopWebsocket:
				return
			default:
				// Read message from WebSocket
				_, message, err := wsConn.ReadMessage()
				if err != nil {
					return
				}
				
				// Parse the message
				var wsMessage struct {
					Type      string    `json:"type"`
					ID        string    `json:"id,omitempty"`
					RoomID    string    `json:"room_id,omitempty"`
					SenderID  string    `json:"sender_id,omitempty"`
					Username  string    `json:"username,omitempty"`
					Content   string    `json:"content,omitempty"`
					CreatedAt time.Time `json:"created_at,omitempty"`
				}
				
				if err := json.Unmarshal(message, &wsMessage); err != nil {
					continue
				}
				
				// Handle different message types
				if wsMessage.Type == "new_message" {
					// Skip displaying messages from ourselves (to avoid duplicates)
					// since we already show the message when we send it
					if wsMessage.SenderID != username {
						app.QueueUpdateDraw(func() {
							displayMessage(wsMessage.Username, wsMessage.Content, wsMessage.CreatedAt)
						})
					}
				}
			}
		}
	}()
}

// sendMessage sends a chat message to the server
func sendMessage(roomID, content string) {
	// Prepare request data
	reqData := map[string]string{
		"content": content,
	}
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		showInfoModal("Error", "Failed to prepare message: "+err.Error())
		return
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/rooms/%s/messages", apiBaseURL, roomID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		showInfoModal("Error", "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		showInfoModal("Error", "Connection error: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized {
		showInfoModal("Error", "Unauthorized: Token may be invalid or expired")
		return
	} else if resp.StatusCode != http.StatusCreated {
		showInfoModal("Error", fmt.Sprintf("Failed to send message (status %d)", resp.StatusCode))
		return
	}

	// Display message locally immediately (without waiting for WebSocket)
	var message struct {
		ID        string    `json:"id"`
		RoomID    string    `json:"room_id"`
		SenderID  string    `json:"sender_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		return
	}
	
	// Display our own message immediately
	displayMessage(username, message.Content, message.CreatedAt)
}

// fetchMessages gets all messages for a room
func fetchMessages(roomID string) {
	// Create HTTP request
	req, err := http.NewRequest("GET", apiBaseURL+"/rooms/"+roomID+"/messages", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+authToken)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return
	}

	// Parse the response
	var messages []struct {
		ID        string    `json:"id"`
		RoomID    string    `json:"room_id"`
		SenderID  string    `json:"sender_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return
	}

	// Clear previous messages
	chatDisplay.Clear()
	displayMessage("System", "Welcome to the chat room!", time.Now())
	displayMessage("System", "Press Ctrl+Q to quit, ESC to go back", time.Now())

	// Display messages
	for _, msg := range messages {
		displayMessage(msg.SenderID, msg.Content, msg.CreatedAt)
	}
}

// displayMessage adds a message to the chat display
func displayMessage(senderID, content string, timestamp time.Time) {
	timeStr := timestamp.Format("15:04:05")
	
	var senderName string
	if senderID == username {
		senderName = "[green]You[-]"
	} else if senderID == "System" {
		senderName = "[blue]System[-]"
	} else {
		senderName = "[yellow]" + senderID + "[-]"
	}
	
	msg := fmt.Sprintf("[gray]%s[-] %s: %s\n", timeStr, senderName, content)
	fmt.Fprint(chatDisplay, msg)
	
	// Scroll to end
	chatDisplay.ScrollToEnd()
}

// joinRoom sends a request to join an existing room
func joinRoom(roomCode string) {
	// Create HTTP request
	req, err := http.NewRequest("GET", apiBaseURL+"/rooms/code/"+roomCode, nil)
	if err != nil {
		showInfoModal("Error", "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Authorization", "Bearer "+authToken)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		showInfoModal("Error", "Connection error: "+err.Error())
		return
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			showInfoModal("Error", "Room not found")
		} else {
			showInfoModal("Error", fmt.Sprintf("Failed to join room (status %d)", resp.StatusCode))
		}
		return
	}

	// Parse the response
	var room models.Room
	if err := json.NewDecoder(resp.Body).Decode(&room); err != nil {
		showInfoModal("Error", "Failed to parse response: "+err.Error())
		return
	}

	// Show the chat interface
	setupChatRoom(&room)
	pages.SwitchToPage("chat")
}
