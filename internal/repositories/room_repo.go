package repositories

import (
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
	return r.db.Create(room).Error
}

func (r *roomRepo) FindByName(roomName string) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("name = ?", roomName).First(&room).Error
	return &room, err
}

func (r *roomRepo) FindByCode(roomCode string) (*models.Room, error) {
	var room models.Room
	err := r.db.Where("code = ?", roomCode).First(&room).Error
	return &room, err
}
