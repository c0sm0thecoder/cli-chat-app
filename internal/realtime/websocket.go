package realtime

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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

func GetRedisClient() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable not set")
	}
	return CreateRedisChannel(redisURL), nil
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	if roomCode == "" {
		http.Error(w, "Room code is required", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP request to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade connection to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Set ping handler to keep connection alive
	conn.SetPingHandler(func(appData string) error {
		err := conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
		if err == websocket.ErrCloseSent {
			return nil
		}
		return err
	})

	// Create and subscribe to a Redis channel for this room
	redisClient, err := GetRedisClient()
	if err != nil {
		log.Printf("Redis client error: %v", err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Server error"))
		return
	}

	channelName := "room:" + roomCode
	pubsub := redisClient.Subscribe(ctxWS, channelName)
	defer pubsub.Close()

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Read the messages from Redis and write to Websocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		ch := pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					log.Printf("Error writing message to WebSocket: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Send ping messages periodically to keep the connection alive
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Ping error: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Process incoming WebSocket messages
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			close(done)
			break
		}
		
		if messageType != websocket.TextMessage {
			continue
		}

		// Publish message to Redis
		err = redisClient.Publish(ctxWS, channelName, message).Err()
		if err != nil {
			log.Printf("Redis publish error: %v", err)
			close(done)
			break
		}
	}

	wg.Wait()
}
