package routes

import (
	"net/http"

	"hotel-backend/middleware"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// SetupRoutes builds and registers all API routes and global middleware
func SetupRoutes() *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(middleware.RequestLogger())
	router.Use(middleware.SetupCORS())

	// Handle 404
	router.NoRoute(func(c *gin.Context) {
		util.SendError(c, http.StatusNotFound, "The requested endpoint was not found")
	})

	// Register modular route groups
	RegisterHealthRoutes(router)
	RegisterAuthRoutes(router)
	RegisterPublicRoutes(router)
	RegisterAdminRoutes(router)
	RegisterSuperAdminRoutes(router)

	return router
}
