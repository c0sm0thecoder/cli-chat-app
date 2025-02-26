package repositories

import (
	"errors"
	"fmt"

	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"gorm.io/gorm"
)

type RoomRepository interface {
	Create(room *models.Room) error
	FindByName(roomName string) (*models.Room, error)
	FindByCode(roomCode string) (*models.Room, error)
}

type roomRepo struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) RoomRepository {
	return &roomRepo{db: db}
}

func (r *roomRepo) Create(room *models.Room) error {
	if err := r.db.Create(room).Error; err != nil {
		return fmt.Errorf("failed to create room: %w", err)
	}
	return nil
}

func (r *roomRepo) FindByName(roomName string) (*models.Room, error) {
	var room models.Room
	if err := r.db.Where("name = ?", roomName).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to find room by name: %w", err)
	}
	return &room, nil
}

func (r *roomRepo) FindByCode(roomCode string) (*models.Room, error) {
	var room models.Room
	if err := r.db.Where("code = ?", roomCode).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to find room by code: %w", err)
	}
	return &room, nil
}
