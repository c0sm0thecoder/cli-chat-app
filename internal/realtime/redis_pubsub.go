package realtime

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var ctxRedis = context.Background()

func CreateRedisChannel(redisUrl string) *redis.Client {
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Fatalf("Failed to parse redis URL: %v", err)
	}
	client := redis.NewClient(opt)
	return client
}

func PublishMessage(client *redis.Client, channel, message string) error {
	err := client.Publish(ctxRedis, channel, message)
	if err != nil {
		log.Fatalf("Error publishing message to Redis: %v", err)
	}
	return nil
}

func SubscribeChannel(client *redis.Client, channel string, messageHandler func(message string)) {
	pubsub := client.Subscribe(ctxRedis, channel)
	_, err := pubsub.Receive(ctxRedis)
	if err != nil {
		log.Printf("Failed to subscribe to channel %s: %v", channel, err)
		return
	}
	ch := pubsub.Channel()
	go func() {
		for msg := range ch {
			messageHandler(msg.Payload)
		}
	}()
}
