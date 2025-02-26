package services

import (
	"errors"
	"log"
	"time"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	SignUp(username, password string) error
	Login(username, password string) (string, error)
}

type authService struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *authService) SignUp(username, password string) error {
	// Check if user already exists
	existingUser, err := s.userRepo.FindByUsername(username)
	if err == nil && existingUser != nil {
		return ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Create user
	user := &models.User{
		UserName: username,
		PasswordHash: string(hashedPassword), // Assuming the field is PasswordHash
	}

	return s.userRepo.Create(user)
}

func (s *authService) Login(username, password string) (string, error) {
	log.Printf("Attempting login for user: %s", username)
	
	// Find user by username
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		log.Printf("Login error - user not found: %s", username)
		return "", ErrUserNotFound
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Login error - invalid password for user: %s", username)
		return "", ErrInvalidCredentials
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.UserName, // Use username as subject
		"exp": time.Now().Add(24 * time.Hour).Unix(), // Token expires in 24 hours
		"iat": time.Now().Unix(),
	})

	// Sign token with secret key
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		log.Printf("Error signing JWT token: %v", err)
		return "", err
	}

	log.Printf("Login successful for user: %s, token created", username)
	return tokenString, nil
}
