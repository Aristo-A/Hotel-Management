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

// HousekeepingTaskDTO formats housekeeping task cards for the operational board
type HousekeepingTaskDTO struct {
	ID             string     `json:"id"`
	RoomID         string     `json:"roomId"`
	RoomNumber     string     `json:"roomNumber"`
	Floor          int        `json:"floor"`
	CategoryName   string     `json:"categoryName"`
	RoomStatus     string     `json:"roomStatus"`
	HkStatus       string     `json:"hkStatus"`   // clean, pending, in_progress, maintenance
	TaskStatus     string     `json:"taskStatus"` // clean, pending, in_progress, maintenance, completed
	AssignedTo     string     `json:"assignedTo"`
	LastCleaned    *time.Time `json:"lastCleaned,omitempty"`
	LastCheckedOut *time.Time `json:"lastCheckedOut,omitempty"`
	Notes          string     `json:"notes"`
	Priority       string     `json:"priority"` // high, medium, low
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// UpdateHousekeepingRequest defines payload for updating room housekeeping state
type UpdateHousekeepingRequest struct {
	TaskStatus string `json:"taskStatus"` // completed, in_progress, pending, maintenance, clean
	HkStatus   string `json:"hkStatus"`   // clean, pending, in_progress, maintenance
	AssignedTo string `json:"assignedTo"`
	Notes      string `json:"notes"`
}

// GetHousekeepingTasks returns active room housekeeping status and task assignments
// GET /api/admin/housekeeping
// GET /api/v1/hotel/housekeeping
func GetHousekeepingTasks(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	statusFilter := strings.TrimSpace(c.Query("status"))
	floorFilter := strings.TrimSpace(c.Query("floor"))
	search := strings.TrimSpace(c.Query("search"))

	// Fetch all rooms for this hotel
	query := database.DB.Preload("Category").Where("hotel_id = ?", hotelID)
	if floorFilter != "" {
		if f, err := strconv.Atoi(floorFilter); err == nil {
			query = query.Where("floor = ?", f)
		}
	}
	if search != "" {
		query = query.Where("room_number LIKE ?", "%"+search+"%")
	}

	var rooms []models.Room
	if err := query.Order("floor asc, room_number asc").Find(&rooms).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve rooms for housekeeping. Please try again.")
		return
	}

	tasks := make([]HousekeepingTaskDTO, 0, len(rooms))
	for _, room := range rooms {
		catName := "Standard Room"
		if room.Category != nil {
			catName = room.Category.Name
		}

		// Find the latest housekeeping log for this room
		var latestLog models.HousekeepingLog
		hasLog := database.DB.Where("room_id = ? AND hotel_id = ?", room.ID, hotelID).
			Order("created_at desc").
			First(&latestLog).Error == nil

		// Derive hkStatus: clean, pending, in_progress, maintenance
		hkStatus := "clean"
		assignedTo := "Unassigned"
		notes := room.Notes
		priority := "low"

		switch room.Status {
		case "dirty":
			hkStatus = "pending"
			priority = "high"
		case "cleaning":
			hkStatus = "in_progress"
			priority = "medium"
		case "out_of_service":
			hkStatus = "maintenance"
			priority = "medium"
		}

		var lastCleaned *time.Time
		var lastCheckedOut *time.Time
		if hasLog {
			if latestLog.AssignedTo != "" {
				assignedTo = latestLog.AssignedTo
			}
			if latestLog.Notes != "" {
				notes = latestLog.Notes
			}
			if latestLog.HkStatus == "clean" || latestLog.HkStatus == "completed" {
				t := latestLog.CreatedAt
				lastCleaned = &t
			}
		}

		// Check last checkout date
		var lastResv models.Reservation
		if err := database.DB.Where("room_id = ? AND status = 'checked_out'", room.ID).
			Order("check_out desc").
			First(&lastResv).Error; err == nil {
			t := lastResv.CheckOut
			lastCheckedOut = &t
		}

		// Filter by status if specified
		if statusFilter != "" && statusFilter != "all" && statusFilter != "All" {
			normStatus := strings.ToLower(statusFilter)
			if normStatus == "needs cleaning" {
				normStatus = "pending"
			}
			if normStatus == "in progress" {
				normStatus = "in_progress"
			}
			if normStatus != hkStatus && normStatus != room.Status {
				continue
			}
		}

		taskID := fmt.Sprintf("hk-%s", room.ID)
		if hasLog {
			taskID = latestLog.ID
		}

		tasks = append(tasks, HousekeepingTaskDTO{
			ID:             taskID,
			RoomID:         room.ID,
			RoomNumber:     room.RoomNumber,
			Floor:          room.Floor,
			CategoryName:   catName,
			RoomStatus:     room.Status,
			HkStatus:       hkStatus,
			TaskStatus:     hkStatus,
			AssignedTo:     assignedTo,
			LastCleaned:    lastCleaned,
			LastCheckedOut: lastCheckedOut,
			Notes:          notes,
			Priority:       priority,
			UpdatedAt:      room.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tasks,
		"tasks":   tasks,
		"total":   len(tasks),
	})
}

// UpdateHousekeepingTask updates a room's housekeeping status and logs the event
// PATCH /api/admin/housekeeping/:roomId
// PATCH /api/v1/hotel/housekeeping/:roomId
func UpdateHousekeepingTask(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	roomIdentifier := c.Param("roomId")
	if roomIdentifier == "" {
		util.SendError(c, http.StatusBadRequest, "Room ID or room number required")
		return
	}

	var req UpdateHousekeepingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	// Support looking up room by ID or RoomNumber
	var room models.Room
	if err := database.DB.Where("hotel_id = ? AND (id = ? OR room_number = ?)", hotelID, roomIdentifier, roomIdentifier).
		First(&room).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	// Normalize requested status
	statusInput := strings.ToLower(strings.TrimSpace(req.HkStatus))
	if statusInput == "" {
		statusInput = strings.ToLower(strings.TrimSpace(req.TaskStatus))
	}

	newRoomStatus := room.Status
	newHkStatus := "clean"

	switch statusInput {
	case "completed", "clean":
		if room.Status == "occupied" {
			newRoomStatus = "occupied"
		} else {
			newRoomStatus = "available"
		}
		newHkStatus = "clean"
	case "in_progress", "cleaning":
		newRoomStatus = "cleaning"
		newHkStatus = "in_progress"
	case "pending", "dirty", "needs cleaning":
		newRoomStatus = "dirty"
		newHkStatus = "pending"
	case "maintenance", "out_of_service":
		newRoomStatus = "out_of_service"
		newHkStatus = "maintenance"
	default:
		// Keep room status if unspecified
		switch room.Status {
		case "dirty":
			newHkStatus = "pending"
		case "cleaning":
			newHkStatus = "in_progress"
		}
	}

	// Update room status
	oldStatus := room.Status
	room.Status = newRoomStatus
	if req.Notes != "" {
		room.Notes = req.Notes
	}
	if err := database.DB.Save(&room).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update room housekeeping status. Please try again.")
		return
	}

	// Create HousekeepingLog audit entry
	assigned := strings.TrimSpace(req.AssignedTo)
	if assigned == "" {
		assigned = "Housekeeping Staff"
	}
	hkLog := models.HousekeepingLog{
		RoomID:     room.ID,
		HotelID:    hotelID,
		HkStatus:   newHkStatus,
		AssignedTo: assigned,
		Notes:      req.Notes,
	}
	database.DB.Create(&hkLog)

	// Record activity log
	var uid *string
	if val, exists := c.Get("userID"); exists {
		if s, ok := val.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "HOUSEKEEPING_UPDATE",
		Details: fmt.Sprintf("Room %s status transitioned from '%s' to '%s' (Assigned to: %s)", room.RoomNumber, oldStatus, newRoomStatus, assigned),
	})

	now := time.Now()
	dto := HousekeepingTaskDTO{
		ID:         hkLog.ID,
		RoomID:     room.ID,
		RoomNumber: room.RoomNumber,
		Floor:      room.Floor,
		RoomStatus: room.Status,
		HkStatus:   newHkStatus,
		TaskStatus: newHkStatus,
		AssignedTo: assigned,
		Notes:      req.Notes,
		UpdatedAt:  now,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Housekeeping task updated successfully",
		"data":    dto,
		"task":    dto,
	})
}
