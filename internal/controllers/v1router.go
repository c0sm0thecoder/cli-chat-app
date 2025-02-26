package controllers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/c0sm0thecoder/cli-chat-app/internal/middlewares"
	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/go-chi/chi/v5"
)

type v1Router struct {
	AuthService services.AuthService
	ChatService services.ChatService
}

func NewV1Router(authService services.AuthService, chatService services.ChatService) http.Handler {
	router := chi.NewRouter()

	router.Use(middlewares.LoggingMiddleware)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	router.Post("/signup", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserName == "" || req.Password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		if err := authService.SignUp(req.UserName, req.Password); err != nil {
			switch err {
			case services.ErrUserAlreadyExists:
				http.Error(w, "Username already exists", http.StatusConflict)
			default:
				http.Error(w, "Failed to create account: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "User created successfully"})
	})

	router.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserName == "" || req.Password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		token, err := authService.Login(req.UserName, req.Password)
		if err != nil {
			switch err {
			case services.ErrUserNotFound, services.ErrInvalidCredentials:
				http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			default:
				http.Error(w, "Login failed: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	router.Group(func(r chi.Router) {
		r.Use(middlewares.JWTMiddleware(os.Getenv("JWT_SECRET")))

		// Update the room creation handler to match the new service interface
		r.Post("/rooms", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			
			if req.Name == "" {
				http.Error(w, "Room name is required", http.StatusBadRequest)
				return
			}
			
			room, err := chatService.CreateRoom(req.Name)
			if err != nil {
				switch err {
				case services.ErrEmptyRoomName:
					http.Error(w, "Room name cannot be empty", http.StatusBadRequest)
				default:
					http.Error(w, "Error creating room: "+err.Error(), http.StatusInternalServerError)
				}
				return
			}
			
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"id": room.ID,
				"name": room.Name,
				"code": room.Code,
			})
		})

		r.Post("/rooms/{code}/messages", func(w http.ResponseWriter, r *http.Request) {
			roomCode := chi.URLParam(r, "code")
			if roomCode == "" {
				http.Error(w, "Room code is required", http.StatusBadRequest)
				return
			}

			room, err := chatService.GetRoomByCode(roomCode)
			if err != nil {
				http.Error(w, "Room not found", http.StatusNotFound)
				return
			}

			var req struct {
				SenderID string `json:"sender_id"`
				Content  string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
			
			if req.Content == "" {
				http.Error(w, "Message content is required", http.StatusBadRequest)
				return
			}

			if err := chatService.SendMessage(room.ID, req.SenderID, req.Content); err != nil {
				http.Error(w, "Error sending message: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"message": "Message sent successfully"})
		})
		
		// Add an endpoint to get room messages
		r.Get("/rooms/{code}/messages", func(w http.ResponseWriter, r *http.Request) {
			roomCode := chi.URLParam(r, "code")
			if roomCode == "" {
				http.Error(w, "Room code is required", http.StatusBadRequest)
				return
			}

			room, err := chatService.GetRoomByCode(roomCode)
			if err != nil {
				http.Error(w, "Room not found", http.StatusNotFound)
				return
			}

			messages, err := chatService.GetMessages(room.ID)
			if err != nil {
				http.Error(w, "Error retrieving messages: "+err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(messages)
		})
	})

	return router
}
