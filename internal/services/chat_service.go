package services

import (
	"errors"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"github.com/c0sm0thecoder/cli-chat-app/internal/repositories"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ChatService interface {
	CreateRoom(roomName string) error
	SendMessage(roomID, senderID, messageContent string) error
	GetMessages(roomID string) ([]models.Message, error)
}

type chatService struct {
	roomRepo    repositories.RoomRepository
	messageRepo repositories.MessageRepository
}

func NewChatService(roomRepo repositories.RoomRepository, messageRepo repositories.MessageRepository) ChatService {
	return &chatService{
		roomRepo:    roomRepo,
		messageRepo: messageRepo,
	}
}

func (s *chatService) CreateRoom(roomName string) error {
	roomCode, err := gonanoid.New(12)
	if err != nil {
		return err
	}
	room := &models.Room{
		Name: roomName,
		Code: roomCode,
	}
	return s.roomRepo.Create(room)
}

func (s *chatService) SendMessage(roomID, senderID, messageContent string) error {
	if messageContent == "" {
		return errors.New("message body can't be empty")
	}
	message := &models.Message{
		RoomID:   roomID,
		SenderID: senderID,
		Content:  messageContent,
	}
	return s.messageRepo.Create(message)
}

func (s *chatService) GetMessages(roomID string) ([]models.Message, error) {
	return s.messageRepo.FindByRoom(roomID)
}
