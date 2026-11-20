package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(RequireAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing auth header, got: %d", w.Code)
	}
	_ = c
}

func TestRequireAuth_InvalidBearerFormat(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(RequireAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // Not Bearer
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid auth format, got: %d", w.Code)
	}
}

func TestRequireAuth_ValidTokenInjectsContext(t *testing.T) {
	os.Setenv("JWT_SECRET", "secret_for_middleware_test")

	hotelID := "hotel-mumbai-01"
	tokenStr, _ := util.GenerateToken("user-123", "Arjun Sharma", "admin@horizon.in", "hotel_admin", &hotelID, "Grand Horizon")

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var capturedUserID, capturedRole, capturedHotelID string
	r.Use(RequireAuth())
	r.GET("/protected", func(c *gin.Context) {
		if val, exists := c.Get("userID"); exists {
			capturedUserID, _ = val.(string)
		}
		if val, exists := c.Get("userRole"); exists {
			capturedRole, _ = val.(string)
		}
		if val, exists := c.Get("hotelID"); exists {
			capturedHotelID, _ = val.(string)
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got: %d", w.Code)
	}
	if capturedUserID != "user-123" {
		t.Errorf("expected user-123, got: %s", capturedUserID)
	}
	if capturedRole != "hotel_admin" {
		t.Errorf("expected hotel_admin, got: %s", capturedRole)
	}
	if capturedHotelID != hotelID {
		t.Errorf("expected hotel ID %s, got: %s", hotelID, capturedHotelID)
	}
}
