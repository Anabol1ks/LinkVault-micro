package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Click struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	ShortLinkID uuid.UUID `gorm:"type:uuid;not null"`
	ShortLink   ShortLink `gorm:"foreignKey:ShortLinkID"`

	IP        string    `gorm:"type:text;not null"`
	UserAgent string    `gorm:"type:text;not null"`
	Country   string    `gorm:"type:text;not null"`
	Region    string    `gorm:"type:text;not null"`
	ClickedAt time.Time `gorm:"autoCreateTime"`
}

func (m *Click) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New()
	return
}
