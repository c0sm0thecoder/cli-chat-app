package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        string `gorm:"type:uuid;primary_key;"`
	RoomID    string `gorm:"type:uuid;not null;index"`
	SenderID  string `gorm:"type:uuid;not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Set UUID rather than numeric ID
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	m.ID = uuid.New().String()
	return nil
}
