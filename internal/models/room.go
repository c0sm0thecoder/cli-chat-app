package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:100;not null"`
	Code      string    `gorm:"size:12;uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BeforeCreate will set a UUID rather than numeric ID
func (r *Room) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		// Generate a new UUID and convert to string
		r.ID = uuid.New().String()
	}
	return nil
}
