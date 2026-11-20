package routes

import (
	"hotel-backend/handler/auth"
	"hotel-backend/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes sets up auth endpoints
func RegisterAuthRoutes(router *gin.Engine) {
	registerGroup := func(group *gin.RouterGroup) {
		group.POST("/login", auth.Login)
		group.POST("/logout", middleware.RequireAuth(), auth.Logout)
		group.GET("/me", middleware.RequireAuth(), auth.Me)
		group.PUT("/profile", middleware.RequireAuth(), auth.UpdateProfile)
		group.PATCH("/password", middleware.RequireAuth(), auth.ChangePassword)
		group.PUT("/password", middleware.RequireAuth(), auth.ChangePassword)
		group.POST("/refresh", middleware.RequireAuth(), auth.Refresh)
	}

	// 1. Primary auth group (/api/auth)
	registerGroup(router.Group("/api/auth"))

	// 2. Integration alias group (/api/v1/auth)
	registerGroup(router.Group("/api/v1/auth"))
}
