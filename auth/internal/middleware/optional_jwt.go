package middleware

import (
	"linkv-auth/config"
	"linkv-auth/internal/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

func OptionalJWTAuth(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwt.ParseAccessToken(tokenStr, cfg.Access)
			if err == nil {
				c.Set("user_id", claims.UserID)
			}
		}
		c.Next()
	}
}
