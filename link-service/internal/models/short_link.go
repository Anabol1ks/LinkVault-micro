package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShortLink struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID      *uuid.UUID `gorm:"type:uuid"`
	OriginalURL string     `gorm:"type:text;not null"`
	ShortCode   string     `gorm:"type:text;unique;not null"`
	IsActive    bool       `gorm:"not null"`
	ExpireAt    *time.Time
	CreatedAt   time.Time `gorm:"autoCreateTime"`

	Clicks []Click `gorm:"foreignKey:ShortLinkID"`
}

func (m *ShortLink) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New()
	return
}
