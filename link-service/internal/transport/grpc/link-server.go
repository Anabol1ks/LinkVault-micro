package grpc

import (
	"context"
	"fmt"
	"link-service/config"
	"link-service/internal/service"
	"time"

	linkv1 "github.com/Anabol1ks/linkvault-proto/link/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"

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

	userID, ok := ctx.Value("user_id").(uuid.UUID)

	var expireAfter *time.Duration
	if ok {
		if req.ExpireAfter != "" {
			d, err := time.ParseDuration(req.ExpireAfter)
			if err != nil {
				s.shortService.Log.Warn("failed", zap.String("op", "CreateShortLink"), zap.Error(err))
				return nil, status.Errorf(codes.InvalidArgument, "invalid duration: %v", err)
			}
			expireAfter = &d
		}
	} else {
		userID = uuid.UUID{}
		expireAfter = nil
	}

	var userIDPtr *uuid.UUID
	if ok {
		userIDPtr = &userID
	} else {
		userIDPtr = nil
	}

	shortLink, err := s.shortService.CreateShortLink(req.OriginalUrl, userIDPtr, expireAfter)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "CreateShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create short link: %v", err)
	}

	shortURL := fmt.Sprintf("%s/%s", s.cfg.Domain, shortLink.ShortCode)

	var userIdValue *wrapperspb.StringValue
	if userIDPtr != nil && *userIDPtr != uuid.Nil {
		userIdValue = wrapperspb.String(userIDPtr.String())
	} else {
		userIdValue = nil
	}

	var expireAt string
	if expireAfter != nil {
		expireAt = expireAfter.String()
	} else {
		expireAt = ""
	}

	resp := &linkv1.ShortLinkResponse{
		Id:          shortLink.ID.String(),
		ShortUrl:    shortURL,
		OriginalUrl: shortLink.OriginalURL,
		ShortCode:   shortLink.ShortCode,
		UserId:      userIdValue,
		ExpireAt:    expireAt,
		IsActive:    shortLink.IsActive,
	}

	return resp, nil
}
