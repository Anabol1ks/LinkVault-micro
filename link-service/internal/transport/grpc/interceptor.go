package grpc

import (
	authv1 "auth-service/api/proto/auth/v1"
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(authClient authv1.AuthServiceClient) grpc.UnaryServerInterceptor {
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

		resp, err := authClient.ValidateAccessToken(ctx, &authv1.ValidateAccessTokenRequest{
			AccessToken: tokenStr,
		})
		if err != nil || !resp.Valid {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, "user_id", resp.UserId)
		return handler(ctx, req)
	}
}

func OptionalAuthInterceptor(authClient authv1.AuthServiceClient) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !authOptionalMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		authHeaders := md.Get("authorization")
		if len(authHeaders) > 0 && strings.HasPrefix(authHeaders[0], "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeaders[0], "Bearer ")
			resp, err := authClient.ValidateAccessToken(ctx, &authv1.ValidateAccessTokenRequest{
				AccessToken: tokenStr,
			})
			if err == nil && resp.Valid {
				ctx = context.WithValue(ctx, "user_id", resp.UserId)
			}
		}
		return handler(ctx, req)
	}
}

var authRequiredMethods = map[string]bool{}

var authOptionalMethods = map[string]bool{
	"/link.v1.LinkServer/CreateShortLink": true,
}
