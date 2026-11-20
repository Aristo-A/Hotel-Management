package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireRole_Authorized(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set("userRole", "superadmin")
		c.Next()
	})
	r.Use(RequireRole("superadmin"))
	r.GET("/superadmin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"access": "granted"})
	})

	req, _ := http.NewRequest("GET", "/superadmin-only", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got: %d", w.Code)
	}
}

func TestRequireRole_ForbiddenRole(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// User is hotel_admin attempting to access a superadmin-only route
	r.Use(func(c *gin.Context) {
		c.Set("userRole", "hotel_admin")
		c.Next()
	})
	r.Use(RequireRole("superadmin"))
	r.GET("/superadmin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"access": "granted"})
	})

	req, _ := http.NewRequest("GET", "/superadmin-only", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for unauthorized role, got: %d", w.Code)
	}
}
