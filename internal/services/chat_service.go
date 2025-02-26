package services

import (
	"errors"
	"fmt"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/realtime"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type ChatService interface {
	CreateRoom(roomName string) (*models.Room, error)
	SendMessage(roomID, senderID, messageContent string) (*models.Message, error)
	GetMessages(roomID string) ([]models.Message, error)
	GetRoomByCode(roomCode string) (*models.Room, error)
}

type chatService struct {
	roomRepo    repositories.RoomRepository
	messageRepo repositories.MessageRepository
	userRepo    repositories.UserRepository
}

var (
	ErrEmptyRoomName     = errors.New("room name cannot be empty")
	ErrEmptyMessage      = errors.New("message content cannot be empty")
	ErrRoomNotFound      = errors.New("room not found")
	ErrInvalidSenderID   = errors.New("invalid sender ID")
)

func NewChatService(roomRepo repositories.RoomRepository, messageRepo repositories.MessageRepository, userRepo repositories.UserRepository) ChatService {
	return &chatService{
		roomRepo:    roomRepo,
		messageRepo: messageRepo,
		userRepo:    userRepo,
	}
}

func (s *chatService) CreateRoom(roomName string) (*models.Room, error) {
	if roomName == "" {
		return nil, ErrEmptyRoomName
	}
	
	// Generate a unique room code
	roomCode, err := gonanoid.New(12)
	if err != nil {
		return nil, fmt.Errorf("failed to generate room code: %w", err)
	}
	
	room := &models.Room{
		Name: roomName,
		Code: roomCode,
	}
	
	if err := s.roomRepo.Create(room); err != nil {
		return nil, err
	}
	
	return room, nil
}

func (s *chatService) SendMessage(roomID, senderID, messageContent string) (*models.Message, error) {
	if messageContent == "" {
		return nil, ErrEmptyMessage
	}
	
	if senderID == "" {
		return nil, ErrInvalidSenderID
	}
	
	message := &models.Message{
		RoomID:   roomID,
		SenderID: senderID,
		Content:  messageContent,
	}
	
	if err := s.messageRepo.Create(message); err != nil {
		return nil, err
	}
	
	// Get user info for the sender (for display name)
	user, err := s.userRepo.FindByUsername(senderID)
	
	// Get username to display
	username := senderID // Default to ID if user can't be found
	if err == nil && user != nil {
		username = user.UserName
	}
	
	// Broadcast the message to all WebSocket clients in this room
	go realtime.BroadcastMessage(roomID, message, username)
	
	return message, nil
}

func (s *chatService) GetMessages(roomID string) ([]models.Message, error) {
	messages, err := s.messageRepo.FindByRoom(roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages: %w", err)
	}
	return messages, nil
}

func (s *chatService) GetRoomByCode(roomCode string) (*models.Room, error) {
	room, err := s.roomRepo.FindByCode(roomCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return room, nil
}
