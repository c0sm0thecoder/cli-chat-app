package repositories

import (
	"errors"
	"fmt"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"gorm.io/gorm"
)

type MessageRepository interface {
	Create(message *models.Message) error
	FindByRoom(roomID string) ([]models.Message, error)
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(message *models.Message) error {
	if err := r.db.Create(message).Error; err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	return nil
}

func (r *messageRepo) FindByRoom(roomID string) ([]models.Message, error) {
	var messages []models.Message
	if err := r.db.Where("room_id = ?", roomID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to find messages for room: %w", err)
	}
	return messages, nil
}
