package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/go-chi/chi/v5"
)

type AuthController struct {
	authService services.AuthService
}

func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// RegisterRoutes registers all auth-related routes
func (c *AuthController) RegisterRoutes(r chi.Router) {
	r.Post("/register", c.Register)
	r.Post("/login", c.Login)
}

// Register handles user registration
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	
	err := c.authService.SignUp(req.Username, req.Password)
	if err != nil {
		switch err {
		case services.ErrUserAlreadyExists:
			http.Error(w, "Username already exists", http.StatusConflict)
		default:
			http.Error(w, "Registration failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User registered successfully",
	})
}

// Login handles user authentication
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}
	
	token, err := c.authService.Login(req.Username, req.Password)
	if err != nil {
		switch err {
		case services.ErrUserNotFound, services.ErrInvalidCredentials:
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		default:
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
