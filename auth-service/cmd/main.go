package main

import (
	"context"
	"linkv-auth/config"
	_ "linkv-auth/docs"
	"linkv-auth/internal/maintenance"
	"linkv-auth/internal/repository"
	"linkv-auth/internal/service"
	"linkv-auth/internal/storage"
	"linkv-auth/pkg/logger"
	"net"
	"os"
	"os/signal"
	"syscall"

	authv1 "linkv-auth/api/proto/auth/v1"
	grpcserver "linkv-auth/internal/transport/grpc"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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
	emailTokenRepo := repository.NewEmailVerificationTokenRepository(db)
	passwordResetRepo := repository.NewPasswordResetTokenRepository(db)

	userService := service.NewUserService(userRepo, refreshTokenRepo, emailTokenRepo, passwordResetRepo, cfg)

	scheduler := maintenance.NewScheduler(log, refreshTokenRepo, emailTokenRepo, passwordResetRepo)
	appCtx, cancelScheduler := context.WithCancel(context.Background())
	if err := scheduler.Start(appCtx); err != nil {
		log.Error("Не удалось запустить планировщик", zap.Error(err))
	}

	// gRPC server setup
	lis, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.AuthInterceptor(&cfg.JWT)),
	)

	// gRPC health check service
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	authv1.RegisterAuthServiceServer(grpcServer, grpcserver.NewAuthServer(userService, log))

	go func() {
		log.Info("Starting gRPC server", zap.String("addr", cfg.Port))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down gRPC server...")

	grpcServer.GracefulStop()
	cancelScheduler()
	storage.CloseDB(db, log)
	log.Info("Server exiting")
}
