package storage

import (
	"linkv-auth/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB, log *zap.Logger) {
	if err := db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
	); err != nil {
		log.Fatal("Не удалось выполнить миграцию базы данных", zap.Error(err))
	}
	log.Info("Миграция базы данных успешно выполнена")
}
