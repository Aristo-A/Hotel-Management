package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireTenant_MissingHotelID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	// User is authenticated but has no assigned hotel property
	r.Use(func(c *gin.Context) {
		c.Set("userRole", "hotel_admin")
		// No hotelID set in context
		c.Next()
	})
	r.Use(RequireTenant())
	r.GET("/admin/rooms", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"rooms": []string{}})
	})

	req, _ := http.NewRequest("GET", "/admin/rooms", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when hotelID is missing from tenant session, got: %d", w.Code)
	}
}

func TestRequireTenant_ValidHotelID(t *testing.T) {
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(func(c *gin.Context) {
		c.Set("userRole", "hotel_admin")
		c.Set("hotelID", "property-uuid-456")
		c.Next()
	})
	r.Use(RequireTenant())
	r.GET("/admin/rooms", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "tenant access granted"})
	})

	req, _ := http.NewRequest("GET", "/admin/rooms", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid tenant context, got: %d", w.Code)
	}
}
