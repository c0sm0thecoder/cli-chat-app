package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	RoomID    string    `gorm:"type:uuid;not null;index"`
	SenderID  string    `gorm:"type:varchar(255);not null;index"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeCreate will set a UUID rather than numeric ID
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		// Generate a new UUID and convert to string
		m.ID = uuid.New().String()
	}
	return nil
}
