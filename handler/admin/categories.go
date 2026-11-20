package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// CategoryDTO formats room categories for clean JSON consumption
type CategoryDTO struct {
	ID           string    `json:"id"`
	HotelID      string    `json:"hotelId"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	BaseRate     float64   `json:"baseRate"`
	DefaultRate  float64   `json:"defaultRate"`
	MaxOccupancy int       `json:"maxOccupancy"`
	BedType      string    `json:"bedType"`
	Size         int       `json:"size"`
	RoomSize     int       `json:"roomSize"`
	Amenities    []string  `json:"amenities"`
	Image        string    `json:"image"`
	Images       []string  `json:"images"`
	TotalRooms   int64     `json:"totalRooms"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Helper to convert models.RoomCategory into CategoryDTO
func buildCategoryDTO(cat *models.RoomCategory) CategoryDTO {
	var totalRooms int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND category_id = ?", cat.HotelID, cat.ID).Count(&totalRooms)

	// Parse amenities
	amenities := make([]string, 0)
	if cat.Amenities != "" {
		if strings.HasPrefix(strings.TrimSpace(cat.Amenities), "[") {
			_ = json.Unmarshal([]byte(cat.Amenities), &amenities)
		} else {
			parts := strings.Split(cat.Amenities, ",")
			for _, p := range parts {
				trimmed := strings.TrimSpace(p)
				if trimmed != "" {
					amenities = append(amenities, trimmed)
				}
			}
		}
	}

	// Parse images
	images := make([]string, 0)
	mainImg := ""
	if cat.Images != "" {
		if strings.HasPrefix(strings.TrimSpace(cat.Images), "[") {
			_ = json.Unmarshal([]byte(cat.Images), &images)
			if len(images) > 0 {
				mainImg = images[0]
			}
		} else {
			mainImg = strings.TrimSpace(cat.Images)
			images = append(images, mainImg)
		}
	}

	size := cat.RoomSize
	if size <= 0 {
		size = 30
	}

	rate := cat.DefaultRate
	if rate <= 0 {
		rate = cat.BaseRate
	}

	return CategoryDTO{
		ID:           cat.ID,
		HotelID:      cat.HotelID,
		Name:         cat.Name,
		Description:  cat.Description,
		BaseRate:     rate,
		DefaultRate:  rate,
		MaxOccupancy: cat.MaxOccupancy,
		BedType:      cat.BedType,
		Size:         size,
		RoomSize:     size,
		Amenities:    amenities,
		Image:        mainImg,
		Images:       images,
		TotalRooms:   totalRooms,
		CreatedAt:    cat.CreatedAt,
	}
}

// CategoryRequest defines incoming JSON for creating or updating a category
type CategoryRequest struct {
	Name         string      `json:"name" binding:"required"`
	Description  string      `json:"description"`
	DefaultRate  float64     `json:"defaultRate"`
	BaseRate     float64     `json:"baseRate"`
	MaxOccupancy int         `json:"maxOccupancy"`
	BedType      string      `json:"bedType"`
	RoomSize     int         `json:"roomSize"`
	Size         int         `json:"size"`
	Amenities    interface{} `json:"amenities"`
	Images       interface{} `json:"images"`
	Image        string      `json:"image"`
}

// parseAmenitiesToString converts incoming string or array to JSON or clean text
func parseAmenitiesToString(val interface{}) string {
	if val == nil {
		return "[]"
	}
	switch v := val.(type) {
	case []interface{}:
		strs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				strs = append(strs, strings.TrimSpace(s))
			}
		}
		bytes, _ := json.Marshal(strs)
		return string(bytes)
	case []string:
		bytes, _ := json.Marshal(v)
		return string(bytes)
	case string:
		parts := strings.Split(v, ",")
		strs := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				strs = append(strs, trimmed)
			}
		}
		bytes, _ := json.Marshal(strs)
		return string(bytes)
	default:
		return "[]"
	}
}

// parseImagesToString converts incoming string or array to JSON
func parseImagesToString(val interface{}, singleImage string) string {
	if singleImage != "" && (val == nil || val == "") {
		bytes, _ := json.Marshal([]string{strings.TrimSpace(singleImage)})
		return string(bytes)
	}
	if val == nil {
		if singleImage != "" {
			bytes, _ := json.Marshal([]string{strings.TrimSpace(singleImage)})
			return string(bytes)
		}
		return "[]"
	}
	switch v := val.(type) {
	case []interface{}:
		strs := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				strs = append(strs, strings.TrimSpace(s))
			}
		}
		bytes, _ := json.Marshal(strs)
		return string(bytes)
	case []string:
		bytes, _ := json.Marshal(v)
		return string(bytes)
	case string:
		if strings.HasPrefix(strings.TrimSpace(v), "[") {
			return v
		}
		bytes, _ := json.Marshal([]string{strings.TrimSpace(v)})
		return string(bytes)
	default:
		return "[]"
	}
}

// GetCategories returns all room categories for the authenticated hotel admin's property
// GET /api/admin/room-categories
// GET /api/v1/hotel/categories
func GetCategories(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var categories []models.RoomCategory
	if err := database.DB.Where("hotel_id = ?", hotelID).Order("default_rate asc").Find(&categories).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve room categories. Please try again.")
		return
	}

	dtos := make([]CategoryDTO, 0, len(categories))
	for i := range categories {
		dtos = append(dtos, buildCategoryDTO(&categories[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       dtos,
		"categories": dtos,
	})
}

// GetCategoryByID returns single room category
// GET /api/admin/room-categories/:id
// GET /api/v1/hotel/categories/:id
func GetCategoryByID(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var cat models.RoomCategory
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&cat).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room category not found")
		return
	}

	dto := buildCategoryDTO(&cat)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"data":     dto,
		"category": dto,
	})
}

// CreateCategory registers a new room tier/category for the property
// POST /api/admin/room-categories
// POST /api/v1/hotel/categories
func CreateCategory(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	cleanName := strings.TrimSpace(req.Name)
	if cleanName == "" {
		util.SendError(c, http.StatusBadRequest, "Category name is required")
		return
	}

	// 1. Verify unique name within this property
	var count int64
	database.DB.Model(&models.RoomCategory{}).Where("hotel_id = ? AND LOWER(name) = ?", hotelID, strings.ToLower(cleanName)).Count(&count)
	if count > 0 {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("A room category named '%s' already exists for this property", cleanName))
		return
	}

	rate := req.DefaultRate
	if rate <= 0 {
		rate = req.BaseRate
	}
	if rate <= 0 {
		util.SendError(c, http.StatusBadRequest, "A valid positive rate is required")
		return
	}

	maxOcc := req.MaxOccupancy
	if maxOcc <= 0 {
		maxOcc = 2
	}
	if maxOcc > 20 {
		util.SendError(c, http.StatusBadRequest, "Maximum occupancy cannot exceed 20 guests")
		return
	}

	bed := strings.TrimSpace(req.BedType)
	if bed == "" {
		bed = "King"
	}

	size := req.RoomSize
	if size <= 0 {
		size = req.Size
	}
	if size <= 0 {
		size = 30
	}

	amenitiesStr := parseAmenitiesToString(req.Amenities)
	imagesStr := parseImagesToString(req.Images, req.Image)

	category := models.RoomCategory{
		HotelID:      hotelID,
		Name:         cleanName,
		Description:  strings.TrimSpace(req.Description),
		DefaultRate:  rate,
		MaxOccupancy: maxOcc,
		BedType:      bed,
		RoomSize:     size,
		Amenities:    amenitiesStr,
		Images:       imagesStr,
	}

	if err := database.DB.Create(&category).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to create room category. Please try again.")
		return
	}

	// Log activity
	var uid *string
	if uVal, exists := c.Get("userID"); exists {
		if s, ok := uVal.(string); ok && s != "" {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "create_room_category",
		Details: fmt.Sprintf("Created room category '%s' (Base rate: ₹%.2f)", category.Name, category.DefaultRate),
	})

	dto := buildCategoryDTO(&category)
	c.JSON(http.StatusCreated, gin.H{
		"success":  true,
		"message":  "Room category successfully created",
		"data":     dto,
		"category": dto,
	})
}

// UpdateCategory modifies an existing room tier
// PUT /api/admin/room-categories/:id
// PUT /api/v1/hotel/categories/:id
func UpdateCategory(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var cat models.RoomCategory
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&cat).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room category not found")
		return
	}

	var req CategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	cleanName := strings.TrimSpace(req.Name)
	if cleanName != "" && !strings.EqualFold(cat.Name, cleanName) {
		var count int64
		database.DB.Model(&models.RoomCategory{}).
			Where("hotel_id = ? AND LOWER(name) = ? AND id != ?", hotelID, strings.ToLower(cleanName), id).
			Count(&count)
		if count > 0 {
			util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Another category named '%s' already exists", cleanName))
			return
		}
		cat.Name = cleanName
	}

	if req.Description != "" {
		cat.Description = strings.TrimSpace(req.Description)
	}

	if req.DefaultRate < 0 || req.BaseRate < 0 {
		util.SendError(c, http.StatusBadRequest, "Room category rate cannot be negative")
		return
	}

	rate := req.DefaultRate
	if rate <= 0 {
		rate = req.BaseRate
	}
	if rate > 0 {
		cat.DefaultRate = rate
	}

	if req.MaxOccupancy > 20 {
		util.SendError(c, http.StatusBadRequest, "Maximum occupancy cannot exceed 20 guests")
		return
	}
	if req.MaxOccupancy > 0 {
		cat.MaxOccupancy = req.MaxOccupancy
	}
	if req.BedType != "" {
		cat.BedType = strings.TrimSpace(req.BedType)
	}
	if req.RoomSize > 0 {
		cat.RoomSize = req.RoomSize
	} else if req.Size > 0 {
		cat.RoomSize = req.Size
	}

	if req.Amenities != nil {
		cat.Amenities = parseAmenitiesToString(req.Amenities)
	}
	if req.Images != nil || req.Image != "" {
		cat.Images = parseImagesToString(req.Images, req.Image)
	}

	if err := database.DB.Save(&cat).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to update room category. Please try again.")
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
		Action:  "update_room_category",
		Details: fmt.Sprintf("Updated room category '%s'", cat.Name),
	})

	dto := buildCategoryDTO(&cat)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Room category successfully updated",
		"data":     dto,
		"category": dto,
	})
}

// DeleteCategory removes a room category if not in use by any rooms
// DELETE /api/admin/room-categories/:id
// DELETE /api/v1/hotel/categories/:id
func DeleteCategory(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var cat models.RoomCategory
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&cat).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Room category not found")
		return
	}

	// Conflict check: cannot delete category if rooms currently belong to it
	var roomCount int64
	database.DB.Model(&models.Room{}).Where("hotel_id = ? AND category_id = ?", hotelID, id).Count(&roomCount)
	if roomCount > 0 {
		util.SendError(c, http.StatusConflict, fmt.Sprintf("Cannot delete category '%s' because %d rooms currently reference it", cat.Name, roomCount))
		return
	}

	if err := database.DB.Delete(&cat).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to delete room category. Please try again.")
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
		Action:  "delete_room_category",
		Details: fmt.Sprintf("Deleted room category '%s'", cat.Name),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Room category successfully deleted",
		"id":      id,
	})
}
