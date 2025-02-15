package services

import (
	"errors"
	"time"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
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

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return &authService{userRepo: userRepo, jwtSecret: jwtSecret}
}

func (s *authService) SignUp(username, password string) error {
	// Hash the entered password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		UserName:     username,
		PasswordHash: string(hash),
	}

	return s.userRepo.Create(user)
}

func (s *authService) Login(username, password string) (string, error) {
	// Look up if the user exists
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Check if the password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("incorrect credentials")
	}

	// Create a JWT token valid for 24 hours
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.UserName,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	// Return the generated JWT token string
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *authService) ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
}
