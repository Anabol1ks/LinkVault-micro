package repository

import (
	"link-service/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShortLinkRepository struct {
	db *gorm.DB
}

func NewShortLinkRepository(db *gorm.DB) *ShortLinkRepository {
	return &ShortLinkRepository{
		db: db,
	}
}

func (r *ShortLinkRepository) Create(shortLink *models.ShortLink) error {
	return r.db.Create(shortLink).Error
}

func (r *ShortLinkRepository) GetByShortCode(shortLink *models.ShortLink, shortCode string) error {
	return r.db.Where("short_code = ? AND is_active = ? AND (expire_at IS NULL OR expire_at > ?)", shortCode, true, time.Now()).First(shortLink).Error
}

func (r *ShortLinkRepository) GetByUserID(userID uuid.UUID) ([]*models.ShortLink, error) {
	var shortLinks []*models.ShortLink
	if err := r.db.Where("user_id = ? AND is_active = ? AND (expire_at IS NULL OR expire_at > ?)", userID, true, time.Now()).Find(&shortLinks).Error; err != nil {
		return nil, err
	}
	return shortLinks, nil
}

func (r *ShortLinkRepository) DeactivateByID(id, userID uuid.UUID) error {
	return r.db.Model(&models.ShortLink{}).
		Where("id = ? AND user_id = ? AND is_active = ?", id, userID, true).
		Update("is_active", false).Error
}

func (r *ShortLinkRepository) GetByID(id string) (*models.ShortLink, error) {
	var shortLink models.ShortLink
	if err := r.db.Where("id = ? AND is_active = ? AND (expire_at IS NULL OR expire_at > ?)", id, true, time.Now()).First(&shortLink).Error; err != nil {
		return nil, err
	}
	return &shortLink, nil
}
