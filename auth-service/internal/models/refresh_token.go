package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	JTI       string    `gorm:"type:uuid;uniqueIndex;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	Revoked   bool      `gorm:"default:false;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (m *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
