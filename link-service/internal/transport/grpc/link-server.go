package grpc

import (
	"context"
	"errors"
	"fmt"
	"link-service/config"
	"link-service/internal/service"
	"sync"
	"time"

	linkv1 "github.com/Anabol1ks/linkvault-proto/link/v1"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type LinkServer struct {
	linkv1.UnimplementedLinkServiceServer
	shortService *service.ShortLinkService
	clickService *service.ClickService
	cfg          *config.Config
	wg           *sync.WaitGroup
}

func NewLinkServer(shortService *service.ShortLinkService, clickService *service.ClickService, cfg *config.Config, wg *sync.WaitGroup) *LinkServer {
	return &LinkServer{
		shortService: shortService,
		clickService: clickService,
		cfg:          cfg,
		wg:           wg,
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

func (s *LinkServer) RedirectLink(ctx context.Context, req *linkv1.RedirectLinkRequest) (*linkv1.RedirectLinkResponse, error) {
	s.shortService.Log.Info("start", zap.String("op", "RedirectLink"))
	if err := req.Validate(); err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "RedirectLink"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	shortLink, err := s.shortService.GetLinkByCode(req.ShortCode)
	if err != nil {
		return nil, status.Error(codes.NotFound, "short link not found")
	}
	originalURL := shortLink.OriginalURL

	var ip, userAgent string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-forwarded-for"); len(vals) > 0 {
			ip = vals[0]
		}
		if vals := md.Get("user-agent"); len(vals) > 0 {
			userAgent = vals[0]
		}
	}
	if shortLink.UserID != nil && s.wg != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			_ = s.clickService.CreateClick(shortLink.ID, ip, userAgent)
		}()
	}

	return &linkv1.RedirectLinkResponse{
		OriginalUrl: originalURL,
	}, nil
}

func (s *LinkServer) ListShortLinks(ctx context.Context, _ *emptypb.Empty) (*linkv1.ListShortLinksResponse, error) {
	s.shortService.Log.Info("start", zap.String("op", "ListShortLinks"))

	userID, ok := ctx.Value("user_id").(uuid.UUID)

	if !ok {
		s.shortService.Log.Warn("failed", zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}

	shortLinks, err := s.shortService.GetLinksUser(userID)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "ListShortLinks"), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list short links: %v", err)
	}

	var resp *linkv1.ListShortLinksResponse
	if shortLinks != nil {
		resp = &linkv1.ListShortLinksResponse{
			Links: make([]*linkv1.ShortLinkResponse, 0, len(shortLinks)),
		}
		for _, link := range shortLinks {
			resp.Links = append(resp.Links, &linkv1.ShortLinkResponse{
				Id:          link.ID.String(),
				ShortUrl:    fmt.Sprintf("%s/%s", s.cfg.Domain, link.ShortCode),
				OriginalUrl: link.OriginalURL,
				ShortCode:   link.ShortCode,
				UserId:      wrapperspb.String(link.UserID.String()),
				ExpireAt:    link.ExpireAt.String(),
				IsActive:    link.IsActive,
			})
		}
	}
	return resp, nil
}

func (s *LinkServer) DeleteShortLink(ctx context.Context, req *linkv1.DeleteShortLinkRequest) (*linkv1.DeleteShortLinkResponse, error) {
	s.shortService.Log.Info("start", zap.String("op", "DeleteShortLink"))
	if err := req.Validate(); err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "DeleteShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	userID, ok := ctx.Value("user_id").(uuid.UUID)

	if !ok {
		s.shortService.Log.Warn("failed", zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}

	linkID, err := uuid.Parse(req.Id)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "DeleteShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "invalid link ID: %v", err)
	}

	err = s.shortService.DeactivateShortLink(linkID, userID)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "DeleteShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete short link: %v", err)
	}

	return &linkv1.DeleteShortLinkResponse{
		Message: "short link deleted successfully",
	}, nil
}

func (s *LinkServer) GetShortLink(ctx context.Context, req *linkv1.GetShortLinkRequest) (*linkv1.ShortLinkResponse, error) {
	s.shortService.Log.Info("start", zap.String("op", "GetShortLink"))
	if err := req.Validate(); err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "GetShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	userID, ok := ctx.Value("user_id").(uuid.UUID)

	if !ok {
		s.shortService.Log.Warn("failed", zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}

	shortLink, err := s.shortService.GetShortLinkByID(req.Id, userID)
	if err != nil {
		s.shortService.Log.Warn("failed", zap.String("op", "GetShortLink"), zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "short link not found: %v", err)
	}

	return &linkv1.ShortLinkResponse{
		Id:          shortLink.ID.String(),
		ShortUrl:    fmt.Sprintf("%s/%s", s.cfg.Domain, shortLink.ShortCode),
		OriginalUrl: shortLink.OriginalURL,
		ShortCode:   shortLink.ShortCode,
		UserId:      wrapperspb.String(shortLink.UserID.String()),
		ExpireAt:    shortLink.ExpireAt.String(),
		IsActive:    shortLink.IsActive,
	}, nil
}
