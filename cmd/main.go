package main

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/c0sm0thecoder/cli-chat-app/internal/controllers"
	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/c0sm0thecoder/cli-chat-app/internal/ui"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// main.go
func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize repositories
	db := initDB()
	userRepo := repositories.NewUserRepository(db)
	roomRepo := repositories.NewRoomRepository(db)
	messageRepo := repositories.NewMessageRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, os.Getenv("JWT_SECRET"))
	chatService := services.NewChatService(roomRepo, messageRepo)

	// Create router
	router := controllers.NewV1Router(authService, chatService)

	// Start HTTP server in a goroutine
	serverReady := make(chan struct{})
	go func() {
		addr := ":8080"
		log.Printf("Attempting to start server on %s", addr)

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			log.Printf("Failed to create listener: %v", err)
			log.Fatal(err)
		}
		log.Printf("Listener created successfully")

		close(serverReady)

		log.Printf("Server starting...")
		if err := http.Serve(listener, router); err != nil {
			log.Printf("Server error: %v", err)
			log.Fatal(err)
		}
	}()

	<-serverReady
	log.Printf("Server ready signal received")

	log.Printf("Starting CLI...")
	if err := ui.StartCLI(); err != nil {
		log.Printf("CLI error: %v", err)
		log.Fatal(err)
	}
}

func initDB() *gorm.DB {
	db, err := gorm.Open(postgres.Open(os.Getenv("PG_URL")), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.AutoMigrate(&models.User{}, &models.Room{}, &models.Message{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	return db
}
