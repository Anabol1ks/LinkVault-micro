package main

import (
	"context"
	"linkv-auth/config"
	_ "linkv-auth/docs"
	"linkv-auth/internal/handler"
	"linkv-auth/internal/maintenance"
	"linkv-auth/internal/repository"
	"linkv-auth/internal/router"
	"linkv-auth/internal/service"
	"linkv-auth/internal/storage"
	"linkv-auth/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// @Title						LinkVault AuthService API
// @Version					1.0
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @host      localhost:8081
// @BasePath  /api/v1
func main() {
	_ = godotenv.Load()
	isDev := os.Getenv("ENV") == "development"
	if err := logger.Init(isDev); err != nil {
		panic(err)
	}

	defer logger.Sync()

	log := logger.L()

	cfg := config.Load(log)

	db := storage.ConnectDB(&cfg.DB, log)
	if db == nil {
		log.Fatal("Не удалось подключиться к базе данных")
	}

	storage.Migrate(db, log)

	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	userService := service.NewUserService(userRepo, refreshTokenRepo, log, cfg)
	userHandler := handler.NewUserHandler(userService)

	scheduler := maintenance.NewScheduler(log, refreshTokenRepo)
	appCtx, cancelScheduler := context.WithCancel(context.Background())
	if err := scheduler.Start(appCtx); err != nil {
		log.Error("Не удалось запустить планировщик", zap.Error(err))
	}

	r := router.Router(db, log, userHandler, cfg)
	srv := &http.Server{
		Addr:    cfg.Port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server start failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cancelScheduler()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	storage.CloseDB(db, log)

	log.Info("Server exiting")
}
