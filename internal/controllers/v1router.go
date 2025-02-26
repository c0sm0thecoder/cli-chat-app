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
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := authService.SignUp(req.UserName, req.Password); err != nil {
			http.Error(w, "Request failed:"+err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
	})

	router.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		token, err := authService.Login(req.UserName, req.Password)
		if err != nil {
			http.Error(w, "Invalid Credentials:"+err.Error(), http.StatusUnauthorized)
		}

		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	router.Group(func(r chi.Router) {
		r.Use(middlewares.JWTMiddleware(os.Getenv("JWT_SECRET")))

		r.Post("/rooms", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}
			if err := chatService.CreateRoom(req.Name); err != nil {
				http.Error(w, "Error creating room:"+err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
		})

		r.Post("/rooms/{code}/messages", func(w http.ResponseWriter, r *http.Request) {
			roomCode := chi.URLParam(r, "code")

			room, err := chatService.GetRoomByCode(roomCode)
			if err != nil {
				http.Error(w, "Room could not be found", http.StatusBadRequest)
				return
			}

			var req struct {
				SenderID string `json:"sender_id"`
				Content  string `json:"content"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request", http.StatusBadRequest)
				return
			}

			if err := chatService.SendMessage(room.ID, req.SenderID, req.Content); err != nil {
				http.Error(w, "Error sending message", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		})
	})

	router.Mount("/api/v1", router)
	return router
}
