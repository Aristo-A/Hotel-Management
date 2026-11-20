package middleware

import (
	"net/http"

	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// RequireTenant ensures that tenant-scoped operations for hotel_admin have an assigned hotel_id
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		hotelIDObj, exists := c.Get("hotelID")
		if !exists || hotelIDObj == "" {
			util.SendError(c, http.StatusForbidden, "No hotel property assigned to this administrative account")
			return
		}

		c.Next()
	}
}
