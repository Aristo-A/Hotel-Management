package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// RoomDTO formats room objects for frontend consumption
type RoomDTO struct {
	ID           string       `json:"id"`
	HotelID      string       `json:"hotelId"`
	RoomNumber   string       `json:"roomNumber"`
	Floor        int          `json:"floor"`
	CategoryID   string       `json:"categoryId"`
	CategoryName string       `json:"categoryName"`
	Rate         float64      `json:"rate"`
	Status       string       `json:"status"`
	Notes        string       `json:"notes"`
	Description  string       `json:"description,omitempty"`
	CurrentGuest *string      `json:"currentGuest"`
	Category     *CategoryDTO `json:"category,omitempty"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

var validRoomStatuses = map[string]bool{
	"available":      true,
	"occupied":       true,
	"reserved":       true,
	"dirty":          true,
	"cleaning":       true,
	"out_of_service": true,
}

// buildRoomDTO populates category name and current active guest details
func buildRoomDTO(room *models.Room) RoomDTO {
	catName := ""
	var catDTO *CategoryDTO

	if room.Category != nil {
		catName = room.Category.Name
		dto := buildCategoryDTO(room.Category)
		catDTO = &dto
	} else if room.CategoryID != "" {
		var cat models.RoomCategory
		if err := database.DB.Where("id = ?", room.CategoryID).First(&cat).Error; err == nil {
			catName = cat.Name
			dto := buildCategoryDTO(&cat)
			catDTO = &dto
		}
	}

	var currentGuest *string
	if room.Status == "occupied" || room.Status == "reserved" {
		var resv models.Reservation
		if err := database.DB.Where("room_id = ? AND status IN ('checked_in', 'confirmed')", room.ID).
			Order("check_in desc").
			First(&resv).Error; err == nil {
			currentGuest = &resv.GuestName
		}
	}

	notes := room.Notes
	if notes == "" {
		notes = room.Description
	}

	return RoomDTO{
		ID:           room.ID,
		HotelID:      room.HotelID,
		RoomNumber:   room.RoomNumber,
		Floor:        room.Floor,
		CategoryID:   room.CategoryID,
		CategoryName: catName,
		Rate:         room.Rate,
		Status:       room.Status,
		Notes:        notes,
		Description:  notes,
		CurrentGuest: currentGuest,
		Category:     catDTO,
		CreatedAt:    room.CreatedAt,
		UpdatedAt:    room.UpdatedAt,
	}
}

// GetRooms lists all rooms for the authenticated property with search and filtering
// GET /api/admin/rooms
// GET /api/v1/hotel/rooms
func GetRooms(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 || limit > 200 {
		limit = 50
	}

	status := strings.TrimSpace(c.Query("status"))
	floorStr := strings.TrimSpace(c.Query("floor"))
	categoryID := strings.TrimSpace(c.Query("categoryId"))
	search := strings.TrimSpace(c.Query("search"))

	query := database.DB.Model(&models.Room{}).Where("hotel_id = ?", hotelID)

	if status != "" && !strings.EqualFold(status, "all") {
		query = query.Where("status = ?", strings.ToLower(status))
	}

	if floorStr != "" {
		if f, err := strconv.Atoi(floorStr); err == nil {
			query = query.Where("floor = ?", f)
		}
	}

	if categoryID != "" && !strings.EqualFold(categoryID, "all") {
		query = query.Where("category_id = ?", categoryID)
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(room_number) LIKE ? OR LOWER(notes) LIKE ? OR LOWER(description) LIKE ?", s, s, s)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var rooms []models.Room
	if err := query.Preload("Category").
		Order("floor asc, room_number asc").
		Offset(offset).
		Limit(limit).
		Find(&rooms).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve room inventory. Please try again.")
		return
	}

	dtos := make([]RoomDTO, 0, len(rooms))
	for i := range rooms {
		dtos = append(dtos, buildRoomDTO(&rooms[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dtos,
		"rooms":   dtos,
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

// GetRoomByID retrieves a single room with category and current reservation details
// GET /api/admin/rooms/:id
// GET /api/v1/hotel/rooms/:id
func GetRoomByID(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var room models.Room
	if err := database.DB.Preload("Category").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&room).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	dto := buildRoomDTO(&room)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"room":    dto,
	})
}

// CreateRoomRequest defines incoming body for room creation
type CreateRoomRequest struct {
	RoomNumber  string  `json:"roomNumber" binding:"required"`
	Floor       int     `json:"floor" binding:"required"`
	CategoryID  string  `json:"categoryId" binding:"required"`
	Rate        float64 `json:"rate"`
	Status      string  `json:"status"`
	Notes       string  `json:"notes"`
	Description string  `json:"description"`
}

// CreateRoom adds a new physical room to the hotel inventory
// POST /api/admin/rooms
// POST /api/v1/hotel/rooms
func CreateRoom(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	cleanNumber := strings.TrimSpace(req.RoomNumber)
	if cleanNumber == "" {
		util.SendError(c, http.StatusBadRequest, "Room number is required")
		return
	}

	if req.Floor < 0 || req.Floor > 200 {
		util.SendError(c, http.StatusBadRequest, "Floor must be between 0 and 200")
		return
	}
	if req.Rate < 0 {
		util.SendError(c, http.StatusBadRequest, "Room rate cannot be negative")
		return
	}

	// 1. Verify category exists for this hotel
	var cat models.RoomCategory
	if err := database.DB.Where("id = ? AND hotel_id = ?", req.CategoryID, hotelID).First(&cat).Error; err != nil {
		util.SendError(c, http.StatusBadRequest, "Selected room category does not exist for this property")
		return
	}

	// 2. Verify room number uniqueness within this hotel
	var count int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND room_number = ?", hotelID, cleanNumber).Count(&count)
	if count > 0 {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Room number '%s' already exists for this property", cleanNumber))
		return
	}

	rate := req.Rate
	if rate <= 0 {
		rate = cat.DefaultRate
	}
	if rate <= 0 {
		util.SendError(c, http.StatusBadRequest, "Room rate must be greater than zero")
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = "available"
	} else if !validRoomStatuses[status] {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid room status '%s'", status))
		return
	}

	notes := strings.TrimSpace(req.Notes)
	if notes == "" {
		notes = strings.TrimSpace(req.Description)
	}

	room := models.Room{
		HotelID:     hotelID,
		CategoryID:  cat.ID,
		RoomNumber:  cleanNumber,
		Floor:       req.Floor,
		Rate:        rate,
		Status:      status,
		Notes:       notes,
		Description: notes,
	}

	if err := database.DB.Create(&room).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to create room. Please try again.")
		return
	}

	// Housekeeping log if initial status is dirty or cleaning
	if status == "dirty" || status == "cleaning" {
		hk := models.HousekeepingLog{
			RoomID:   room.ID,
			HotelID:  hotelID,
			HkStatus: status,
			Notes:    "Initial room setup",
		}
		database.DB.Create(&hk)
	}

	// Activity log
	var uid *string
	if uVal, exists := c.Get("userID"); exists {
		if s, ok := uVal.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "create_room",
		Details: fmt.Sprintf("Created room %s on Floor %d (%s, ₹%.2f)", room.RoomNumber, room.Floor, cat.Name, room.Rate),
	})

	room.Category = &cat
	dto := buildRoomDTO(&room)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Room successfully created",
		"data":    dto,
		"room":    dto,
	})
}

// UpdateRoom updates room specifications (rate, category, floor, notes)
// PUT /api/admin/rooms/:id
// PUT /api/v1/hotel/rooms/:id
func UpdateRoom(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var room models.Room
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&room).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	cleanNumber := strings.TrimSpace(req.RoomNumber)
	if cleanNumber != "" && cleanNumber != room.RoomNumber {
		var count int64
		database.DB.Model(&models.Room{}).
			Where("hotel_id = ? AND room_number = ? AND id != ?", hotelID, cleanNumber, id).
			Count(&count)
		if count > 0 {
			util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Room number '%s' already exists for this property", cleanNumber))
			return
		}
		room.RoomNumber = cleanNumber
	}

	if req.Floor >= 1 {
		room.Floor = req.Floor
	}

	if req.CategoryID != "" && req.CategoryID != room.CategoryID {
		var cat models.RoomCategory
		if err := database.DB.Where("id = ? AND hotel_id = ?", req.CategoryID, hotelID).First(&cat).Error; err != nil {
			util.SendError(c, http.StatusBadRequest, "Selected room category does not exist for this property")
			return
		}
		room.CategoryID = cat.ID
	}

	if req.Rate < 0 {
		util.SendError(c, http.StatusBadRequest, "Room rate cannot be negative")
		return
	}
	if req.Rate > 0 {
		room.Rate = req.Rate
	}

	if req.Status != "" {
		st := strings.ToLower(strings.TrimSpace(req.Status))
		if validRoomStatuses[st] {
			room.Status = st
		}
	}

	notes := strings.TrimSpace(req.Notes)
	if notes == "" {
		notes = strings.TrimSpace(req.Description)
	}
	if notes != "" {
		room.Notes = notes
		room.Description = notes
	}

	if err := database.DB.Save(&room).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update room details. Please try again.")
		return
	}

	var uid *string
	if uVal, exists := c.Get("userID"); exists {
		if s, ok := uVal.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "update_room",
		Details: fmt.Sprintf("Updated room %s", room.RoomNumber),
	})

	database.DB.Preload("Category").First(&room, "id = ?", room.ID)
	dto := buildRoomDTO(&room)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Room successfully updated",
		"data":    dto,
		"room":    dto,
	})
}

// RoomStatusRequest defines payload for updating room operational status
type RoomStatusRequest struct {
	Status     string `json:"status" binding:"required"`
	Notes      string `json:"notes"`
	AssignedTo string `json:"assignedTo"`
}

// UpdateRoomStatus updates the room state machine and records housekeeping logs
// PATCH /api/admin/rooms/:id/status
// PATCH /api/v1/hotel/rooms/:id/status
func UpdateRoomStatus(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var room models.Room
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&room).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	var req RoomStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	newStatus := strings.ToLower(strings.TrimSpace(req.Status))
	if !validRoomStatuses[newStatus] {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid room status '%s'. Valid statuses: available, occupied, reserved, dirty, cleaning, out_of_service", newStatus))
		return
	}

	oldStatus := room.Status
	room.Status = newStatus
	if err := database.DB.Model(&room).Update("status", newStatus).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update room operational status. Please try again.")
		return
	}

	// Record in housekeeping logs for operational statuses
	if newStatus == "dirty" || newStatus == "cleaning" || newStatus == "available" {
		hk := models.HousekeepingLog{
			RoomID:     room.ID,
			HotelID:    hotelID,
			HkStatus:   newStatus,
			AssignedTo: strings.TrimSpace(req.AssignedTo),
			Notes:      strings.TrimSpace(req.Notes),
		}
		database.DB.Create(&hk)
	}

	// Activity log
	var uid *string
	if uVal, exists := c.Get("userID"); exists {
		if s, ok := uVal.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "update_room_status",
		Details: fmt.Sprintf("Changed Room %s status from %s to %s", room.RoomNumber, oldStatus, newStatus),
	})

	database.DB.Preload("Category").First(&room, "id = ?", room.ID)
	dto := buildRoomDTO(&room)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Room status updated to %s", newStatus),
		"data":    dto,
		"room":    dto,
	})
}

// DeleteRoom removes a room from inventory if no active/upcoming reservations exist
// DELETE /api/admin/rooms/:id
// DELETE /api/v1/hotel/rooms/:id
func DeleteRoom(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var room models.Room
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&room).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	// Conflict check: cannot delete room with active reservations
	var activeReservations int64
	database.DB.Model(&models.Reservation{}).
		Where("room_id = ? AND status IN ('confirmed', 'checked_in')", id).
		Count(&activeReservations)

	if activeReservations > 0 {
		util.SendError(c, http.StatusConflict, fmt.Sprintf("Cannot delete Room %s because it has active or upcoming reservations", room.RoomNumber))
		return
	}

	if err := database.DB.Delete(&room).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to delete room. Please try again.")
		return
	}

	var uid *string
	if uVal, exists := c.Get("userID"); exists {
		if s, ok := uVal.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "delete_room",
		Details: fmt.Sprintf("Deleted Room %s", room.RoomNumber),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Room successfully deleted",
		"id":      id,
	})
}
