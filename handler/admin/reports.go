package admin

import (
	"math"
	"net/http"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// GetDashboard returns today's operational snapshot for the property
// GET /api/admin/dashboard
// GET /api/v1/hotel/dashboard/stats
func GetDashboard(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var totalRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ?", hotelID).Count(&totalRooms)

	var availableRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'available'", hotelID).Count(&availableRooms)

	var occupiedRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'occupied'", hotelID).Count(&occupiedRooms)

	var reservedRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'reserved'", hotelID).Count(&reservedRooms)

	var dirtyRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'dirty'", hotelID).Count(&dirtyRooms)

	var outOfServiceRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'out_of_service'", hotelID).Count(&outOfServiceRooms)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var todayCheckIns int64
	database.DB.Model(&models.Reservation{}).
		Where("hotel_id = ? AND check_in = ? AND status = 'confirmed'", hotelID, today).
		Count(&todayCheckIns)

	var todayCheckOuts int64
	database.DB.Model(&models.Reservation{}).
		Where("hotel_id = ? AND check_out = ? AND status = 'checked_in'", hotelID, today).
		Count(&todayCheckOuts)

	var pendingReservations int64
	database.DB.Model(&models.Reservation{}).
		Where("hotel_id = ? AND status = 'pending'", hotelID).
		Count(&pendingReservations)

	// Monthly revenue (current calendar month)
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var monthlyRevSum struct {
		Total float64
	}
	database.DB.Model(&models.Payment{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("hotel_id = ? AND status = 'completed' AND created_at >= ?", hotelID, firstOfMonth).
		Scan(&monthlyRevSum)
	monthlyRevenue := monthlyRevSum.Total

	// Fallback to all completed payments if 0
	if monthlyRevenue == 0 {
		database.DB.Model(&models.Payment{}).
			Select("COALESCE(SUM(amount), 0) as total").
			Where("hotel_id = ? AND status = 'completed'", hotelID).
			Scan(&monthlyRevSum)
		monthlyRevenue = monthlyRevSum.Total
	}

	var occupancyRate float64
	if totalRooms > 0 {
		occupancyRate = math.Round((float64(occupiedRooms)/float64(totalRooms))*1000) / 10
	}

	// Fetch recent 5 reservations
	var rawResvs []models.Reservation
	database.DB.Preload("Room").Preload("Room.Category").
		Where("hotel_id = ?", hotelID).
		Order("created_at desc").
		Limit(5).
		Find(&rawResvs)
	recentResvs := make([]ReservationDTO, 0, len(rawResvs))
	for i := range rawResvs {
		recentResvs = append(recentResvs, buildReservationDTO(&rawResvs[i]))
	}

	// Fetch rooms for room status grid
	var rawRooms []models.Room
	database.DB.Preload("Category").
		Where("hotel_id = ?", hotelID).
		Order("floor asc, room_number asc").
		Find(&rawRooms)
	roomDTOs := make([]RoomDTO, 0, len(rawRooms))
	for i := range rawRooms {
		roomDTOs = append(roomDTOs, buildRoomDTO(&rawRooms[i]))
	}

	statsData := gin.H{
		"totalRooms":          totalRooms,
		"availableRooms":      availableRooms,
		"occupiedRooms":       occupiedRooms,
		"reservedRooms":       reservedRooms,
		"dirtyRooms":          dirtyRooms,
		"outOfServiceRooms":   outOfServiceRooms,
		"todayCheckIns":       todayCheckIns,
		"todayCheckOuts":      todayCheckOuts,
		"pendingReservations": pendingReservations,
		"monthlyRevenue":      monthlyRevenue,
		"occupancyRate":       occupancyRate,
		"currency":            "INR",
		"recentReservations":  recentResvs,
		"rooms":               roomDTOs,
	}

	c.JSON(http.StatusOK, gin.H{
		"success":             true,
		"totalRooms":          totalRooms,
		"availableRooms":      availableRooms,
		"occupiedRooms":       occupiedRooms,
		"reservedRooms":       reservedRooms,
		"dirtyRooms":          dirtyRooms,
		"outOfServiceRooms":   outOfServiceRooms,
		"todayCheckIns":       todayCheckIns,
		"todayCheckOuts":      todayCheckOuts,
		"pendingReservations": pendingReservations,
		"monthlyRevenue":      monthlyRevenue,
		"occupancyRate":       occupancyRate,
		"currency":            "INR",
		"recentReservations":  recentResvs,
		"rooms":               roomDTOs,
		"data":                statsData,
	})
}

// GetOccupancyReport returns occupancy breakdown
// GET /api/admin/reports/occupancy
func GetOccupancyReport(c *gin.Context) {
	hotelID := c.GetString("hotelID")

	var totalRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ?", hotelID).Count(&totalRooms)
	var occupiedRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'occupied'", hotelID).Count(&occupiedRooms)

	var rate float64
	if totalRooms > 0 {
		rate = math.Round((float64(occupiedRooms)/float64(totalRooms))*1000) / 10
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"totalRooms":    totalRooms,
			"occupiedRooms": occupiedRooms,
			"occupancyRate": rate,
			"reportType":    "occupancy",
		},
	})
}

// GetRevenueReport returns revenue breakdown
// GET /api/admin/reports/revenue
func GetRevenueReport(c *gin.Context) {
	hotelID := c.GetString("hotelID")

	var totalCollected struct {
		Total float64
	}
	database.DB.Model(&models.Payment{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("hotel_id = ? AND status = 'completed'", hotelID).
		Scan(&totalCollected)

	var totalBilled struct {
		Total    float64
		Subtotal float64
		Tax      float64
	}
	database.DB.Model(&models.Bill{}).
		Select("COALESCE(SUM(total_amount), 0) as total, COALESCE(SUM(subtotal), 0) as subtotal, COALESCE(SUM(tax_amount), 0) as tax").
		Where("hotel_id = ?", hotelID).
		Scan(&totalBilled)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"totalCollected": totalCollected.Total,
			"totalBilled":    totalBilled.Total,
			"subtotal":       totalBilled.Subtotal,
			"taxCollected":   totalBilled.Tax,
			"currency":       "INR",
			"reportType":     "revenue",
		},
	})
}

// GetHousekeepingReport returns recent housekeeping logs
// GET /api/admin/reports/housekeeping
func GetHousekeepingReport(c *gin.Context) {
	hotelID := c.GetString("hotelID")

	var logs []models.HousekeepingLog
	database.DB.Preload("Room").
		Where("hotel_id = ?", hotelID).
		Order("created_at desc").
		Limit(100).
		Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logs,
		"logs":    logs,
		"total":   len(logs),
	})
}

// GetBookingReport returns booking statistics
// GET /api/admin/reports/bookings
func GetBookingReport(c *gin.Context) {
	hotelID := c.GetString("hotelID")

	var total int64
	var confirmed int64
	var checkedIn int64
	var checkedOut int64
	var cancelled int64

	database.DB.Model(&models.Reservation{}).Where("hotel_id = ?", hotelID).Count(&total)
	database.DB.Model(&models.Reservation{}).Where("hotel_id = ? AND status = 'confirmed'", hotelID).Count(&confirmed)
	database.DB.Model(&models.Reservation{}).Where("hotel_id = ? AND status = 'checked_in'", hotelID).Count(&checkedIn)
	database.DB.Model(&models.Reservation{}).Where("hotel_id = ? AND status = 'checked_out'", hotelID).Count(&checkedOut)
	database.DB.Model(&models.Reservation{}).Where("hotel_id = ? AND status = 'cancelled'", hotelID).Count(&cancelled)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total":      total,
			"confirmed":  confirmed,
			"checkedIn":  checkedIn,
			"checkedOut": checkedOut,
			"cancelled":  cancelled,
			"reportType": "bookings",
		},
	})
}

// GetRoomStatusReport returns a live breakdown of rooms by their operational status
// GET /api/admin/reports/room-status
// GET /api/v1/hotel/reports/room-status
func GetRoomStatusReport(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var total int64
	var available int64
	var occupied int64
	var reserved int64
	var dirty int64
	var cleaning int64
	var outOfService int64

	database.DB.Model(&models.Room{}).Where("hotel_id = ?", hotelID).Count(&total)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'available'", hotelID).Count(&available)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'occupied'", hotelID).Count(&occupied)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'reserved'", hotelID).Count(&reserved)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'dirty'", hotelID).Count(&dirty)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'cleaning'", hotelID).Count(&cleaning)
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'out_of_service'", hotelID).Count(&outOfService)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available":    available,
			"occupied":     occupied,
			"reserved":     reserved,
			"dirty":        dirty,
			"cleaning":     cleaning,
			"outOfService": outOfService,
			"total":        total,
		},
	})
}

