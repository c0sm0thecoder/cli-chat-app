package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var ctxWS = context.Background()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	// Map to store clients by room
	roomClients = make(map[string][]*Client)
	clientMutex = &sync.Mutex{}
)

// Client represents a WebSocket client connection
type Client struct {
	conn     *websocket.Conn
	roomID   string
	userID   string
	username string
}

func GetRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable not set")
	}
	return CreateRedisChannel(redisURL), nil
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract token from URL query parameters or Authorization header
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Validate token and extract user ID
	userID, username, err := validateToken(token)
	if err != nil {
		// Use HTTP error instead of logging
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract room ID from query parameters
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		// Use HTTP error instead of logging
		http.Error(w, "Missing room_id parameter", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Server-side error, won't affect client UI
		return
	}

	// Create new client
	client := &Client{
		conn:     conn,
		roomID:   roomID,
		userID:   userID,
		username: username,
	}

	// Register client
	clientMutex.Lock()
	roomClients[roomID] = append(roomClients[roomID], client)
	clientMutex.Unlock()

	// Handle client messages
	go handleClient(client)
}

// validateToken validates the JWT token and extracts the user ID
func validateToken(tokenString string) (string, string, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(getJWTSecret()), nil
	})

	if err != nil {
		return "", "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["sub"].(string)
		if !ok {
			return "", "", jwt.ErrTokenInvalidClaims
		}
		return userID, userID, nil // Using userID as username for now
	}

	return "", "", jwt.ErrTokenInvalidClaims
}

// getJWTSecret retrieves the JWT secret key
func getJWTSecret() string {
	// In a real app, this would be fetched from environment or config
	return os.Getenv("JWT_SECRET")
}

// handleClient manages the WebSocket connection for a client
func handleClient(client *Client) {
	defer func() {
		// Remove client when connection closes
		client.conn.Close()
		removeClient(client)
	}()

	for {
		// Read message from client
		_, _, err := client.conn.ReadMessage()
		if err != nil {
			// Clean exit without logging to console
			break
		}

		// For now we're just keeping the connection alive
		// Message processing would be implemented here if needed
	}
}

// BroadcastMessage sends a message to all clients in a room
func BroadcastMessage(roomID string, message *models.Message, username string) {
	clientMutex.Lock()
	clients := roomClients[roomID]
	clientMutex.Unlock()

	// Create message payload
	payload := map[string]interface{}{
		"type":       "new_message",
		"id":         message.ID,
		"room_id":    message.RoomID,
		"sender_id":  message.SenderID,
		"username":   username,
		"content":    message.Content,
		"created_at": message.CreatedAt,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Send to all clients in the room
	for _, client := range clients {
		err := client.conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			client.conn.Close()
			removeClient(client)
		}
	}
}

// removeClient removes a client from the room
func removeClient(client *Client) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	clients := roomClients[client.roomID]
	for i, c := range clients {
		if c == client {
			roomClients[client.roomID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// Clean up empty rooms
	if len(roomClients[client.roomID]) == 0 {
		delete(roomClients, client.roomID)
	}
}
