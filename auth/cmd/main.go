package main

import (
	"linkv-auth/config"
	"linkv-auth/internal/storage"
	"linkv-auth/pkg/logger"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	isDev := os.Getenv("ENV") == "development"
	if err := logger.Init(isDev); err != nil {
		panic(err)
	}
	defer logger.Sync()

	defer logger.Sync()

	log := logger.L()

	cfg := config.Load(log)

	db := storage.ConnectDB(&cfg.DB, log)
	if db == nil {
		log.Fatal("Не удалось подключиться к базе данных")
	}

	storage.Migrate(db, log)
}
