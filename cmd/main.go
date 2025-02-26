package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/c0sm0thecoder/cli-chat-app/internal/controllers"
	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/realtime"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	"github.com/c0sm0thecoder/cli-chat-app/internal/services"
	"github.com/c0sm0thecoder/cli-chat-app/internal/ui"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Parse command line flags
	var cliMode bool
	var serverURL string
	
	flag.BoolVar(&cliMode, "cli", false, "Run in CLI mode")
	flag.StringVar(&serverURL, "server", "", "Server URL (default: http://localhost:8080/api/v1)")
	flag.Parse()

	// Add debug output
	log.Printf("Starting application. CLI mode: %v, Server URL: %s", cliMode, serverURL)

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using environment variables.")
	}

	// Set default JWT secret if not provided
	if os.Getenv("JWT_SECRET") == "" {
		log.Println("Warning: JWT_SECRET not set. Using default value for development.")
		os.Setenv("JWT_SECRET", "your-secret-key-replace-in-production")
	}

	// If CLI mode is enabled, start the CLI interface
	if cliMode {
		if serverURL == "" {
			serverURL = "http://localhost:8080/api/v1"
		}
		log.Printf("Starting CLI with server URL: %s", serverURL)
		ui.StartCLI(serverURL)
		return
	}

	// Connect to database (only needed for server mode)
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	roomRepo := repositories.NewRoomRepository(db)
	messageRepo := repositories.NewMessageRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, os.Getenv("JWT_SECRET"))
	chatService := services.NewChatService(roomRepo, messageRepo, userRepo)

	// Start the HTTP server
	router := chi.NewRouter()

	// API routes
	apiRouter := controllers.NewV1Router(authService, chatService)
	router.Mount("/api/v1", apiRouter)

	// WebSocket handler
	router.Get("/api/v1/ws", realtime.HandleWebSocket)

	// Static file server for web client (if exists)
	fs := http.FileServer(http.Dir("./web/dist"))
	router.Handle("/*", http.StripPrefix("/", fs))

	// Configure the HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Server listening on http://localhost%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

// setupDatabase initializes the database connection and performs migrations
func setupDatabase() (*gorm.DB, error) {
	// Read PostgreSQL connection string from environment variables
	pgURL := os.Getenv("PG_URL")
	
	if pgURL == "" {
		// Fallback to individual connection parameters if PG_URL is not defined
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbUser := os.Getenv("DB_USER")
		dbPassword := os.Getenv("DB_PASSWORD")
		dbName := os.Getenv("DB_NAME")
		dbSSLMode := os.Getenv("DB_SSLMODE")
		
		// Set defaults if not provided
		if dbHost == "" {
			dbHost = "localhost"
		}
		if dbPort == "" {
			dbPort = "5432"
		}
		if dbSSLMode == "" {
			dbSSLMode = "disable" // For development
		}
		
		// Construct the PostgreSQL connection string
		pgURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", 
			dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)
	}
	
	// Log the database connection (mask password for security)
	log.Printf("Connecting to PostgreSQL database with connection string: %s", maskPassword(pgURL))
	
	// Configure GORM with detailed logging
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}
	
	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(pgURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	
	// Get the underlying SQL DB to set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	
	// Auto-migrate the database schema
	log.Println("Running database migrations...")
	err = db.AutoMigrate(&models.User{}, &models.Room{}, &models.Message{})
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	log.Println("Database migrations completed successfully")
	
	return db, nil
}

// maskPassword replaces the password in a connection string with ****
func maskPassword(connectionString string) string {
	// This is a simple implementation - you might want to use regex for a more robust version
	parts := strings.Split(connectionString, ":")
	if len(parts) >= 3 {
		// Find the password part and mask it
		passwordEndIdx := strings.Index(parts[2], "@")
		if passwordEndIdx > 0 {
			parts[2] = "****" + parts[2][passwordEndIdx:]
		}
	}
	return strings.Join(parts, ":")
}
