package main

import (
	"linkv-auth/config"
	"linkv-auth/internal/handler"
	"linkv-auth/internal/repository"
	"linkv-auth/internal/router"
	"linkv-auth/internal/service"
	"linkv-auth/internal/storage"
	"linkv-auth/pkg/logger"
	"os"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
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

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo, log, cfg)
	userHandler := handler.NewUserHandler(userService)

	r := router.Router(db, log, userHandler, cfg)
	if err := r.Run(cfg.Port); err != nil {
		log.Fatal("Не удалось запустить сервер", zap.Error(err))
	}
}
