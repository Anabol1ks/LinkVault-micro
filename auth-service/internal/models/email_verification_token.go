package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailVerificationToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null"`
	Token     string    `gorm:"type:varchar(255);unique"`
	ExpiresAt time.Time `gorm:"index;not null"`
	Used      bool      `gorm:"default:false;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (m *EmailVerificationToken) BeforeCreate(tx *gorm.DB) (err error) {
	m.ID = uuid.New()
	return
}
