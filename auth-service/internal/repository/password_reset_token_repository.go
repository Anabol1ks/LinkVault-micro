package repository

import (
	"linkv-auth/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PasswordResetTokenRepository struct {
	db *gorm.DB
}

func NewPasswordResetTokenRepository(db *gorm.DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

func (r *PasswordResetTokenRepository) Create(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *PasswordResetTokenRepository) FindByToken(token string) (*models.PasswordResetToken, error) {
	var prt models.PasswordResetToken
	if err := r.db.Where("token = ? AND used = false AND expires_at > ?", token, time.Now()).First(&prt).Error; err != nil {
		return nil, err
	}
	return &prt, nil
}

func (r *PasswordResetTokenRepository) MarkUsed(tokenID uuid.UUID) error {
	return r.db.Model(&models.PasswordResetToken{}).Where("id = ?", tokenID).Update("used", true).Error
}

func (r *PasswordResetTokenRepository) DeleteExpired(now time.Time) (int64, error) {
	res := r.db.Where("expires_at <= ? OR used = true", now).Delete(&models.PasswordResetToken{})
	return res.RowsAffected, res.Error
}

func (r *PasswordResetTokenRepository) FindLatestByUser(userID uuid.UUID) (*models.PasswordResetToken, error) {
	var prt models.PasswordResetToken
	if err := r.db.Where("user_id = ?", userID).Order("created_at desc").First(&prt).Error; err != nil {
		return nil, err
	}
	return &prt, nil
}
