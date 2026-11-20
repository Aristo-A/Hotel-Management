package routes

import (
	"hotel-backend/handler/superadmin"
	"hotel-backend/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterSuperAdminRoutes sets up platform-wide superadmin endpoints
func RegisterSuperAdminRoutes(router *gin.Engine) {
	// 1. Primary route group used by React frontend (hotelsAPI)
	superGroup := router.Group("/api/superadmin")
	superGroup.Use(middleware.RequireAuth(), middleware.RequireRole("superadmin"))
	{
		// Platform dashboard
		superGroup.GET("/dashboard", superadmin.GetDashboard)

		// Hotels management
		superGroup.GET("/hotels", superadmin.GetHotels)
		superGroup.POST("/hotels", superadmin.CreateHotel)
		superGroup.GET("/hotels/:id", superadmin.GetHotelByID)
		superGroup.PUT("/hotels/:id", superadmin.UpdateHotel)
		superGroup.DELETE("/hotels/:id", superadmin.DeleteHotel)
		superGroup.PATCH("/hotels/:id/status", superadmin.UpdateHotelStatus)
		superGroup.GET("/hotels/:id/admins", superadmin.GetHotelAdmins)

		// Admins management
		superGroup.GET("/admins", superadmin.GetAdmins)
		superGroup.POST("/admins", superadmin.CreateAdmin)
		superGroup.GET("/admins/:id", superadmin.GetAdminByID)
		superGroup.PUT("/admins/:id", superadmin.UpdateAdmin)
		superGroup.PATCH("/admins/:id/status", superadmin.UpdateAdminStatus)
		superGroup.DELETE("/admins/:id", superadmin.DeleteAdmin)
	}

	// 2. Integration aliases matching BACKEND_INTEGRATION.md specification (/api/v1/admin/*)
	v1AdminGroup := router.Group("/api/v1/admin")
	v1AdminGroup.Use(middleware.RequireAuth(), middleware.RequireRole("superadmin"))
	{
		v1AdminGroup.GET("/stats", superadmin.GetDashboard)
		v1AdminGroup.GET("/hotels", superadmin.GetHotels)
		v1AdminGroup.POST("/hotels", superadmin.CreateHotel)
		v1AdminGroup.GET("/hotels/:id", superadmin.GetHotelByID)
		v1AdminGroup.PUT("/hotels/:id", superadmin.UpdateHotel)
		v1AdminGroup.DELETE("/hotels/:id", superadmin.DeleteHotel)
		v1AdminGroup.PATCH("/hotels/:id/status", superadmin.UpdateHotelStatus)
		v1AdminGroup.GET("/hotels/:id/admins", superadmin.GetHotelAdmins)

		v1AdminGroup.GET("/users", superadmin.GetAdmins)
		v1AdminGroup.POST("/users", superadmin.CreateAdmin)
		v1AdminGroup.GET("/users/:id", superadmin.GetAdminByID)
		v1AdminGroup.PUT("/users/:id", superadmin.UpdateAdmin)
		v1AdminGroup.DELETE("/users/:id", superadmin.DeleteAdmin)
		v1AdminGroup.PATCH("/users/:id/status", superadmin.UpdateAdminStatus)
	}
}
