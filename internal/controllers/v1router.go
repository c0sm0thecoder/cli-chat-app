package controllers

import (
	"os"

	"github.com/c0sm0thecoder/cli-chat-app/internal/middlewares"
	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/go-chi/chi/v5"
)

// NewV1Router creates a new router for API v1
func NewV1Router(authService services.AuthService, chatService services.ChatService) chi.Router {
	r := chi.NewRouter()

	// Create controllers
	authController := NewAuthController(authService)
	roomController := NewRoomController(chatService)

	// Register public routes (no authentication required)
	authController.RegisterRoutes(r)

	// Protected routes that require authentication
	r.Group(func(r chi.Router) {
		r.Use(middlewares.JWTMiddleware(os.Getenv("JWT_SECRET")))
		
		// Register protected routes
		roomController.RegisterRoutes(r)
	})

	return r
}
