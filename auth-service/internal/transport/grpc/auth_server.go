package grpc

import (
	authv1 "auth-service/api/proto/auth/v1"
	"auth-service/internal/jwt"
	"auth-service/internal/service"
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer
	userService *service.UserService
}

func NewAuthServer(userService *service.UserService, log *zap.Logger) *AuthServer {
	return &AuthServer{
		userService: userService,
	}
}

func (s *AuthServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	s.userService.Log.Info("start", zap.String("op", "Register"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "Register"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	user, err := s.userService.Register(req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			s.userService.Log.Warn("failed", zap.String("op", "Register"), zap.Error(err))
			return nil, status.Errorf(codes.AlreadyExists, "user already exists: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "Register"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &authv1.RegisterResponse{
		Id:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.TokenPair, error) {
	s.userService.Log.Info("start", zap.String("op", "Login"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "Login"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	access, refresh, err := s.userService.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			s.userService.Log.Warn("failed", zap.String("op", "Login"), zap.Error(err))
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		case errors.Is(err, service.ErrInvalidPassword):
			s.userService.Log.Warn("failed", zap.String("op", "Login"), zap.Error(err))
			return nil, status.Errorf(codes.Unauthenticated, "invalid password: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "Login"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}

	return &authv1.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthServer) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.TokenPair, error) {
	s.userService.Log.Info("start", zap.String("op", "Refresh"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "Refresh"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	access, refresh, err := s.userService.Refresh(req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			s.userService.Log.Warn("failed", zap.String("op", "Refresh"), zap.Error(err))
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "Refresh"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &authv1.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthServer) GetProfile(ctx context.Context, _ *authv1.GetProfileRequest) (*authv1.UserProfile, error) {
	s.userService.Log.Info("start", zap.String("op", "Profile"))
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		s.userService.Log.Warn("failed", zap.String("op", "Profile"), zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}
	user, err := s.userService.Profile(userID)
	if err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "Profile"), zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}
	return &authv1.UserProfile{
		Id:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*emptypb.Empty, error) {
	s.userService.Log.Info("start", zap.String("op", "Logout"))
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		s.userService.Log.Warn("failed", zap.String("op", "Logout"), zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}
	if err := s.userService.Logout(userID); err != nil {
		s.userService.Log.Error("failed", zap.String("op", "Logout"), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
	}
	s.userService.Log.Info("success", zap.String("op", "Logout"))
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) ValidateAccessToken(ctx context.Context, req *authv1.ValidateAccessTokenRequest) (*authv1.ValidateAccessTokenResponse, error) {
	claims, err := jwt.ParseAccessToken(req.AccessToken, s.userService.Cfg.JWT.Access)
	if err != nil {
		return &authv1.ValidateAccessTokenResponse{
			Valid: false,
		}, nil
	}
	return &authv1.ValidateAccessTokenResponse{
		UserId: claims.UserID,
		Valid:  true,
	}, nil
}

func (s *AuthServer) VerifyEmail(ctx context.Context, req *authv1.VerifyEmailRequest) (*emptypb.Empty, error) {
	s.userService.Log.Info("start", zap.String("op", "VerifyEmail"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "VerifyEmail"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	if err := s.userService.VerifyEmail(req.Token); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			s.userService.Log.Warn("failed", zap.String("op", "VerifyEmail"), zap.Error(err))
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "VerifyEmail"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) ResendVerificationEmail(ctx context.Context, _ *authv1.ResendVerificationEmailRequest) (*emptypb.Empty, error) {
	s.userService.Log.Info("start", zap.String("op", "ResendVerificationEmail"))
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		s.userService.Log.Warn("failed", zap.String("op", "ResendVerificationEmail"), zap.Error(errors.New("user_id not found in context")))
		return nil, status.Errorf(codes.Unauthenticated, "user not found: %v", "user_id not found in context")
	}
	if err := s.userService.ResendVerificationEmail(userID); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			s.userService.Log.Warn("failed", zap.String("op", "ResendVerificationEmail"), zap.Error(err))
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		case errors.Is(err, service.ErrEmailAlready):
			s.userService.Log.Warn("failed", zap.String("op", "ResendVerificationEmail"), zap.Error(err))
			return nil, status.Errorf(codes.AlreadyExists, "email already verified: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "ResendVerificationEmail"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) RequestPasswordReset(ctx context.Context, req *authv1.RequestPasswordResetRequest) (*emptypb.Empty, error) {
	s.userService.Log.Info("start", zap.String("op", "RequestPasswordReset"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "RequestPasswordReset"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	if err := s.userService.RequestPasswordReset(req.Email); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			s.userService.Log.Warn("failed", zap.String("op", "RequestPasswordReset"), zap.Error(err))
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "RequestPasswordReset"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *AuthServer) ConfirmPasswordReset(ctx context.Context, req *authv1.ConfirmPasswordResetRequest) (*emptypb.Empty, error) {
	s.userService.Log.Info("start", zap.String("op", "ConfirmPasswordReset"))
	if err := req.Validate(); err != nil {
		s.userService.Log.Warn("failed", zap.String("op", "ConfirmPasswordReset"), zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}
	if err := s.userService.ConfirmPasswordReset(req.Token, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			s.userService.Log.Warn("failed", zap.String("op", "ConfirmPasswordReset"), zap.Error(err))
			return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
		case errors.Is(err, service.ErrUserNotFound):
			s.userService.Log.Warn("failed", zap.String("op", "ConfirmPasswordReset"), zap.Error(err))
			return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
		default:
			s.userService.Log.Error("failed", zap.String("op", "ConfirmPasswordReset"), zap.Error(err))
			return nil, status.Errorf(codes.Internal, "internal server error: %v", err)
		}
	}
	return &emptypb.Empty{}, nil
}
