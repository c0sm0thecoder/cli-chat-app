package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/go-chi/chi/v5"
)

type RoomController struct {
	chatService services.ChatService
}

func NewRoomController(chatService services.ChatService) *RoomController {
	return &RoomController{
		chatService: chatService,
	}
}

// RegisterRoutes registers all room-related routes
func (c *RoomController) RegisterRoutes(r chi.Router) {
	r.Post("/rooms", c.CreateRoom)
	r.Get("/rooms/code/{code}", c.GetRoomByCode)
	r.Get("/rooms/{roomID}/messages", c.GetMessages)
	r.Post("/rooms/{roomID}/messages", c.SendMessage)
}

// CreateRoom handles room creation requests
func (c *RoomController) CreateRoom(w http.ResponseWriter, r *http.Request) {
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
	
	room, err := c.chatService.CreateRoom(req.Name)
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
	json.NewEncoder(w).Encode(room)
}

// GetRoomByCode retrieves a room by its join code
func (c *RoomController) GetRoomByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		http.Error(w, "Room code is required", http.StatusBadRequest)
		return
	}
	
	room, err := c.chatService.GetRoomByCode(code)
	if err != nil {
		switch err {
		case services.ErrRoomNotFound:
			http.Error(w, "Room not found", http.StatusNotFound)
		default:
			http.Error(w, "Error retrieving room: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	json.NewEncoder(w).Encode(room)
}

// GetMessages retrieves all messages for a room
func (c *RoomController) GetMessages(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}
	
	messages, err := c.chatService.GetMessages(roomID)
	if err != nil {
		switch err {
		case services.ErrRoomNotFound:
			http.Error(w, "Room not found", http.StatusNotFound)
		default:
			http.Error(w, "Error retrieving messages: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	json.NewEncoder(w).Encode(messages)
}

// SendMessage adds a new message to a room
func (c *RoomController) SendMessage(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}
	
	// Get user ID from context (set by JWT middleware)
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}
	
	message, err := c.chatService.SendMessage(roomID, userID, req.Content)
	if err != nil {
		switch err {
		case services.ErrRoomNotFound:
			http.Error(w, "Room not found", http.StatusNotFound)
		case services.ErrInvalidSenderID:
			http.Error(w, "Invalid sender ID", http.StatusBadRequest)
		default:
			http.Error(w, "Error sending message: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(message)
}
