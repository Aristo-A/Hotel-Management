package routes

import (
	"hotel-backend/handler/admin"
	"hotel-backend/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAdminRoutes sets up tenant-isolated hotel admin routes
func RegisterAdminRoutes(router *gin.Engine) {
	// 1. Primary route group matching frontend API endpoints (roomsAPI, billingAPI, reservationsAPI, reportsAPI)
	adminGroup := router.Group("/api/admin")
	adminGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		// Operational Dashboard
		adminGroup.GET("/dashboard", admin.GetDashboard)

		// Room Categories
		adminGroup.GET("/room-categories", admin.GetCategories)
		adminGroup.POST("/room-categories", admin.CreateCategory)
		adminGroup.GET("/room-categories/:id", admin.GetCategoryByID)
		adminGroup.PUT("/room-categories/:id", admin.UpdateCategory)
		adminGroup.DELETE("/room-categories/:id", admin.DeleteCategory)

		// Rooms
		adminGroup.GET("/rooms", admin.GetRooms)
		adminGroup.POST("/rooms", admin.CreateRoom)
		adminGroup.GET("/rooms/:id", admin.GetRoomByID)
		adminGroup.PUT("/rooms/:id", admin.UpdateRoom)
		adminGroup.DELETE("/rooms/:id", admin.DeleteRoom)
		adminGroup.PATCH("/rooms/:id/status", admin.UpdateRoomStatus)

		// Reservations & Operations
		adminGroup.GET("/reservations", admin.GetReservations)
		adminGroup.POST("/reservations", admin.CreateReservation)
		adminGroup.GET("/reservations/today/check-ins", admin.GetTodayCheckIns)
		adminGroup.GET("/reservations/today/check-outs", admin.GetTodayCheckOuts)
		adminGroup.GET("/reservations/:id", admin.GetReservationByID)
		adminGroup.PUT("/reservations/:id", admin.UpdateReservation)
		adminGroup.PATCH("/reservations/:id/cancel", admin.CancelReservation)
		adminGroup.PATCH("/reservations/:id/check-in", admin.CheckIn)
		adminGroup.POST("/reservations/:id/checkin", admin.CheckIn)
		adminGroup.PATCH("/reservations/:id/check-out", admin.CheckOut)
		adminGroup.POST("/reservations/:id/checkout", admin.CheckOut)

		// Housekeeping Tasks & Operations
		adminGroup.GET("/housekeeping", admin.GetHousekeepingTasks)
		adminGroup.PATCH("/housekeeping/:roomId", admin.UpdateHousekeepingTask)

		// Facility Charges (Running Folio)
		adminGroup.GET("/facility-charges", admin.GetFacilityCharges)
		adminGroup.POST("/facility-charges", admin.CreateFacilityCharge)
		adminGroup.DELETE("/facility-charges/:id", admin.DeleteFacilityCharge)

		// Billing & Invoicing
		adminGroup.GET("/bills", admin.GetBills)
		adminGroup.POST("/bills", admin.CreateBill)
		adminGroup.GET("/bills/:id", admin.GetBillByID)
		adminGroup.GET("/bills/:id/print", admin.GetPrintBill)
		adminGroup.POST("/bills/:id/items", admin.AddLineItem)
		adminGroup.POST("/bills/:id/payments", admin.CreateBillPaymentByBillID)
		adminGroup.PATCH("/bills/:id/discount", admin.ApplyDiscount)

		// Payments
		adminGroup.GET("/payments", admin.GetPayments)
		adminGroup.POST("/payments", admin.CreatePayment)
		adminGroup.GET("/payments/:id", admin.GetPaymentByID)

		// Analytics & Reports
		adminGroup.GET("/reports/occupancy", admin.GetOccupancyReport)
		adminGroup.GET("/reports/revenue", admin.GetRevenueReport)
		adminGroup.GET("/reports/payments", admin.GetRevenueReport)
		adminGroup.GET("/reports/housekeeping", admin.GetHousekeepingReport)
		adminGroup.GET("/reports/bookings", admin.GetBookingReport)
		adminGroup.GET("/reports/reservations", admin.GetBookingReport)
		adminGroup.GET("/reports/room-status", admin.GetRoomStatusReport)
	}

	// 2. Enterprise aliases matching BACKEND_INTEGRATION.md (/api/v1/hotel/*)
	v1HotelGroup := router.Group("/api/v1/hotel")
	v1HotelGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		// Operational Snapshot
		v1HotelGroup.GET("/dashboard/stats", admin.GetDashboard)

		// Housekeeping
		v1HotelGroup.GET("/housekeeping", admin.GetHousekeepingTasks)
		v1HotelGroup.PATCH("/housekeeping/:roomId", admin.UpdateHousekeepingTask)

		// Facility Charges (Running Folio)
		v1HotelGroup.GET("/facility-charges", admin.GetFacilityCharges)
		v1HotelGroup.POST("/facility-charges", admin.CreateFacilityCharge)
		v1HotelGroup.DELETE("/facility-charges/:id", admin.DeleteFacilityCharge)

		// Categories
		v1HotelGroup.GET("/categories", admin.GetCategories)
		v1HotelGroup.POST("/categories", admin.CreateCategory)
		v1HotelGroup.GET("/categories/:id", admin.GetCategoryByID)
		v1HotelGroup.PUT("/categories/:id", admin.UpdateCategory)
		v1HotelGroup.DELETE("/categories/:id", admin.DeleteCategory)

		// Rooms
		v1HotelGroup.GET("/rooms", admin.GetRooms)
		v1HotelGroup.POST("/rooms", admin.CreateRoom)
		v1HotelGroup.GET("/rooms/:id", admin.GetRoomByID)
		v1HotelGroup.PUT("/rooms/:id", admin.UpdateRoom)
		v1HotelGroup.DELETE("/rooms/:id", admin.DeleteRoom)
		v1HotelGroup.PATCH("/rooms/:id/status", admin.UpdateRoomStatus)

		// Reservations
		v1HotelGroup.GET("/reservations", admin.GetReservations)
		v1HotelGroup.POST("/reservations", admin.CreateReservation)
		v1HotelGroup.GET("/reservations/:id", admin.GetReservationByID)
		v1HotelGroup.PUT("/reservations/:id", admin.UpdateReservation)
		v1HotelGroup.POST("/reservations/:id/checkin", admin.CheckIn)
		v1HotelGroup.POST("/reservations/:id/checkout", admin.CheckOut)
		v1HotelGroup.POST("/reservations/:id/cancel", admin.CancelReservation)

		// Billing & Payments
		v1HotelGroup.GET("/bills", admin.GetBills)
		v1HotelGroup.POST("/bills", admin.CreateBill)
		v1HotelGroup.GET("/bills/:id", admin.GetBillByID)
		v1HotelGroup.GET("/bills/:id/print", admin.GetPrintBill)
		v1HotelGroup.POST("/bills/:id/items", admin.AddLineItem)
		v1HotelGroup.POST("/bills/:id/payments", admin.CreateBillPaymentByBillID)
		v1HotelGroup.POST("/payments", admin.CreatePayment)
		v1HotelGroup.GET("/payments", admin.GetPayments)

		// Reports
		v1HotelGroup.GET("/reports/occupancy", admin.GetOccupancyReport)
		v1HotelGroup.GET("/reports/revenue", admin.GetRevenueReport)
		v1HotelGroup.GET("/reports/payments", admin.GetRevenueReport)
		v1HotelGroup.GET("/reports/housekeeping", admin.GetHousekeepingReport)
		v1HotelGroup.GET("/reports/bookings", admin.GetBookingReport)
		v1HotelGroup.GET("/reports/reservations", admin.GetBookingReport)
		v1HotelGroup.GET("/reports/room-status", admin.GetRoomStatusReport)
	}
}
