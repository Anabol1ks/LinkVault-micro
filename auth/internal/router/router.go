package router

import (
	"linkv-auth/config"
	"linkv-auth/internal/handler"
	"linkv-auth/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Router(db *gorm.DB, log *zap.Logger, userHandler *handler.UserHandler, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", userHandler.Register)
		auth.POST("/login", userHandler.Login)
		auth.POST("/refresh", userHandler.Refresh)
	}

	r.GET("/api/v1/profile/me", middleware.JWTAuth(&cfg.JWT), userHandler.Profile)

	return r
}
