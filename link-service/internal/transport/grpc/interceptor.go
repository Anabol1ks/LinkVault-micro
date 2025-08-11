package grpc

import (
	"context"
	"fmt"
	"strings"

	authv1 "github.com/Anabol1ks/linkvault-proto/auth/v1"
	"github.com/google/uuid"

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

		userID, err := uuid.Parse(resp.UserId)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid user_id in token")
		}
		ctx = context.WithValue(ctx, "user_id", userID)
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
		fmt.Println("[OptionalAuthInterceptor] info.FullMethod:", info.FullMethod)
		if !authOptionalMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		fmt.Printf("[OptionalAuthInterceptor] metadata: %+v\n", md)
		authHeaders := md.Get("authorization")
		fmt.Printf("[OptionalAuthInterceptor] authHeaders: %+v\n", authHeaders)
		if len(authHeaders) > 0 && strings.HasPrefix(authHeaders[0], "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeaders[0], "Bearer ")
			fmt.Println("Calling ValidateAccessToken with token: ", tokenStr)
			resp, err := authClient.ValidateAccessToken(ctx, &authv1.ValidateAccessTokenRequest{
				AccessToken: tokenStr,
			})
			if err == nil && resp.Valid {
				userID, err := uuid.Parse(resp.UserId)
				if err != nil {
					return nil, status.Error(codes.Unauthenticated, "invalid user_id in token")
				}
				fmt.Println("[AuthInterceptor] user_id extracted from token:", userID)

				ctx = context.WithValue(ctx, "user_id", userID)
			}
		}
		fmt.Println("[OptionalAuthInterceptor] handler called")
		return handler(ctx, req)
	}
}

var authRequiredMethods = map[string]bool{}

var authOptionalMethods = map[string]bool{
	"/link.v1.LinkService/CreateShortLink": true,
}
