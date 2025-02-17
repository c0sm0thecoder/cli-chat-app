package repositories

import (
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
	return r.db.Create(message).Error
}

func (r *messageRepo) FindByRoom(roomID string) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("roomID = ?", roomID).Find(&messages).Error
	return messages, err
}
