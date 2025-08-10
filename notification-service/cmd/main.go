package main

import (
	"notification-service/config"
	"notification-service/internal/sender"
	"notification-service/pkg/logger"
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

	log := logger.L()

	cfg := config.Load(log)

	emailSender := sender.NewEmailSender(cfg)
	_ = emailSender
}
