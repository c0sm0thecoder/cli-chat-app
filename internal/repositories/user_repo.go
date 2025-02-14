package repositories

import (
	"github.com/c0sm0thecoder/cli-chat-app/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	FindByUsername(username string) (*models.User, error)
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepo) FindByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}
