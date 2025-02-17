package realtime

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var ctxWS = context.Background()

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func GetRedisClient() *redis.Client {
	redisUrl := os.Getenv("REDIS_URL")
	return CreateRedisChannel(redisUrl)
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomCode := r.URL.Query().Get("room")
	if roomCode == "" {
		http.Error(w, "Error parsing room code", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP request to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Websocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Create and subscribe to a Redis channel for this room
	redisClient := GetRedisClient()
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
					log.Println("Error writing message to Websocket")
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message from Websocket:", err)
			close(done)
			break
		}
		if messageType != websocket.TextMessage {
			continue
		}

		err = redisClient.Publish(ctxWS, channelName, message).Err()
		if err != nil {
			log.Println("Error publishing message to Redis:", err)
			close(done)
			break
		}
	}

	wg.Wait()
}
