package main

import (
	"link-service/config"
	"link-service/internal/repository"
	"link-service/internal/service"
	"link-service/internal/storage"
	grpcserver "link-service/internal/transport/grpc"
	"link-service/pkg/logger"
	"net"
	"os"
	"os/signal"
	"syscall"

	authv1 "github.com/Anabol1ks/linkvault-proto/auth/v1"
	linkv1 "github.com/Anabol1ks/linkvault-proto/link/v1"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
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

	db := storage.ConnectDB(&cfg.DB, log)
	if db == nil {
		log.Fatal("Не удалось подключиться к базе данных")
	}

	storage.Migrate(db, log)

	// подключение к auth-service
	authClient, authConn, err := createAuthClient(cfg.AuthAddr)
	if err != nil {
		log.Fatal("auth connect error", zap.Error(err))
	}
	log.Info("auth connected")
	defer authConn.Close()

	shortLinkRepo := repository.NewShortLinkRepository(db)
	shortLinkService := service.NewShortLinkService(shortLinkRepo, log)

	clickRepo := repository.NewClickRepository(db)
	clickService := service.NewClickService(clickRepo, log)

	lis, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpcserver.AuthInterceptor(authClient),
				grpcserver.OptionalAuthInterceptor(authClient),
			),
		),
	)

	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(grpcServer)

	linkv1.RegisterLinkServiceServer(grpcServer, grpcserver.NewLinkServer(shortLinkService, clickService, cfg))

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
	storage.CloseDB(db, log)
	log.Info("Server exiting")
}

func createAuthClient(authAddr string) (authv1.AuthServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(authAddr, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	client := authv1.NewAuthServiceClient(conn)
	return client, conn, nil
}
