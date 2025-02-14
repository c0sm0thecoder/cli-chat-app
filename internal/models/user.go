package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           string `gorm:"type:uuid;primary_key;"`
	UserName     string `gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Set UUID rather than numeric ID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = uuid.New().String()
	return nil
}
