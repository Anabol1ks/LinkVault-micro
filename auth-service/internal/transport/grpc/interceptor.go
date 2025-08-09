package grpc

import (
	"context"
	"strings"

	"linkv-auth/config"
	"linkv-auth/internal/jwt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(jwtCfg *config.JWTConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !authRequiredMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 || !strings.HasPrefix(authHeaders[0], "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid token")
		}
		tokenStr := strings.TrimPrefix(authHeaders[0], "Bearer ")
		claims, err := jwt.ParseAccessToken(tokenStr, jwtCfg.Access)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid user_id in token")
		}
		ctx = context.WithValue(ctx, "user_id", userID)
		return handler(ctx, req)
	}
}

var authRequiredMethods = map[string]bool{
	"/auth.v1.AuthService/GetProfile": true,
	"/auth.v1.AuthService/Logout":     true,
}
