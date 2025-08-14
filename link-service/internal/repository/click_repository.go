package repository

import (
	"link-service/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClickRepository struct {
	db *gorm.DB
}

func NewClickRepository(db *gorm.DB) *ClickRepository {
	return &ClickRepository{
		db: db,
	}
}

func (r *ClickRepository) Create(click *models.Click) error {
	return r.db.Create(click).Error
}

func (r *ClickRepository) GetClicksByShortLinkID(shortLinkID string) ([]models.Click, error) {
	var clicks []models.Click
	err := r.db.Where("short_link_id = ?", shortLinkID).Find(&clicks).Error
	return clicks, err
}

func (c *ClickRepository) GetCount(shortLinkID string) (int64, error) {
	var count int64
	err := c.db.Model(&models.Click{}).
		Where("short_link_id = ?", shortLinkID).
		Count(&count).Error
	return count, err
}

// Количество уникальных IP
func (c *ClickRepository) GetUniqueIPCount(shortLinkID string) (int64, error) {
	var count int64
	err := c.db.Model(&models.Click{}).
		Where("short_link_id = ?", shortLinkID).
		Distinct("ip").
		Count(&count).Error
	return count, err
}

// География: количество переходов по странам
func (c *ClickRepository) GetCountryStats(shortLinkID string) (map[string]int64, error) {
	rows, err := c.db.Model(&models.Click{}).
		Select("country, COUNT(*) as cnt").
		Where("short_link_id = ?", shortLinkID).
		Group("country").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	var country string
	var cnt int64
	for rows.Next() {
		if err := rows.Scan(&country, &cnt); err != nil {
			return nil, err
		}
		stats[country] = cnt
	}
	return stats, nil
}

// График по дням: количество переходов по датам
func (c *ClickRepository) GetDailyStats(shortLinkID string) (map[string]int64, error) {
	rows, err := c.db.Model(&models.Click{}).
		Select("DATE(clicked_at) as day, COUNT(*) as cnt").
		Where("short_link_id = ?", shortLinkID).
		Group("day").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int64)
	var day string
	var cnt int64
	for rows.Next() {
		if err := rows.Scan(&day, &cnt); err != nil {
			return nil, err
		}
		stats[day] = cnt
	}
	return stats, nil
}

// Получить список уникальных IP
func (c *ClickRepository) GetUniqueIPs(shortLinkID string) ([]string, error) {
	var ips []string
	err := c.db.Model(&models.Click{}).
		Where("short_link_id = ?", shortLinkID).
		Distinct().
		Pluck("ip", &ips).Error
	return ips, err
}

// Получить список уникальных стран
func (c *ClickRepository) GetUniqueCountries(shortLinkID string) ([]string, error) {
	var countries []string
	err := c.db.Model(&models.Click{}).
		Where("short_link_id = ?", shortLinkID).
		Distinct().
		Pluck("country", &countries).Error
	return countries, err
}

// Удалить клики по short_link_id
func (r *ClickRepository) DeleteClicksByShortLinkID(id uuid.UUID) error {
	return r.db.Where("short_link_id = ?", id).Delete(&models.Click{}).Error
}
