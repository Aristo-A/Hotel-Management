package routes

import (
	"hotel-backend/handler/public"

	"github.com/gin-gonic/gin"
)

// RegisterPublicRoutes sets up guest-facing public endpoints (zero-barrier / no auth)
func RegisterPublicRoutes(router *gin.Engine) {
	// 1. Primary route group matching frontend publicAPI
	publicGroup := router.Group("/api/public")
	{
		publicGroup.GET("/hotels/:hotelId", public.GetHotel)
		publicGroup.GET("/hotels/:hotelId/categories", public.GetRooms)
		publicGroup.GET("/hotels/:hotelId/rooms", public.GetRooms)
		publicGroup.GET("/hotels/:hotelId/availability", public.CheckAvailability)
		publicGroup.GET("/rooms/:id", public.GetPublicRoomDetail)
		publicGroup.GET("/categories/:id", public.GetPublicRoomDetail)
		publicGroup.POST("/bookings", public.CreateBooking)
		publicGroup.GET("/bookings/:bookingReference", public.GetBookingByRef)
	}

	// 2. Integration aliases matching BACKEND_INTEGRATION.md (/api/v1/public/*)
	v1PublicGroup := router.Group("/api/v1/public")
	{
		v1PublicGroup.GET("/hotel", public.GetHotel)
		v1PublicGroup.GET("/hotels/:hotelId", public.GetHotel)
		v1PublicGroup.GET("/rooms", public.CheckAvailability)
		v1PublicGroup.GET("/rooms/:id", public.GetPublicRoomDetail)
		v1PublicGroup.GET("/categories", public.GetRooms)
		v1PublicGroup.GET("/categories/:id", public.GetPublicRoomDetail)
		v1PublicGroup.POST("/bookings", public.CreateBooking)
		v1PublicGroup.GET("/bookings/:bookingReference", public.GetBookingByRef)
	}
}
