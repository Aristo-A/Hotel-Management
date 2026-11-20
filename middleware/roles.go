package middleware

import (
	"net/http"

	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// RequireRole checks whether the authenticated user has one of the allowed roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleObj, exists := c.Get("userRole")
		if !exists {
			util.SendError(c, http.StatusUnauthorized, "Authentication required")
			return
		}

		userRole, ok := roleObj.(string)
		if !ok {
			util.SendError(c, http.StatusForbidden, "Invalid user role")
			return
		}

		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		util.SendError(c, http.StatusForbidden, "Forbidden: insufficient permissions")
	}
}
