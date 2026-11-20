package admin

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// ReservationDTO formats reservation records for the front desk
type ReservationDTO struct {
	ID               string    `json:"id"`
	BookingReference string    `json:"bookingReference"`
	HotelID          string    `json:"hotelId"`
	RoomID           *string   `json:"roomId"`
	RoomNumber       string    `json:"roomNumber"`
	CategoryID       string    `json:"categoryId,omitempty"`
	CategoryName     string    `json:"categoryName"`
	GuestID          *string   `json:"guestId,omitempty"`
	GuestName        string    `json:"guestName"`
	GuestEmail       string    `json:"guestEmail"`
	GuestPhone       string    `json:"guestPhone"`
	CheckIn          string    `json:"checkIn"`
	CheckOut         string    `json:"checkOut"`
	Nights           int       `json:"nights"`
	NumGuests        int       `json:"numGuests"`
	RoomRate         float64   `json:"roomRate"`
	Subtotal         float64   `json:"subtotal"`
	TaxRate          float64   `json:"taxRate"`
	TaxAmount        float64   `json:"taxAmount"`
	TotalAmount      float64   `json:"totalAmount"`
	Status           string    `json:"status"`
	SpecialRequests  string    `json:"specialRequests,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

func buildReservationDTO(resv *models.Reservation) ReservationDTO {
	roomNum := "TBA"
	catName := "Standard Room"
	catID := ""

	if resv.Room != nil {
		roomNum = resv.Room.RoomNumber
		if resv.Room.Category != nil {
			catName = resv.Room.Category.Name
			catID = resv.Room.Category.ID
		}
	} else if resv.RoomID != nil {
		var r models.Room
		if err := database.DB.Preload("Category").Where("id = ?", *resv.RoomID).First(&r).Error; err == nil {
			roomNum = r.RoomNumber
			if r.Category != nil {
				catName = r.Category.Name
				catID = r.Category.ID
			}
		}
	}

	subtotal := math.Round(resv.RoomRate*float64(resv.Nights)*100) / 100
	taxRate := 0.12
	taxAmount := math.Round(subtotal*taxRate*100) / 100
	total := resv.TotalAmount
	if total <= 0 {
		total = math.Round((subtotal+taxAmount)*100) / 100
	}

	return ReservationDTO{
		ID:               resv.ID,
		BookingReference: resv.BookingReference,
		HotelID:          resv.HotelID,
		RoomID:           resv.RoomID,
		RoomNumber:       roomNum,
		CategoryID:       catID,
		CategoryName:     catName,
		GuestID:          resv.GuestID,
		GuestName:        resv.GuestName,
		GuestEmail:       resv.GuestEmail,
		GuestPhone:       resv.GuestPhone,
		CheckIn:          resv.CheckIn.Format("2006-01-02"),
		CheckOut:         resv.CheckOut.Format("2006-01-02"),
		Nights:           resv.Nights,
		NumGuests:        resv.NumGuests,
		RoomRate:         resv.RoomRate,
		Subtotal:         subtotal,
		TaxRate:          taxRate,
		TaxAmount:        taxAmount,
		TotalAmount:      total,
		Status:           resv.Status,
		SpecialRequests:  resv.SpecialRequests,
		CreatedAt:        resv.CreatedAt,
	}
}

func generateAdminBookingRef() string {
	year := time.Now().Year()
	n, _ := rand.Int(rand.Reader, big.NewInt(90000))
	randNum := 10000 + n.Int64()
	return fmt.Sprintf("BK-%d-%05d", year, randNum)
}

// GetReservations lists reservations for this hotel with search & filters
// GET /api/admin/reservations
// GET /api/v1/hotel/reservations
func GetReservations(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	dateFrom := strings.TrimSpace(c.Query("dateFrom"))
	dateTo := strings.TrimSpace(c.Query("dateTo"))

	query := database.DB.Model(&models.Reservation{}).
		Preload("Room").
		Preload("Room.Category").
		Where("hotel_id = ?", hotelID)

	if status != "" && !strings.EqualFold(status, "all") {
		query = query.Where("status = ?", strings.ToLower(status))
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(booking_reference) LIKE ? OR LOWER(guest_name) LIKE ? OR LOWER(guest_email) LIKE ? OR guest_phone LIKE ?", s, s, s, s)
	}

	if dateFrom != "" {
		if tFrom, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("check_in >= ?", tFrom)
		}
	}

	if dateTo != "" {
		if tTo, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("check_in <= ?", tTo)
		}
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var resvs []models.Reservation
	if err := query.Order("check_in desc, created_at desc").Offset(offset).Limit(limit).Find(&resvs).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve reservations. Please try again.")
		return
	}

	dtos := make([]ReservationDTO, 0, len(resvs))
	for i := range resvs {
		dtos = append(dtos, buildReservationDTO(&resvs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         dtos,
		"reservations": dtos,
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

// GetReservationByID returns a single reservation
// GET /api/admin/reservations/:id
func GetReservationByID(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var resv models.Reservation
	if err := database.DB.Preload("Room").Preload("Room.Category").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	dto := buildReservationDTO(&resv)
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"data":        dto,
		"reservation": dto,
	})
}

// CreateReservationRequest defines payload for staff booking creation
type CreateReservationRequest struct {
	RoomID          string `json:"roomId"`
	GuestName       string `json:"guestName" binding:"required"`
	GuestEmail      string `json:"guestEmail" binding:"required,email"`
	GuestPhone      string `json:"guestPhone" binding:"required"`
	CheckIn         string `json:"checkIn" binding:"required"`
	CheckOut        string `json:"checkOut" binding:"required"`
	NumGuests       int    `json:"numGuests"`
	SpecialRequests string `json:"specialRequests"`
}

// CreateReservation creates a new reservation from the front desk
// POST /api/admin/reservations
func CreateReservation(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	checkIn, err := time.Parse("2006-01-02", req.CheckIn)
	if err != nil {
		util.SendError(c, http.StatusBadRequest, "Invalid checkIn date format (YYYY-MM-DD)")
		return
	}
	checkOut, err := time.Parse("2006-01-02", req.CheckOut)
	if err != nil {
		util.SendError(c, http.StatusBadRequest, "Invalid checkOut date format (YYYY-MM-DD)")
		return
	}

	if !checkOut.After(checkIn) {
		util.SendError(c, http.StatusBadRequest, "Check-out must be after check-in")
		return
	}

	// Validate check-in date is not in the past (allow today for walk-ins)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if checkIn.Before(today) {
		util.SendError(c, http.StatusBadRequest, "Check-in date cannot be in the past")
		return
	}

	nights := int(checkOut.Sub(checkIn).Hours() / 24)
	if nights < 1 {
		nights = 1
	}
	if nights > 365 {
		util.SendError(c, http.StatusBadRequest, "Reservation stay duration cannot exceed 365 nights")
		return
	}

	numGuests := req.NumGuests
	if numGuests < 1 {
		numGuests = 1
	}

	// 1. Resolve room
	var room models.Room
	if req.RoomID != "" {
		if err := database.DB.Preload("Category").
			Where("id = ? AND hotel_id = ?", req.RoomID, hotelID).
			First(&room).Error; err != nil {
			util.SendError(c, http.StatusBadRequest, "Selected room does not exist in this property")
			return
		}

		// Overlap check
		var overlapCount int64
		database.DB.Model(&models.Reservation{}).
			Where("room_id = ? AND status IN ('confirmed', 'checked_in') AND check_in < ? AND check_out > ?",
				room.ID, checkOut, checkIn).
			Count(&overlapCount)
		if overlapCount > 0 {
			util.SendError(c, http.StatusConflict, fmt.Sprintf("Room %s is already reserved for the selected dates", room.RoomNumber))
			return
		}
	} else {
		// Auto-pick available room
		var overlapIDs []string
		database.DB.Model(&models.Reservation{}).
			Where("hotel_id = ? AND status IN ('confirmed', 'checked_in') AND check_in < ? AND check_out > ? AND room_id IS NOT NULL",
				hotelID, checkOut, checkIn).
			Pluck("room_id", &overlapIDs)

		q := database.DB.Preload("Category").Where("hotel_id = ? AND status != 'out_of_service'", hotelID)
		if len(overlapIDs) > 0 {
			q = q.Where("id NOT IN ?", overlapIDs)
		}
		if err := q.Order("room_number asc").First(&room).Error; err != nil {
			util.SendError(c, http.StatusConflict, "No available rooms for the requested dates")
			return
		}
	}

	// Capacity check
	if room.Category != nil && room.Category.MaxOccupancy > 0 && numGuests > room.Category.MaxOccupancy {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Number of guests (%d) exceeds room maximum capacity of %d", numGuests, room.Category.MaxOccupancy))
		return
	}

	rate := room.Rate
	subtotal := math.Round(rate*float64(nights)*100) / 100
	taxRate := 0.12
	taxAmount := math.Round(subtotal*taxRate*100) / 100
	total := math.Round((subtotal+taxAmount)*100) / 100

	bookingRef := generateAdminBookingRef()

	resv := models.Reservation{
		BookingReference: bookingRef,
		HotelID:          hotelID,
		RoomID:           &room.ID,
		GuestName:        strings.TrimSpace(req.GuestName),
		GuestEmail:       strings.ToLower(strings.TrimSpace(req.GuestEmail)),
		GuestPhone:       strings.TrimSpace(req.GuestPhone),
		CheckIn:          checkIn,
		CheckOut:         checkOut,
		Nights:           nights,
		NumGuests:        numGuests,
		RoomRate:         rate,
		TotalAmount:      total,
		Status:           "confirmed",
		SpecialRequests:  strings.TrimSpace(req.SpecialRequests),
	}

	if err := database.DB.Create(&resv).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to complete reservation. Please try again.")
		return
	}

	// Create initial bill
	catName := "Room"
	if room.Category != nil {
		catName = room.Category.Name
	}

	bill := models.Bill{
		HotelID:       hotelID,
		ReservationID: resv.ID,
		GuestName:     resv.GuestName,
		GuestEmail:    resv.GuestEmail,
		GuestPhone:    resv.GuestPhone,
		RoomNumber:    room.RoomNumber,
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		Nights:        nights,
		Subtotal:      subtotal,
		TaxRate:       taxRate,
		TaxAmount:     taxAmount,
		TotalAmount:   total,
		Balance:       total,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	billItem := models.BillItem{
		BillID:      bill.ID,
		Description: fmt.Sprintf("%s – %d nights @ ₹%.2f/night", catName, nights, rate),
		Quantity:    nights,
		UnitPrice:   rate,
		Amount:      subtotal,
	}
	database.DB.Create(&billItem)

	resv.Room = &room
	dto := buildReservationDTO(&resv)

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"message":     "Reservation successfully created",
		"data":        dto,
		"reservation": dto,
	})
}

// UpdateReservation updates reservation details
// PUT /api/admin/reservations/:id
func UpdateReservation(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var resv models.Reservation
	if err := database.DB.Preload("Room").Preload("Room.Category").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	if resv.Status == "cancelled" || resv.Status == "checked_out" {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Cannot modify a reservation in '%s' status", resv.Status))
		return
	}

	var req CreateReservationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	if req.GuestName != "" {
		resv.GuestName = strings.TrimSpace(req.GuestName)
	}
	if req.GuestEmail != "" {
		resv.GuestEmail = strings.ToLower(strings.TrimSpace(req.GuestEmail))
	}
	if req.GuestPhone != "" {
		resv.GuestPhone = strings.TrimSpace(req.GuestPhone)
	}
	if req.SpecialRequests != "" {
		resv.SpecialRequests = strings.TrimSpace(req.SpecialRequests)
	}

	if err := database.DB.Save(&resv).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update reservation details. Please try again.")
		return
	}

	dto := buildReservationDTO(&resv)
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Reservation successfully updated",
		"data":        dto,
		"reservation": dto,
	})
}

// CheckIn transitions reservation to checked_in and room to occupied
// PATCH /api/admin/reservations/:id/check-in
// POST /api/admin/reservations/:id/checkin
func CheckIn(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var resv models.Reservation
	if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", id, hotelID).First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	if resv.Status != "confirmed" {
		if resv.Status == "cancelled" {
			util.SendError(c, http.StatusBadRequest, "Cannot check in a cancelled reservation")
			return
		}
		if resv.Status == "checked_in" {
			util.SendError(c, http.StatusBadRequest, "Guest is already checked in")
			return
		}
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Cannot check in reservation in '%s' status", resv.Status))
		return
	}

	// Verify room is not currently occupied by another guest or out of service
	if resv.RoomID != nil {
		var r models.Room
		if err := database.DB.Where("id = ? AND hotel_id = ?", *resv.RoomID, hotelID).First(&r).Error; err == nil {
			if r.Status == "occupied" {
				util.SendError(c, http.StatusConflict, fmt.Sprintf("Room %s is currently occupied by another guest", r.RoomNumber))
				return
			}
			if r.Status == "out_of_service" {
				util.SendError(c, http.StatusConflict, fmt.Sprintf("Room %s is out of service and unavailable for check-in", r.RoomNumber))
				return
			}
		}
	}

	resv.Status = "checked_in"
	database.DB.Model(&resv).Update("status", "checked_in")

	// Update physical room status to occupied
	if resv.RoomID != nil {
		database.DB.Model(&models.Room{}).Where("id = ?", *resv.RoomID).Update("status", "occupied")
	}

	// Audit log
	var uid *string
	if val, ok := c.Get("userID"); ok {
		if s, isStr := val.(string); isStr {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "check_in",
		Details: fmt.Sprintf("Guest %s checked in (%s)", resv.GuestName, resv.BookingReference),
	})

	database.DB.Preload("Room").Preload("Room.Category").First(&resv, "id = ?", resv.ID)
	dto := buildReservationDTO(&resv)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Guest successfully checked in",
		"data":        dto,
		"reservation": dto,
	})
}

// CheckOut transitions reservation to checked_out and room to dirty
// PATCH /api/admin/reservations/:id/check-out
// POST /api/admin/reservations/:id/checkout
func CheckOut(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var resv models.Reservation
	if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", id, hotelID).First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	if resv.Status != "checked_in" {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Cannot check out reservation in '%s' status", resv.Status))
		return
	}

	resv.Status = "checked_out"
	database.DB.Model(&resv).Update("status", "checked_out")

	// Update physical room status to dirty and log housekeeping task
	if resv.RoomID != nil {
		database.DB.Model(&models.Room{}).Where("id = ?", *resv.RoomID).Update("status", "dirty")

		hk := models.HousekeepingLog{
			RoomID:   *resv.RoomID,
			HotelID:  hotelID,
			HkStatus: "dirty",
			Notes:    fmt.Sprintf("Room vacated after checkout by %s (%s)", resv.GuestName, resv.BookingReference),
		}
		database.DB.Create(&hk)
	}

	var uid *string
	if val, ok := c.Get("userID"); ok {
		if s, isStr := val.(string); isStr {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "check_out",
		Details: fmt.Sprintf("Guest %s checked out (%s)", resv.GuestName, resv.BookingReference),
	})

	database.DB.Preload("Room").Preload("Room.Category").First(&resv, "id = ?", resv.ID)
	dto := buildReservationDTO(&resv)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Guest successfully checked out",
		"data":        dto,
		"reservation": dto,
	})
}

// CancelReservation cancels a reservation and frees the room
// PATCH /api/admin/reservations/:id/cancel
func CancelReservation(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var resv models.Reservation
	if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", id, hotelID).First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	if resv.Status == "checked_in" {
		util.SendError(c, http.StatusBadRequest, "Cannot cancel a reservation that is currently checked in. Please check out first.")
		return
	}
	if resv.Status == "checked_out" {
		util.SendError(c, http.StatusBadRequest, "Cannot cancel a reservation that has already checked out")
		return
	}
	if resv.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Reservation is already cancelled")
		return
	}

	resv.Status = "cancelled"
	database.DB.Model(&resv).Update("status", "cancelled")

	// Cancel associated unpaid bill folio if untouched
	var bill models.Bill
	if err := database.DB.Where("reservation_id = ? AND hotel_id = ?", resv.ID, hotelID).First(&bill).Error; err == nil {
		if bill.Status == "unpaid" && bill.AmountPaid == 0 {
			database.DB.Model(&bill).Update("status", "cancelled")
		}
	}

	// If room was reserved, ensure it's available
	if resv.RoomID != nil {
		database.DB.Model(&models.Room{}).
			Where("id = ? AND status = 'reserved'", *resv.RoomID).
			Update("status", "available")
	}

	var uid *string
	if val, ok := c.Get("userID"); ok {
		if s, isStr := val.(string); isStr {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "cancel_reservation",
		Details: fmt.Sprintf("Reservation %s cancelled for %s", resv.BookingReference, resv.GuestName),
	})

	database.DB.Preload("Room").Preload("Room.Category").First(&resv, "id = ?", resv.ID)
	dto := buildReservationDTO(&resv)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Reservation successfully cancelled",
		"data":        dto,
		"reservation": dto,
	})
}

// GetTodayCheckIns lists arrivals scheduled for today
// GET /api/admin/reservations/today/check-ins
func GetTodayCheckIns(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var resvs []models.Reservation
	database.DB.Preload("Room").Preload("Room.Category").
		Where("hotel_id = ? AND check_in = ? AND status = 'confirmed'", hotelID, today).
		Order("guest_name asc").
		Find(&resvs)

	dtos := make([]ReservationDTO, 0, len(resvs))
	for i := range resvs {
		dtos = append(dtos, buildReservationDTO(&resvs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"data":     dtos,
		"checkIns": dtos,
		"total":    len(dtos),
	})
}

// GetTodayCheckOuts lists departures scheduled for today
// GET /api/admin/reservations/today/check-outs
func GetTodayCheckOuts(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var resvs []models.Reservation
	database.DB.Preload("Room").Preload("Room.Category").
		Where("hotel_id = ? AND check_out = ? AND status = 'checked_in'", hotelID, today).
		Order("guest_name asc").
		Find(&resvs)

	dtos := make([]ReservationDTO, 0, len(resvs))
	for i := range resvs {
		dtos = append(dtos, buildReservationDTO(&resvs[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      dtos,
		"checkOuts": dtos,
		"total":     len(dtos),
	})
}
