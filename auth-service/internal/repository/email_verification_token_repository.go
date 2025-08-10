package repository

import (
	"linkv-auth/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailVerificationTokenRepository struct {
	db *gorm.DB
}

func NewEmailVerificationTokenRepository(db *gorm.DB) *EmailVerificationTokenRepository {
	return &EmailVerificationTokenRepository{db: db}
}

func (r *EmailVerificationTokenRepository) Create(token *models.EmailVerificationToken) error {
	return r.db.Create(token).Error
}

func (r *EmailVerificationTokenRepository) FindByToken(token string) (*models.EmailVerificationToken, error) {
	var evt models.EmailVerificationToken
	if err := r.db.Where("token = ? AND used = false AND expires_at > ?", token, time.Now()).First(&evt).Error; err != nil {
		return nil, err
	}
	return &evt, nil
}

func (r *EmailVerificationTokenRepository) MarkUsed(tokenID uuid.UUID) error {
	return r.db.Model(&models.EmailVerificationToken{}).Where("id = ?", tokenID).Update("used", true).Error
}

func (r *EmailVerificationTokenRepository) DeleteExpired(now time.Time) (int64, error) {
	res := r.db.Where("expires_at <= ? OR used = true", now).Delete(&models.EmailVerificationToken{})
	return res.RowsAffected, res.Error
}
