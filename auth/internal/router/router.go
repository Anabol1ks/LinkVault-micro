package router

import (
	"linkv-auth/config"
	"linkv-auth/internal/handler"
	"linkv-auth/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Router(db *gorm.DB, log *zap.Logger, userHandler *handler.UserHandler, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
		auth.POST("/refresh", userHandler.Refresh)
	}

	r.GET("/profile/me", middleware.JWTAuth(&cfg.JWT), userHandler.Profile)

	return r
}
