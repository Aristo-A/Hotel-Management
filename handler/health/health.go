package health

import (
	"net/http"

	"hotel-backend/database"

	"github.com/gin-gonic/gin"
)

// Check verifies the HTTP server status and database connectivity
func Check(c *gin.Context) {
	dbStatus := "healthy"
	if database.DB == nil {
		dbStatus = "uninitialized"
	} else {
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			dbStatus = "unreachable"
		}
	}

	statusCode := http.StatusOK
	if dbStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":   "ok",
		"service":  "hotel-management-api",
		"database": dbStatus,
	})
}
