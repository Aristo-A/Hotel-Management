package routes

import (
	"hotel-backend/handler/health"

	"github.com/gin-gonic/gin"
)

// RegisterHealthRoutes attaches health check endpoints
func RegisterHealthRoutes(router *gin.Engine) {
	router.GET("/health", health.Check)
	router.GET("/api/health", health.Check)
}
