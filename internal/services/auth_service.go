package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	SignUp(username, password string) error
	Login(username, password string) (string, error)
	ParseToken(tokenString string) (*jwt.Token, error)
}

type authService struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserAlreadyExists = errors.New("username already exists")
	ErrTokenCreation     = errors.New("failed to create authentication token")
)

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return &authService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *authService) SignUp(username, password string) error {
	// Check if user already exists
	_, err := s.userRepo.FindByUsername(username)
	if err == nil {
		return ErrUserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("database error: %w", err)
	}

	// Hash the entered password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("password hashing error: %w", err)
	}

	user := &models.User{
		UserName:     username,
		PasswordHash: string(hash),
	}

	if err := s.userRepo.Create(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (s *authService) Login(username, password string) (string, error) {
	// Look up if the user exists
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// Check if the password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	// Create a JWT token valid for 24 hours
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.UserName,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(), // Issued at time
	})

	// Return the generated JWT token string
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", ErrTokenCreation
	}

	return tokenString, nil
}

func (s *authService) ParseToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	return token, nil
}
