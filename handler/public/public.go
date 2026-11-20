package public

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

var defaultHotelAmenities = []string{
	"High-Speed Free WiFi",
	"Infinity Swimming Pool",
	"24/7 Room Service",
	"Fine Dining Restaurant & Bar",
	"Fitness & Wellness Center",
	"Luxury Spa & Sauna",
	"Executive Business Lounge",
	"Complimentary Valet Parking",
	"Daily Housekeeping",
}

// PublicHotelDTO formats public hotel details
type PublicHotelDTO struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Address      string   `json:"address"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	Country      string   `json:"country"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Website      string   `json:"website"`
	Image        string   `json:"image"`
	ImageURL     string   `json:"imageUrl"`
	StarRating   int      `json:"starRating"`
	Rating       float64  `json:"rating"`
	CheckInTime  string   `json:"checkInTime"`
	CheckOutTime string   `json:"checkOutTime"`
	Amenities    []string `json:"amenities"`
}

// PublicCategoryDTO formats room categories for guests
type PublicCategoryDTO struct {
	ID             string   `json:"id"`
	HotelID        string   `json:"hotelId"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	DefaultRate    float64  `json:"defaultRate"`
	BaseRate       float64  `json:"baseRate"`
	MaxOccupancy   int      `json:"maxOccupancy"`
	BedType        string   `json:"bedType"`
	RoomSize       int      `json:"roomSize"`
	Size           int      `json:"size"`
	Amenities      []string `json:"amenities"`
	Image          string   `json:"image"`
	Images         []string `json:"images"`
	AvailableRooms int64    `json:"availableRooms"`
}

// PublicRoomDTO formats available rooms
type PublicRoomDTO struct {
	ID         string             `json:"id"`
	HotelID    string             `json:"hotelId"`
	RoomNumber string             `json:"roomNumber"`
	Floor      int                `json:"floor"`
	Rate       float64            `json:"rate"`
	Status     string             `json:"status"`
	CategoryID string             `json:"categoryId"`
	Category   *PublicCategoryDTO `json:"category,omitempty"`
}

// BookingResponseDTO formats confirmed reservation
type BookingResponseDTO struct {
	ID               string    `json:"id"`
	BookingReference string    `json:"bookingReference"`
	Status           string    `json:"status"`
	HotelID          string    `json:"hotelId"`
	HotelName        string    `json:"hotelName"`
	RoomID           string    `json:"roomId"`
	RoomNumber       string    `json:"roomNumber"`
	CategoryID       string    `json:"categoryId"`
	RoomCategory     string    `json:"roomCategory"`
	CategoryName     string    `json:"categoryName"`
	GuestName        string    `json:"guestName"`
	GuestEmail       string    `json:"guestEmail"`
	GuestPhone       string    `json:"guestPhone"`
	CheckIn          string    `json:"checkIn"`
	CheckOut         string    `json:"checkOut"`
	Nights           int       `json:"nights"`
	NumGuests        int       `json:"numGuests"`
	Guests           int       `json:"guests"`
	RatePerNight     float64   `json:"ratePerNight"`
	RoomRate         float64   `json:"roomRate"`
	Subtotal         float64   `json:"subtotal"`
	TaxRate          float64   `json:"taxRate"`
	TaxAmount        float64   `json:"taxAmount"`
	TotalAmount      float64   `json:"totalAmount"`
	GrandTotal       float64   `json:"grandTotal"`
	Currency         string    `json:"currency"`
	SpecialRequests  string    `json:"specialRequests,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

// helper to resolve hotel (by ID or default to first active)
func resolveHotel(hotelID string) (*models.Hotel, error) {
	var hotel models.Hotel
	if hotelID != "" && hotelID != "default" && hotelID != "current" {
		if err := database.DB.Where("id = ?", hotelID).First(&hotel).Error; err != nil {
			return nil, err
		}
		return &hotel, nil
	}

	if err := database.DB.Where("status = 'active'").First(&hotel).Error; err != nil {
		if errFallback := database.DB.First(&hotel).Error; errFallback != nil {
			return nil, errFallback
		}
	}
	return &hotel, nil
}

// helper to generate unique BK-YYYY-XXXXX reference
func generateBookingReference() string {
	year := time.Now().Year()
	n, _ := rand.Int(rand.Reader, big.NewInt(90000))
	randNum := 10000 + n.Int64()
	return fmt.Sprintf("BK-%d-%05d", year, randNum)
}

// GetHotel retrieves public details for a hotel
// GET /api/public/hotels/:hotelId
// GET /api/v1/public/hotel
// GET /api/v1/public/hotels/:hotelId
func GetHotel(c *gin.Context) {
	hotelID := c.Param("hotelId")
	hotel, err := resolveHotel(hotelID)
	if err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel property not found")
		return
	}

	img := hotel.ImageURL
	if img == "" {
		img = hotel.Image
	}

	dto := PublicHotelDTO{
		ID:           hotel.ID,
		Name:         hotel.Name,
		Description:  hotel.Description,
		Address:      hotel.Address,
		City:         hotel.City,
		State:        hotel.State,
		Country:      hotel.Country,
		Phone:        hotel.Phone,
		Email:        hotel.Email,
		Website:      hotel.Website,
		Image:        img,
		ImageURL:     img,
		StarRating:   hotel.StarRating,
		Rating:       float64(hotel.StarRating),
		CheckInTime:  hotel.CheckInTime,
		CheckOutTime: hotel.CheckOutTime,
		Amenities:    defaultHotelAmenities,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"hotel":   dto,
	})
}

// GetRooms retrieves the public room category catalog
// GET /api/public/hotels/:hotelId/categories
// GET /api/public/hotels/:hotelId/rooms
func GetRooms(c *gin.Context) {
	hotelID := c.Param("hotelId")
	hotel, err := resolveHotel(hotelID)
	if err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel property not found")
		return
	}

	var categories []models.RoomCategory
	if err := database.DB.Where("hotel_id = ?", hotel.ID).Order("default_rate asc").Find(&categories).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Failed to retrieve room categories")
		return
	}

	catDTOs := make([]PublicCategoryDTO, 0, len(categories))
	for _, cat := range categories {
		var availableCount int64
		database.DB.Model(&models.Room{}).
			Where("hotel_id = ? AND category_id = ? AND status = 'available'", hotel.ID, cat.ID).
			Count(&availableCount)

		// Parse amenities
		amenities := []string{"High-Speed WiFi", "Air Conditioning", "Smart TV", "Mini Bar", "Safe"}
		if cat.Amenities != "" && !strings.HasPrefix(cat.Amenities, "[") {
			parts := strings.Split(cat.Amenities, ",")
			amenities = make([]string, 0, len(parts))
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					amenities = append(amenities, t)
				}
			}
		}

		img := "https://images.unsplash.com/photo-1631049307264-da0ec9d70304?w=800&q=80"
		if cat.Images != "" && !strings.HasPrefix(cat.Images, "[") {
			img = strings.TrimSpace(cat.Images)
		}

		rate := cat.DefaultRate
		if rate <= 0 {
			rate = cat.BaseRate
		}

		size := cat.RoomSize
		if size <= 0 {
			size = 30
		}

		catDTOs = append(catDTOs, PublicCategoryDTO{
			ID:             cat.ID,
			HotelID:        cat.HotelID,
			Name:           cat.Name,
			Description:    cat.Description,
			DefaultRate:    rate,
			BaseRate:       rate,
			MaxOccupancy:   cat.MaxOccupancy,
			BedType:        cat.BedType,
			RoomSize:       size,
			Size:           size,
			Amenities:      amenities,
			Image:          img,
			Images:         []string{img},
			AvailableRooms: availableCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       catDTOs,
		"categories": catDTOs,
	})
}

// GetPublicRoomDetail returns single room or room category details for public viewing
// GET /api/public/rooms/:id
// GET /api/v1/public/rooms/:id
// GET /api/v1/public/categories/:id
func GetPublicRoomDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		util.SendError(c, http.StatusBadRequest, "Room or category ID is required")
		return
	}

	// 1. Check if ID matches a room category
	var cat models.RoomCategory
	if err := database.DB.Where("id = ?", id).First(&cat).Error; err == nil {
		amenities := defaultHotelAmenities
		if cat.Amenities != "" {
			parts := strings.Split(cat.Amenities, ",")
			parsed := make([]string, 0, len(parts))
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					parsed = append(parsed, t)
				}
			}
			if len(parsed) > 0 {
				amenities = parsed
			}
		}

		img := cat.Image
		if img == "" {
			img = cat.Images
		}
		if img == "" {
			img = "https://images.unsplash.com/photo-1631049307264-da0ec9d70304?w=800&q=80"
		}

		rate := cat.DefaultRate
		if rate <= 0 {
			rate = cat.BaseRate
		}

		size := cat.RoomSize
		if size <= 0 {
			size = 35
		}

		// Find an available room in this category (lowest room number first)
		var availRoom models.Room
		var availRoomNum string = "101"
		var availRoomID string = ""
		if errR := database.DB.Where("category_id = ? AND status = 'available'", cat.ID).Order("room_number asc").First(&availRoom).Error; errR == nil {
			availRoomNum = availRoom.RoomNumber
			availRoomID = availRoom.ID
		}

		catDTO := PublicCategoryDTO{
			ID:             cat.ID,
			HotelID:        cat.HotelID,
			Name:           cat.Name,
			Description:    cat.Description,
			DefaultRate:    rate,
			BaseRate:       rate,
			MaxOccupancy:   cat.MaxOccupancy,
			BedType:        cat.BedType,
			RoomSize:       size,
			Size:           size,
			Amenities:      amenities,
			Image:          img,
			Images:         []string{img},
			AvailableRooms: 1,
		}

		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"data": gin.H{
				"id":           cat.ID,
				"name":         cat.Name,
				"description":  cat.Description,
				"defaultRate":  rate,
				"rate":         rate,
				"baseRate":     rate,
				"maxOccupancy": cat.MaxOccupancy,
				"bedType":      cat.BedType,
				"roomSize":     size,
				"size":         size,
				"amenities":    amenities,
				"images":       []string{img},
				"image":        img,
				"category":     catDTO,
				"roomNumber":   availRoomNum,
				"roomId":       availRoomID,
			},
		})
		return
	}

	// 2. Check if ID matches a physical room
	var room models.Room
	if err := database.DB.Preload("Category").Where("id = ? OR room_number = ?", id, id).First(&room).Error; err == nil {
		catName := "Standard Room"
		bedType := "King"
		maxOcc := 2
		roomSize := 30
		amenities := defaultHotelAmenities
		img := "https://images.unsplash.com/photo-1631049307264-da0ec9d70304?w=800&q=80"

		if room.Category != nil {
			catName = room.Category.Name
			bedType = room.Category.BedType
			maxOcc = room.Category.MaxOccupancy
			if room.Category.RoomSize > 0 {
				roomSize = room.Category.RoomSize
			}
			if room.Category.Image != "" {
				img = room.Category.Image
			} else if room.Category.Images != "" {
				img = room.Category.Images
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"id":         room.ID,
				"roomNumber": room.RoomNumber,
				"floor":      room.Floor,
				"rate":       room.Rate,
				"status":     room.Status,
				"category": gin.H{
					"id":           room.CategoryID,
					"name":         catName,
					"bedType":      bedType,
					"maxOccupancy": maxOcc,
					"roomSize":     roomSize,
					"amenities":    amenities,
					"images":       []string{img},
				},
			},
		})
		return
	}

	util.SendError(c, http.StatusNotFound, "Room or category not found")
}


// CheckAvailability queries rooms with date-overlap prevention logic
// GET /api/public/hotels/:hotelId/availability
// GET /api/v1/public/rooms
func CheckAvailability(c *gin.Context) {
	hotelID := c.Param("hotelId")
	hotel, err := resolveHotel(hotelID)
	if err != nil {
		util.SendError(c, http.StatusNotFound, "Hotel property not found")
		return
	}

	checkInStr := strings.TrimSpace(c.Query("checkIn"))
	checkOutStr := strings.TrimSpace(c.Query("checkOut"))
	guests, _ := strconv.Atoi(c.DefaultQuery("guests", "1"))
	if guests < 1 {
		guests = 1
	}
	categoryID := strings.TrimSpace(c.Query("categoryId"))

	now := time.Now()
	var checkIn, checkOut time.Time
	if checkInStr == "" {
		checkIn = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	} else {
		parsed, err := time.Parse("2006-01-02", checkInStr)
		if err != nil {
			util.SendError(c, http.StatusBadRequest, "Invalid checkIn date format (expected YYYY-MM-DD)")
			return
		}
		checkIn = parsed
	}

	if checkOutStr == "" {
		checkOut = checkIn.AddDate(0, 0, 1)
	} else {
		parsed, err := time.Parse("2006-01-02", checkOutStr)
		if err != nil {
			util.SendError(c, http.StatusBadRequest, "Invalid checkOut date format (expected YYYY-MM-DD)")
			return
		}
		checkOut = parsed
	}

	if !checkOut.After(checkIn) {
		util.SendError(c, http.StatusBadRequest, "checkOut date must be strictly after checkIn date")
		return
	}

	nights := int(checkOut.Sub(checkIn).Hours() / 24)
	if nights < 1 {
		nights = 1
	}

	// 1. Identify all room IDs that have overlapping reservations
	// Overlap condition: check_in < reqCheckOut AND check_out > reqCheckIn
	var overlappingRoomIDs []string
	database.DB.Model(&models.Reservation{}).
		Where("hotel_id = ? AND status IN ('confirmed', 'checked_in') AND check_in < ? AND check_out > ? AND room_id IS NOT NULL",
			hotel.ID, checkOut, checkIn).
		Pluck("room_id", &overlappingRoomIDs)

	// 2. Query available rooms
	roomQuery := database.DB.Model(&models.Room{}).
		Preload("Category").
		Where("hotel_id = ? AND status != 'out_of_service'", hotel.ID)

	if len(overlappingRoomIDs) > 0 {
		roomQuery = roomQuery.Where("id NOT IN ?", overlappingRoomIDs)
	}

	if categoryID != "" && categoryID != "ALL" {
		roomQuery = roomQuery.Where("category_id = ?", categoryID)
	}

	var rooms []models.Room
	if err := roomQuery.Order("floor asc, room_number asc").Find(&rooms).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to query room availability. Please try again.")
		return
	}

	// Filter by capacity and build DTOs
	availableRooms := make([]PublicRoomDTO, 0, len(rooms))
	for _, r := range rooms {
		if r.Category != nil && guests > 0 && r.Category.MaxOccupancy < guests {
			continue
		}

		var catDTO *PublicCategoryDTO
		if r.Category != nil {
			catDTO = &PublicCategoryDTO{
				ID:           r.Category.ID,
				HotelID:      r.Category.HotelID,
				Name:         r.Category.Name,
				Description:  r.Category.Description,
				DefaultRate:  r.Category.DefaultRate,
				BaseRate:     r.Category.DefaultRate,
				MaxOccupancy: r.Category.MaxOccupancy,
				BedType:      r.Category.BedType,
				RoomSize:     r.Category.RoomSize,
				Size:         r.Category.RoomSize,
				Image:        "https://images.unsplash.com/photo-1631049307264-da0ec9d70304?w=800&q=80",
			}
		}

		availableRooms = append(availableRooms, PublicRoomDTO{
			ID:         r.ID,
			HotelID:    r.HotelID,
			RoomNumber: r.RoomNumber,
			Floor:      r.Floor,
			Rate:       r.Rate,
			Status:     "available",
			CategoryID: r.CategoryID,
			Category:   catDTO,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    availableRooms,
		"rooms":   availableRooms,
		"meta": gin.H{
			"total":    len(availableRooms),
			"checkIn":  checkIn.Format("2006-01-02"),
			"checkOut": checkOut.Format("2006-01-02"),
			"nights":   nights,
			"guests":   guests,
		},
	})
}

// CreateBookingRequest defines payload for guest bookings
type CreateBookingRequest struct {
	HotelID         string `json:"hotelId"`
	RoomID          string `json:"roomId"`
	CategoryID      string `json:"categoryId"`
	GuestName       string `json:"guestName" binding:"required"`
	GuestEmail      string `json:"guestEmail" binding:"required,email"`
	GuestPhone      string `json:"guestPhone" binding:"required"`
	IDType          string `json:"idType"`
	IDNumber        string `json:"idNumber"`
	CheckIn         string `json:"checkIn" binding:"required"`
	CheckOut        string `json:"checkOut" binding:"required"`
	NumGuests       int    `json:"numGuests"`
	Guests          int    `json:"guests"`
	SpecialRequests string `json:"specialRequests"`
}

// CreateBooking processes guest booking without requiring authentication
// POST /api/public/bookings
// POST /api/v1/public/bookings
func CreateBooking(c *gin.Context) {
	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	// 1. Resolve Hotel
	hotel, err := resolveHotel(req.HotelID)
	if err != nil {
		util.SendError(c, http.StatusBadRequest, "Failed to resolve hotel property for booking")
		return
	}

	// 2. Parse and validate Dates
	checkIn, err := time.Parse("2006-01-02", req.CheckIn)
	if err != nil {
		util.SendError(c, http.StatusBadRequest, "Invalid checkIn date format (expected YYYY-MM-DD)")
		return
	}
	checkOut, err := time.Parse("2006-01-02", req.CheckOut)
	if err != nil {
		util.SendError(c, http.StatusBadRequest, "Invalid checkOut date format (expected YYYY-MM-DD)")
		return
	}

	if !checkOut.After(checkIn) {
		util.SendError(c, http.StatusBadRequest, "Check-out date must be after check-in date")
		return
	}

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
	if numGuests <= 0 {
		numGuests = req.Guests
	}
	if numGuests <= 0 {
		numGuests = 1
	}

	if len(strings.TrimSpace(req.GuestName)) < 2 {
		util.SendError(c, http.StatusBadRequest, "Guest name must be at least 2 characters")
		return
	}
	if len(strings.TrimSpace(req.GuestPhone)) < 7 {
		util.SendError(c, http.StatusBadRequest, "Please provide a valid contact phone number")
		return
	}

	// 3. Resolve and reserve physical room with overlap verification
	var room models.Room
	if req.RoomID != "" {
		if err := database.DB.Preload("Category").
			Where("id = ? AND hotel_id = ?", req.RoomID, hotel.ID).
			First(&room).Error; err != nil {
			util.SendError(c, http.StatusBadRequest, "Selected room does not exist in this property")
			return
		}

		// Verify no overlap on selected room
		var overlapCount int64
		database.DB.Model(&models.Reservation{}).
			Where("room_id = ? AND status IN ('confirmed', 'checked_in') AND check_in < ? AND check_out > ?",
				room.ID, checkOut, checkIn).
			Count(&overlapCount)
		if overlapCount > 0 {
			util.SendError(c, http.StatusConflict, fmt.Sprintf("Room %s is already booked for the selected dates", room.RoomNumber))
			return
		}
	} else {
		// Auto-assign from category
		catID := req.CategoryID
		if catID == "" {
			var firstCat models.RoomCategory
			if err := database.DB.Where("hotel_id = ?", hotel.ID).First(&firstCat).Error; err != nil {
				util.SendError(c, http.StatusBadRequest, "No room categories found for this property")
				return
			}
			catID = firstCat.ID
		}

		// Query available rooms in this category without overlapping bookings
		var overlapRoomIDs []string
		database.DB.Model(&models.Reservation{}).
			Where("hotel_id = ? AND status IN ('confirmed', 'checked_in') AND check_in < ? AND check_out > ? AND room_id IS NOT NULL",
				hotel.ID, checkOut, checkIn).
			Pluck("room_id", &overlapRoomIDs)

		roomQuery := database.DB.Preload("Category").
			Where("hotel_id = ? AND category_id = ? AND status != 'out_of_service'", hotel.ID, catID)
		if len(overlapRoomIDs) > 0 {
			roomQuery = roomQuery.Where("id NOT IN ?", overlapRoomIDs)
		}

		if err := roomQuery.Order("room_number asc").First(&room).Error; err != nil {
			util.SendError(c, http.StatusConflict, "No available rooms in this category for the requested dates")
			return
		}
	}

	// Max occupancy check
	if room.Category != nil && room.Category.MaxOccupancy > 0 && numGuests > room.Category.MaxOccupancy {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Number of guests (%d) exceeds maximum room capacity of %d", numGuests, room.Category.MaxOccupancy))
		return
	}

	// 4. Rate & Indian GST Calculation (12%)
	ratePerNight := room.Rate
	if ratePerNight <= 0 && room.Category != nil {
		ratePerNight = room.Category.DefaultRate
	}
	subtotal := math.Round(ratePerNight*float64(nights)*100) / 100
	taxRate := 0.12
	taxAmount := math.Round(subtotal*taxRate*100) / 100
	totalAmount := math.Round((subtotal+taxAmount)*100) / 100

	// 5. Upsert Guest record
	var guest models.Guest
	cleanEmail := strings.ToLower(strings.TrimSpace(req.GuestEmail))
	if err := database.DB.Where("LOWER(email) = ?", cleanEmail).First(&guest).Error; err != nil {
		idType := req.IDType
		if idType == "" {
			idType = "Aadhaar"
		}
		guest = models.Guest{
			Name:     strings.TrimSpace(req.GuestName),
			Email:    cleanEmail,
			Phone:    strings.TrimSpace(req.GuestPhone),
			IDType:   idType,
			IDNumber: strings.TrimSpace(req.IDNumber),
		}
		database.DB.Create(&guest)
	}

	// 6. Generate unique booking reference BK-YYYY-XXXXX
	bookingRef := generateBookingReference()
	for {
		var exists int64
		database.DB.Model(&models.Reservation{}).Where("booking_reference = ?", bookingRef).Count(&exists)
		if exists == 0 {
			break
		}
		bookingRef = generateBookingReference()
	}

	// 7. Create Reservation
	resv := models.Reservation{
		BookingReference: bookingRef,
		HotelID:          hotel.ID,
		RoomID:           &room.ID,
		GuestID:          &guest.ID,
		GuestName:        strings.TrimSpace(req.GuestName),
		GuestEmail:       cleanEmail,
		GuestPhone:       strings.TrimSpace(req.GuestPhone),
		CheckIn:          checkIn,
		CheckOut:         checkOut,
		Nights:           nights,
		NumGuests:        numGuests,
		RoomRate:         ratePerNight,
		TotalAmount:      totalAmount,
		Status:           "confirmed",
		SpecialRequests:  strings.TrimSpace(req.SpecialRequests),
	}

	if err := database.DB.Create(&resv).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to complete booking. Please try again.")
		return
	}

	// 8. Generate Folio / Bill
	catName := "Standard Room"
	if room.Category != nil {
		catName = room.Category.Name
	}

	bill := models.Bill{
		HotelID:       hotel.ID,
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
		TotalAmount:   totalAmount,
		Balance:       totalAmount,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	billItem := models.BillItem{
		BillID:      bill.ID,
		Description: fmt.Sprintf("%s – %d nights @ ₹%.2f/night", catName, nights, ratePerNight),
		Quantity:    nights,
		UnitPrice:   ratePerNight,
		Amount:      subtotal,
	}
	database.DB.Create(&billItem)

	// 9. Activity Log
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotel.ID,
		Action:  "public_booking",
		Details: fmt.Sprintf("Confirmed reservation %s for %s in Room %s (₹%.2f)", bookingRef, resv.GuestName, room.RoomNumber, totalAmount),
	})

	respDTO := BookingResponseDTO{
		ID:               resv.ID,
		BookingReference: bookingRef,
		Status:           "confirmed",
		HotelID:          hotel.ID,
		HotelName:        hotel.Name,
		RoomID:           room.ID,
		RoomNumber:       room.RoomNumber,
		CategoryID:       room.CategoryID,
		RoomCategory:     catName,
		CategoryName:     catName,
		GuestName:        resv.GuestName,
		GuestEmail:       resv.GuestEmail,
		GuestPhone:       resv.GuestPhone,
		CheckIn:          checkIn.Format("2006-01-02"),
		CheckOut:         checkOut.Format("2006-01-02"),
		Nights:           nights,
		NumGuests:        numGuests,
		Guests:           numGuests,
		RatePerNight:     ratePerNight,
		RoomRate:         ratePerNight,
		Subtotal:         subtotal,
		TaxRate:          taxRate,
		TaxAmount:        taxAmount,
		TotalAmount:      totalAmount,
		GrandTotal:       totalAmount,
		Currency:         "INR",
		SpecialRequests:  resv.SpecialRequests,
		CreatedAt:        resv.CreatedAt,
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Booking successfully confirmed",
		"data":    respDTO,
		"booking": respDTO,
	})
}

// GetBookingByRef retrieves confirmation by booking reference
// GET /api/public/bookings/:bookingReference
// GET /api/v1/public/bookings/:bookingReference
func GetBookingByRef(c *gin.Context) {
	ref := strings.TrimSpace(c.Param("bookingReference"))

	var resv models.Reservation
	if err := database.DB.Preload("Hotel").Preload("Room").Preload("Room.Category").Preload("Guest").
		Where("LOWER(booking_reference) = ?", strings.ToLower(ref)).
		First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found")
		return
	}

	hotelName := resv.Hotel.Name
	roomNumber := "TBA"
	catName := "Room"
	roomID := ""
	catID := ""
	if resv.Room != nil {
		roomNumber = resv.Room.RoomNumber
		roomID = resv.Room.ID
		if resv.Room.Category != nil {
			catName = resv.Room.Category.Name
			catID = resv.Room.Category.ID
		}
	}

	subtotal := math.Round(resv.RoomRate*float64(resv.Nights)*100) / 100
	taxRate := 0.12
	taxAmount := math.Round(subtotal*taxRate*100) / 100
	totalAmount := math.Round((subtotal+taxAmount)*100) / 100

	dto := BookingResponseDTO{
		ID:               resv.ID,
		BookingReference: resv.BookingReference,
		Status:           resv.Status,
		HotelID:          resv.HotelID,
		HotelName:        hotelName,
		RoomID:           roomID,
		RoomNumber:       roomNumber,
		CategoryID:       catID,
		RoomCategory:     catName,
		CategoryName:     catName,
		GuestName:        resv.GuestName,
		GuestEmail:       resv.GuestEmail,
		GuestPhone:       resv.GuestPhone,
		CheckIn:          resv.CheckIn.Format("2006-01-02"),
		CheckOut:         resv.CheckOut.Format("2006-01-02"),
		Nights:           resv.Nights,
		NumGuests:        resv.NumGuests,
		Guests:           resv.NumGuests,
		RatePerNight:     resv.RoomRate,
		RoomRate:         resv.RoomRate,
		Subtotal:         subtotal,
		TaxRate:          taxRate,
		TaxAmount:        taxAmount,
		TotalAmount:      totalAmount,
		GrandTotal:       totalAmount,
		Currency:         "INR",
		SpecialRequests:  resv.SpecialRequests,
		CreatedAt:        resv.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"booking": dto,
	})
}
