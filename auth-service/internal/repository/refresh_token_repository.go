package repository

import (
	"time"

	"linkv-auth/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(rt *models.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *RefreshTokenRepository) FindValid(jti string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	if err := r.db.Where("jti = ? AND revoked = false AND expires_at > ?", jti, time.Now()).First(&rt).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) RevokeByJTI(jti string) error {
	return r.db.Model(&models.RefreshToken{}).Where("jti = ?", jti).Update("revoked", true).Error
}

func (r *RefreshTokenRepository) RevokeAllForUser(userID uuid.UUID) error {
	return r.db.Model(&models.RefreshToken{}).Where("user_id = ? AND revoked = false", userID).Update("revoked", true).Error
}

// DeleteExpired удаляет все просроченные refresh токены (независимо от revoked) и возвращает число удалённых строк.
func (r *RefreshTokenRepository) DeleteExpired(now time.Time) (int64, error) {
	res := r.db.Where("expires_at <= ? OR revoked = true", now).Delete(&models.RefreshToken{})
	return res.RowsAffected, res.Error
}
