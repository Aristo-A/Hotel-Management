package superadmin

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HotelDTO provides standardized serialization for hotels with aggregated fields
type HotelDTO struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Address        string    `json:"address"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	Country        string    `json:"country"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	Website        string    `json:"website"`
	Image          string    `json:"image"`
	ImageURL       string    `json:"imageUrl"`
	Status         string    `json:"status"`
	StarRating     int       `json:"starRating"`
	Rating         float64   `json:"rating"`
	CheckInTime    string    `json:"checkInTime"`
	CheckOutTime   string    `json:"checkOutTime"`
	TotalRooms     int64     `json:"totalRooms"`
	AvailableRooms int64     `json:"availableRooms"`
	AdminID        string    `json:"adminId,omitempty"`
	AdminName      string    `json:"adminName,omitempty"`
	AdminEmail     string    `json:"adminEmail,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AdminDTO provides standardized serialization for hotel administrators
type AdminDTO struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	HotelID     string     `json:"hotelId,omitempty"`
	HotelName   string     `json:"hotelName,omitempty"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ActivityDTO formats activity logs for the dashboard
type ActivityDTO struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Time        time.Time `json:"time"`
	User        string    `json:"user"`
}

// Helper to log platform-level or property-level activity
func logActivity(c *gin.Context, hotelID *string, action string, details string) {
	var uid *string
	if val, exists := c.Get("userID"); exists {
		if s, ok := val.(string); ok && s != "" {
			uid = &s
		}
	}
	act := models.ActivityLog{
		HotelID: hotelID,
		UserID:  uid,
		Action:  action,
		Details: details,
	}
	database.DB.Create(&act)
}

// Helper to assemble full HotelDTO with room stats and assigned admin
func buildHotelDTO(hotel *models.Hotel) HotelDTO {
	var totalRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ?", hotel.ID).Count(&totalRooms)

	var availableRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND status = 'available'", hotel.ID).Count(&availableRooms)

	var adminID, adminName, adminEmail string
	var assignment models.HotelAdminAssignment
	if err := database.DB.Where("hotel_id = ?", hotel.ID).First(&assignment).Error; err == nil {
		var adminUser models.User
		if err := database.DB.Where("id = ?", assignment.UserID).First(&adminUser).Error; err == nil {
			adminID = adminUser.ID
			adminName = adminUser.Name
			adminEmail = adminUser.Email
		}
	}

	img := hotel.ImageURL
	if img == "" {
		img = hotel.Image
	}

	rating := float64(hotel.StarRating)
	if rating == 0 {
		rating = 4.5
	}

	return HotelDTO{
		ID:             hotel.ID,
		Name:           hotel.Name,
		Description:    hotel.Description,
		Address:        hotel.Address,
		City:           hotel.City,
		State:          hotel.State,
		Country:        hotel.Country,
		Phone:          hotel.Phone,
		Email:          hotel.Email,
		Website:        hotel.Website,
		Image:          img,
		ImageURL:       img,
		Status:         hotel.Status,
		StarRating:     hotel.StarRating,
		Rating:         rating,
		CheckInTime:    hotel.CheckInTime,
		CheckOutTime:   hotel.CheckOutTime,
		TotalRooms:     totalRooms,
		AvailableRooms: availableRooms,
		AdminID:        adminID,
		AdminName:      adminName,
		AdminEmail:     adminEmail,
		CreatedAt:      hotel.CreatedAt,
	}
}

// GetDashboard returns global platform-wide metrics, top hotels, and live activities
// GET /api/superadmin/dashboard
// GET /api/v1/admin/stats
func GetDashboard(c *gin.Context) {
	var totalHotels int64
	database.DB.Model(&models.Hotel{}).Count(&totalHotels)

	var activeHotels int64
	database.DB.Model(&models.Hotel{}).Where("status = 'active'").Count(&activeHotels)

	inactiveHotels := totalHotels - activeHotels

	var totalAdmins int64
	database.DB.Model(&models.User{}).Where("role = 'hotel_admin'").Count(&totalAdmins)

	var totalBookings int64
	database.DB.Model(&models.Reservation{}).Count(&totalBookings)

	// Total revenue from completed payments or total paid bills
	var totalRevenue float64
	var paymentSum struct {
		Total float64
	}
	database.DB.Model(&models.Payment{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("status = 'completed'").
		Scan(&paymentSum)
	totalRevenue = paymentSum.Total

	// Fallback to bills total if payments are zero (e.g. initial seeded state)
	if totalRevenue == 0 {
		var billSum struct {
			Total float64
		}
		database.DB.Model(&models.Bill{}).
			Select("COALESCE(SUM(amount_paid), 0) as total").
			Scan(&billSum)
		totalRevenue = billSum.Total
	}

	// Calculate occupancy rate: occupied rooms / total rooms * 100
	var totalRooms int64
	database.DB.Model(&models.Room{}).Count(&totalRooms)
	var occupiedRooms int64
	database.DB.Model(&models.Room{}).Where("status = 'occupied'").Count(&occupiedRooms)

	var occupancyRate float64
	if totalRooms > 0 {
		occupancyRate = math.Round((float64(occupiedRooms)/float64(totalRooms))*1000) / 10
	} else {
		occupancyRate = 0.0
	}

	// Fetch top 5 hotels for quick overview
	var rawHotels []models.Hotel
	database.DB.Order("created_at desc").Limit(5).Find(&rawHotels)
	hotelDTOs := make([]HotelDTO, 0, len(rawHotels))
	for i := range rawHotels {
		hotelDTOs = append(hotelDTOs, buildHotelDTO(&rawHotels[i]))
	}

	// Fetch recent activities (up to 10)
	var rawLogs []models.ActivityLog
	database.DB.Order("created_at desc").Limit(10).Find(&rawLogs)
	activities := make([]ActivityDTO, 0, len(rawLogs))
	for _, l := range rawLogs {
		userName := "System"
		if l.UserID != nil && *l.UserID != "" {
			var u models.User
			if err := database.DB.Where("id = ?", *l.UserID).First(&u).Error; err == nil {
				userName = u.Name
			}
		}
		activities = append(activities, ActivityDTO{
			ID:          l.ID,
			Type:        l.Action,
			Description: l.Details,
			Time:        l.CreatedAt,
			User:        userName,
		})
	}

	dashboardData := gin.H{
		"totalHotels":      totalHotels,
		"activeHotels":     activeHotels,
		"inactiveHotels":   inactiveHotels,
		"totalAdmins":      totalAdmins,
		"totalRevenue":     totalRevenue,
		"occupancyRate":    occupancyRate,
		"totalBookings":    totalBookings,
		"currency":         "INR",
		"hotels":           hotelDTOs,
		"recentActivities": activities,
	}

	// Send unified response supporting both direct fields and wrapped data
	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"totalHotels":      totalHotels,
		"activeHotels":     activeHotels,
		"inactiveHotels":   inactiveHotels,
		"totalAdmins":      totalAdmins,
		"totalRevenue":     totalRevenue,
		"occupancyRate":    occupancyRate,
		"totalBookings":    totalBookings,
		"currency":         "INR",
		"hotels":           hotelDTOs,
		"recentActivities": activities,
		"data":             dashboardData,
	})
}

// GetHotels returns all hotels with pagination, searching, and status filtering
// GET /api/superadmin/hotels
// GET /api/v1/admin/hotels
func GetHotels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))

	query := database.DB.Model(&models.Hotel{})

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(city) LIKE ? OR LOWER(state) LIKE ? OR LOWER(country) LIKE ?", s, s, s, s)
	}

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var hotels []models.Hotel
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&hotels).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve hotel directory. Please try again.")
		return
	}

	hotelDTOs := make([]HotelDTO, 0, len(hotels))
	for i := range hotels {
		hotelDTOs = append(hotelDTOs, buildHotelDTO(&hotels[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    hotelDTOs,
		"hotels":  hotelDTOs,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetHotelByID returns a single hotel by UUID or ID
// GET /api/superadmin/hotels/:id
// GET /api/v1/admin/hotels/:id
func GetHotelByID(c *gin.Context) {
	id := c.Param("id")

	var hotel models.Hotel
	if err := database.DB.Where("id = ?", id).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel not found")
		return
	}

	dto := buildHotelDTO(&hotel)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"hotel":   dto,
	})
}

// CreateHotelRequest defines the request body for registering a hotel
type CreateHotelRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	Address      string `json:"address" binding:"required"`
	City         string `json:"city" binding:"required"`
	State        string `json:"state"`
	Country      string `json:"country"`
	Phone        string `json:"phone" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Website      string `json:"website"`
	Image        string `json:"image"`
	ImageURL     string `json:"imageUrl"`
	Status       string `json:"status"`
	Rating       int    `json:"rating"`
	StarRating   int    `json:"starRating"`
	CheckInTime  string `json:"checkInTime"`
	CheckOutTime string `json:"checkOutTime"`
	AdminID      string `json:"adminId"`
}

// CreateHotel registers a new hotel property
// POST /api/superadmin/hotels
// POST /api/v1/admin/hotels
func CreateHotel(c *gin.Context) {
	var req CreateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	img := req.ImageURL
	if img == "" {
		img = req.Image
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	country := strings.TrimSpace(req.Country)
	if country == "" {
		country = "India"
	}

	starRating := req.StarRating
	if starRating <= 0 {
		starRating = req.Rating
	}
	if starRating < 1 {
		starRating = 4
	} else if starRating > 5 {
		starRating = 5
	}

	checkIn := strings.TrimSpace(req.CheckInTime)
	if checkIn == "" {
		checkIn = "14:00"
	}
	checkOut := strings.TrimSpace(req.CheckOutTime)
	if checkOut == "" {
		checkOut = "12:00"
	}

	hotel := models.Hotel{
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Address:      strings.TrimSpace(req.Address),
		City:         strings.TrimSpace(req.City),
		State:        strings.TrimSpace(req.State),
		Country:      country,
		Phone:        strings.TrimSpace(req.Phone),
		Email:        strings.TrimSpace(req.Email),
		Website:      strings.TrimSpace(req.Website),
		ImageURL:     img,
		Status:       status,
		StarRating:   starRating,
		CheckInTime:  checkIn,
		CheckOutTime: checkOut,
	}

	if err := database.DB.Create(&hotel).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to register hotel property. Please try again.")
		return
	}

	// Link admin if adminId was supplied
	if req.AdminID != "" {
		var adminUser models.User
		if err := database.DB.Where("id = ? AND role = 'hotel_admin'", req.AdminID).First(&adminUser).Error; err == nil {
			// Remove any previous assignment for this hotel
			database.DB.Where("hotel_id = ?", hotel.ID).Delete(&models.HotelAdminAssignment{})
			assignment := models.HotelAdminAssignment{
				UserID:  adminUser.ID,
				HotelID: hotel.ID,
			}
			database.DB.Create(&assignment)
		}
	}

	logActivity(c, &hotel.ID, "create_hotel", fmt.Sprintf("Created property %s in %s", hotel.Name, hotel.City))

	dto := buildHotelDTO(&hotel)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Hotel successfully created",
		"data":    dto,
		"hotel":   dto,
	})
}

// UpdateHotel updates existing hotel details
// PUT /api/superadmin/hotels/:id
// PUT /api/v1/admin/hotels/:id
func UpdateHotel(c *gin.Context) {
	id := c.Param("id")

	var hotel models.Hotel
	if err := database.DB.Where("id = ?", id).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel not found")
		return
	}

	var req CreateHotelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	if req.Name != "" {
		hotel.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		hotel.Description = strings.TrimSpace(req.Description)
	}
	if req.Address != "" {
		hotel.Address = strings.TrimSpace(req.Address)
	}
	if req.City != "" {
		hotel.City = strings.TrimSpace(req.City)
	}
	if req.State != "" {
		hotel.State = strings.TrimSpace(req.State)
	}
	if req.Country != "" {
		hotel.Country = strings.TrimSpace(req.Country)
	}
	if req.Phone != "" {
		hotel.Phone = strings.TrimSpace(req.Phone)
	}
	if req.Email != "" {
		hotel.Email = strings.TrimSpace(req.Email)
	}
	if req.Website != "" {
		hotel.Website = strings.TrimSpace(req.Website)
	}
	if req.ImageURL != "" {
		hotel.ImageURL = strings.TrimSpace(req.ImageURL)
	} else if req.Image != "" {
		hotel.ImageURL = strings.TrimSpace(req.Image)
	}
	if req.Status != "" {
		hotel.Status = strings.TrimSpace(req.Status)
	}
	if req.StarRating > 0 {
		hotel.StarRating = req.StarRating
	} else if req.Rating > 0 {
		hotel.StarRating = req.Rating
	}
	if req.CheckInTime != "" {
		hotel.CheckInTime = strings.TrimSpace(req.CheckInTime)
	}
	if req.CheckOutTime != "" {
		hotel.CheckOutTime = strings.TrimSpace(req.CheckOutTime)
	}

	if err := database.DB.Save(&hotel).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update hotel property. Please try again.")
		return
	}

	// Update admin assignment if adminId was provided
	if req.AdminID != "" {
		var adminUser models.User
		if err := database.DB.Where("id = ? AND role = 'hotel_admin'", req.AdminID).First(&adminUser).Error; err == nil {
			database.DB.Where("hotel_id = ?", hotel.ID).Delete(&models.HotelAdminAssignment{})
			assignment := models.HotelAdminAssignment{
				UserID:  adminUser.ID,
				HotelID: hotel.ID,
			}
			database.DB.Create(&assignment)
		}
	}

	logActivity(c, &hotel.ID, "update_hotel", fmt.Sprintf("Updated property details for %s", hotel.Name))

	dto := buildHotelDTO(&hotel)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hotel successfully updated",
		"data":    dto,
		"hotel":   dto,
	})
}

// UpdateStatusRequest for status toggle requests
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateHotelStatus toggles active/inactive
// PATCH /api/superadmin/hotels/:id/status
// PATCH /api/v1/admin/hotels/:id/status
func UpdateHotelStatus(c *gin.Context) {
	id := c.Param("id")

	var hotel models.Hotel
	if err := database.DB.Where("id = ?", id).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel not found")
		return
	}

	var req UpdateStatusRequest
	_ = c.ShouldBindJSON(&req)

	newStatus := strings.TrimSpace(req.Status)
	if newStatus == "" {
		if hotel.Status == "active" {
			newStatus = "inactive"
		} else {
			newStatus = "active"
		}
	}

	hotel.Status = newStatus
	if err := database.DB.Model(&hotel).Update("status", newStatus).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update hotel status. Please try again.")
		return
	}

	logActivity(c, &hotel.ID, "update_hotel_status", fmt.Sprintf("Changed status of %s to %s", hotel.Name, newStatus))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Hotel status updated to %s", newStatus),
		"id":      hotel.ID,
		"status":  hotel.Status,
		"data": gin.H{
			"id":     hotel.ID,
			"status": hotel.Status,
		},
	})
}

// DeleteHotel removes a hotel property
// DELETE /api/superadmin/hotels/:id
// DELETE /api/v1/admin/hotels/:id
func DeleteHotel(c *gin.Context) {
	id := c.Param("id")

	var hotel models.Hotel
	if err := database.DB.Where("id = ?", id).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel not found")
		return
	}

	// Security check: cannot delete hotel if active confirmed/checked_in reservations exist
	var activeReservations int64
	database.DB.Model(&models.Reservation{}).
		Where("hotel_id = ? AND status IN ('confirmed', 'checked_in')", id).
		Count(&activeReservations)

	if activeReservations > 0 {
		util.SendError(c, http.StatusBadRequest, "Cannot delete property with active guest reservations")
		return
	}

	// Remove admin assignments
	database.DB.Where("hotel_id = ?", id).Delete(&models.HotelAdminAssignment{})

	// Delete hotel
	if err := database.DB.Delete(&hotel).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to delete hotel property. Please try again.")
		return
	}

	logActivity(c, nil, "delete_hotel", fmt.Sprintf("Deleted property %s (ID: %s)", hotel.Name, id))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Hotel successfully deleted",
		"id":      id,
	})
}

// GetHotelAdmins returns all administrators assigned to a specific hotel
// GET /api/superadmin/hotels/:id/admins
func GetHotelAdmins(c *gin.Context) {
	hotelID := c.Param("id")
	if hotelID == "" {
		hotelID = c.Param("hotelId")
	}

	var hotel models.Hotel
	if err := database.DB.Where("id = ?", hotelID).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel not found")
		return
	}

	var assignments []models.HotelAdminAssignment
	database.DB.Where("hotel_id = ?", hotelID).Find(&assignments)

	adminDTOs := make([]AdminDTO, 0, len(assignments))
	for _, a := range assignments {
		var u models.User
		if err := database.DB.Where("id = ?", a.UserID).First(&u).Error; err == nil {
			adminDTOs = append(adminDTOs, AdminDTO{
				ID:          u.ID,
				Name:        u.Name,
				Email:       u.Email,
				Phone:       u.Phone,
				Role:        u.Role,
				Status:      u.Status,
				HotelID:     hotel.ID,
				HotelName:   hotel.Name,
				LastLoginAt: u.LastLoginAt,
				CreatedAt:   u.CreatedAt,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    adminDTOs,
		"admins":  adminDTOs,
	})
}

// GetAdmins returns all hotel administrators
// GET /api/superadmin/admins
// GET /api/v1/admin/users
func GetAdmins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	hotelID := strings.TrimSpace(c.Query("hotelId"))

	query := database.DB.Model(&models.User{}).Where("role = 'hotel_admin'")

	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR phone LIKE ?", s, s, s)
	}

	if hotelID != "" {
		query = query.Where("id IN (SELECT user_id FROM hotel_admin_assignments WHERE hotel_id = ?)", hotelID)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var users []models.User
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve administrators. Please try again.")
		return
	}

	adminDTOs := make([]AdminDTO, 0, len(users))
	for _, u := range users {
		var a models.HotelAdminAssignment
		var hID, hName string
		if err := database.DB.Where("user_id = ?", u.ID).First(&a).Error; err == nil {
			var h models.Hotel
			if err := database.DB.Where("id = ?", a.HotelID).First(&h).Error; err == nil {
				hID = h.ID
				hName = h.Name
			}
		}

		adminDTOs = append(adminDTOs, AdminDTO{
			ID:          u.ID,
			Name:        u.Name,
			Email:       u.Email,
			Phone:       u.Phone,
			Role:        u.Role,
			Status:      u.Status,
			HotelID:     hID,
			HotelName:   hName,
			LastLoginAt: u.LastLoginAt,
			CreatedAt:   u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    adminDTOs,
		"admins":  adminDTOs,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetAdminByID returns single admin detail
// GET /api/superadmin/admins/:id
// GET /api/v1/admin/users/:id
func GetAdminByID(c *gin.Context) {
	id := c.Param("id")

	var u models.User
	if err := database.DB.Where("id = ? AND role = 'hotel_admin'", id).First(&u).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Administrator not found")
		return
	}

	var a models.HotelAdminAssignment
	var hID, hName string
	if err := database.DB.Where("user_id = ?", u.ID).First(&a).Error; err == nil {
		var h models.Hotel
		if err := database.DB.Where("id = ?", a.HotelID).First(&h).Error; err == nil {
			hID = h.ID
			hName = h.Name
		}
	}

	dto := AdminDTO{
		ID:          u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Phone:       u.Phone,
		Role:        u.Role,
		Status:      u.Status,
		HotelID:     hID,
		HotelName:   hName,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"admin":   dto,
	})
}

// CreateAdminRequest defines request for creating a hotel administrator
type CreateAdminRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	HotelID  string `json:"hotelId" binding:"required"`
	Status   string `json:"status"`
}

// CreateAdmin registers and assigns a new hotel administrator
// POST /api/superadmin/admins
// POST /api/v1/admin/users
func CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	cleanEmail := strings.ToLower(strings.TrimSpace(req.Email))

	// 1. Verify email uniqueness
	var count int64
	database.DB.Model(&models.User{}).Where("LOWER(email) = ?", cleanEmail).Count(&count)
	if count > 0 {
		util.SendError(c, http.StatusBadRequest, "An account with this email address already exists")
		return
	}

	// 2. Verify hotel exists
	var hotel models.Hotel
	if err := database.DB.Where("id = ?", req.HotelID).First(&hotel).Error; err != nil {
		util.SendError(c, http.StatusBadRequest, "Assigned hotel not found")
		return
	}

	// 3. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		util.SendError(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}

	// 4. Create User
	user := models.User{
		Name:         strings.TrimSpace(req.Name),
		Email:        cleanEmail,
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: string(hashedPassword),
		Role:         "hotel_admin",
		Status:       status,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to create administrator account. Please try again.")
		return
	}

	// 5. Create Hotel Assignment
	assignment := models.HotelAdminAssignment{
		UserID:  user.ID,
		HotelID: hotel.ID,
	}
	database.DB.Create(&assignment)

	logActivity(c, &hotel.ID, "create_admin", fmt.Sprintf("Created administrator %s for %s", user.Name, hotel.Name))

	dto := AdminDTO{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		Role:      user.Role,
		Status:    user.Status,
		HotelID:   hotel.ID,
		HotelName: hotel.Name,
		CreatedAt: user.CreatedAt,
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Hotel administrator successfully created",
		"data":    dto,
		"admin":   dto,
	})
}

// UpdateAdminRequest defines the payload for updating an administrator
type UpdateAdminRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	HotelID string `json:"hotelId"`
	Status  string `json:"status"`
}

// UpdateAdmin updates an administrator's profile or assigned hotel
// PUT /api/superadmin/admins/:id
// PUT /api/v1/admin/users/:id
func UpdateAdmin(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.Where("id = ? AND role = 'hotel_admin'", id).First(&user).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Administrator not found")
		return
	}

	var req UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	if req.Name != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	if req.Phone != "" {
		user.Phone = strings.TrimSpace(req.Phone)
	}
	if req.Status != "" {
		user.Status = strings.TrimSpace(req.Status)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update administrator details. Please try again.")
		return
	}

	var hotel models.Hotel
	if req.HotelID != "" {
		if err := database.DB.Where("id = ?", req.HotelID).First(&hotel).Error; err == nil {
			database.DB.Where("user_id = ?", user.ID).Delete(&models.HotelAdminAssignment{})
			assignment := models.HotelAdminAssignment{
				UserID:  user.ID,
				HotelID: hotel.ID,
			}
			database.DB.Create(&assignment)
		}
	} else {
		var a models.HotelAdminAssignment
		if err := database.DB.Where("user_id = ?", user.ID).First(&a).Error; err == nil {
			database.DB.Where("id = ?", a.HotelID).First(&hotel)
		}
	}

	logActivity(c, &hotel.ID, "update_admin", fmt.Sprintf("Updated administrator account for %s", user.Name))

	dto := AdminDTO{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Phone:       user.Phone,
		Role:        user.Role,
		Status:      user.Status,
		HotelID:     hotel.ID,
		HotelName:   hotel.Name,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Administrator successfully updated",
		"data":    dto,
		"admin":   dto,
	})
}

// UpdateAdminStatus toggles or updates an administrator's active/inactive status
// PATCH /api/superadmin/admins/:id/status
func UpdateAdminStatus(c *gin.Context) {
	id := c.Param("id")

	currentUIDVal, _ := c.Get("userID")
	if currentUID, ok := currentUIDVal.(string); ok && currentUID == id {
		util.SendError(c, http.StatusBadRequest, "Cannot modify status of your own account")
		return
	}

	var user models.User
	if err := database.DB.Where("id = ? AND role = 'hotel_admin'", id).First(&user).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Administrator not found")
		return
	}

	var req UpdateStatusRequest
	_ = c.ShouldBindJSON(&req)

	newStatus := strings.TrimSpace(req.Status)
	if newStatus == "" {
		if user.Status == "active" {
			newStatus = "inactive"
		} else {
			newStatus = "active"
		}
	}

	user.Status = newStatus
	if err := database.DB.Model(&user).Update("status", newStatus).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update administrator status. Please try again.")
		return
	}

	logActivity(c, nil, "update_admin_status", fmt.Sprintf("Updated admin %s status to %s", user.Email, newStatus))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Admin status updated to %s", newStatus),
		"id":      user.ID,
		"status":  user.Status,
	})
}

// DeleteAdmin removes an administrator account
// DELETE /api/superadmin/admins/:id
// DELETE /api/v1/admin/users/:id
func DeleteAdmin(c *gin.Context) {
	id := c.Param("id")

	currentUIDVal, _ := c.Get("userID")
	if currentUID, ok := currentUIDVal.(string); ok && currentUID == id {
		util.SendError(c, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	var user models.User
	if err := database.DB.Where("id = ? AND role = 'hotel_admin'", id).First(&user).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Administrator not found")
		return
	}

	database.DB.Where("user_id = ?", id).Delete(&models.HotelAdminAssignment{})
	if err := database.DB.Delete(&user).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to delete administrator account. Please try again.")
		return
	}

	logActivity(c, nil, "delete_admin", fmt.Sprintf("Deleted administrator %s", user.Email))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Administrator successfully deleted",
		"id":      id,
	})
}
