package grpc

import (
	"context"
	"fmt"
	linkv1 "link-service/api/proto/link/v1"
	"link-service/config"
	"link-service/internal/service"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LinkServer struct {
	linkv1.UnimplementedLinkServiceServer
	shortService *service.ShortLinkService
	cfg          *config.Config
}

func NewLinkServer(shortService *service.ShortLinkService, cfg *config.Config) *LinkServer {
	return &LinkServer{
		shortService: shortService,
		cfg:          cfg,
	}
}

func (s *LinkServer) CreateShortLink(ctx context.Context, req *linkv1.CreateShortLinkRequest) (*linkv1.ShortLinkResponse, error) {
	s.shortService.Log.Info("start", zap.String("op", "CreateShortLink"))
	if err := req.Validate(); err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "CreateShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	userID, _ := ctx.Value("user_id").(uuid.UUID)

	var expireAfter *time.Duration
	if req.ExpireAfter != "" {
		d, err := time.ParseDuration(req.ExpireAfter)
		if err != nil {
			s.shortService.Log.Warn("failed", zap.String("op", "CreateShortLink"), zap.Error(err))
			return nil, status.Errorf(codes.InvalidArgument, "invalid duration: %v", err)
		}
		expireAfter = &d
	}

	shortLink, err := s.shortService.CreateShortLink(req.OriginalUrl, &userID, expireAfter)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "CreateShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create short link: %v", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.cfg.Domain, shortLink.ShortCode)

	resp := &linkv1.ShortLinkResponse{
		Id:          shortLink.ID.String(),
		ShortUrl:    shortURL,
		OriginalUrl: shortLink.OriginalURL,
		ShortCode:   shortLink.ShortCode,
		UserId:      userID.String(),
		ExpireAt:    expireAfter.String(),
		IsActive:    shortLink.IsActive,
	}

	return resp, nil
}
