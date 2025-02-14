package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	ID        string `gorm:"type:uuid;primary_key;"`
	Name      string `gorm:"size:100;not null"`
	Code      string `gorm:"size:10;uniqueIndex;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Set UUID rather than numeric ID
func (r *Room) BeforeCreate(tx *gorm.DB) error {
	r.ID = uuid.New().String()
	return nil
}
