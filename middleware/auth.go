package middleware

import (
	"net/http"
	"strings"

	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// RequireAuth validates the JWT in the Authorization Bearer header
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			util.SendError(c, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			util.SendError(c, http.StatusUnauthorized, "Invalid authorization header format (must be Bearer <token>)")
			return
		}

		tokenString := parts[1]
		claims, err := util.ValidateToken(tokenString)
		if err != nil {
			util.SendError(c, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Inject user context
		c.Set("userID", claims.UserID)
		c.Set("userName", claims.Name)
		c.Set("userEmail", claims.Email)
		c.Set("userRole", claims.Role)
		if claims.HotelID != nil {
			c.Set("hotelID", *claims.HotelID)
		}
		if claims.HotelName != "" {
			c.Set("hotelName", claims.HotelName)
		}

		c.Next()
	}
}
