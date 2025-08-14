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
		Updates(map[string]interface{}{"is_active": false, "deactivated_at": time.Now()}).Error
}

func (r *ShortLinkRepository) GetShortLinkByID(id string, userID uuid.UUID) (*models.ShortLink, error) {
	var shortLink models.ShortLink
	if err := r.db.Where("id = ? AND user_id = ? AND is_active = ? AND (expire_at IS NULL OR expire_at > ?)", id, userID, true, time.Now()).First(&shortLink).Error; err != nil {
		return nil, err
	}
	return &shortLink, nil
}

func (r *ShortLinkRepository) FindExpiredAnonLinks() ([]*models.ShortLink, error) {
	var links []*models.ShortLink
	err := r.db.Where("user_id IS NULL AND expire_at IS NOT NULL AND expire_at < ?", time.Now()).Find(&links).Error
	return links, err
}

func (r *ShortLinkRepository) FindExpiredInactiveAnonLinks() ([]*models.ShortLink, error) {
	var links []*models.ShortLink
	err := r.db.Where("user_id IS NULL AND is_active = false AND expire_at IS NOT NULL AND expire_at < ?", time.Now()).Find(&links).Error
	return links, err
}

func (r *ShortLinkRepository) FindExpiredInactiveUserLinks() ([]*models.ShortLink, error) {
	var links []*models.ShortLink
	weekAgo := time.Now().Add(-7 * 24 * time.Hour)
	err := r.db.Where(`user_id IS NOT NULL AND is_active = false AND expire_at IS NOT NULL AND expire_at < ? AND deactivated_at IS NOT NULL AND deactivated_at < ?`, time.Now(), weekAgo).Find(&links).Error
	return links, err
}

func (r *ShortLinkRepository) DeleteLink(link *models.ShortLink) error {
	return r.db.Delete(link).Error
}

func (r *ShortLinkRepository) DeactivateExpiredAnonLinks() error {
	return r.db.Model(&models.ShortLink{}).
		Where("user_id IS NULL AND is_active = true AND expire_at IS NOT NULL AND expire_at < ?", time.Now()).
		Update("is_active", false).Error
}

func (r *ShortLinkRepository) DeactivateExpiredUserLinks() error {
	return r.db.Model(&models.ShortLink{}).
		Where("user_id IS NOT NULL AND is_active = true AND expire_at IS NOT NULL AND expire_at < ?", time.Now()).
		Update("is_active", false).Error
}
