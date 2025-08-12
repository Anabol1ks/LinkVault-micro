package repository

import (
	"auth-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) MarkEmailVerified(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("email_verified", true).Error
}

// UpdatePassword обновляет хеш пароля пользователя по ID
func (r *UserRepository) UpdatePassword(user *models.User) error {
	return r.db.Model(&models.User{}).Where("id = ?", user.ID).Update("password_hash", user.PasswordHash).Error
}
