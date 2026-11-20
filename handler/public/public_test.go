package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var (
	testRouter    *gin.Engine
	seededHotelID string
	seededCatID   string
	seededRoomID  string
)

func setupPublicTestDB(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	database.DB = db

	err = db.AutoMigrate(
		&models.Hotel{},
		&models.User{},
		&models.HotelAdminAssignment{},
		&models.RoomCategory{},
		&models.Room{},
		&models.Guest{},
		&models.Reservation{},
		&models.Bill{},
		&models.BillItem{},
		&models.Payment{},
		&models.HousekeepingLog{},
		&models.ActivityLog{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	// 1. Create demo Hotel
	hotel := models.Hotel{
		Name:         "Grand Horizon Hotel",
		Description:  "Mumbai's iconic 5-star beachfront luxury",
		Address:      "Marine Drive, Nariman Point",
		City:         "Mumbai",
		State:        "Maharashtra",
		Country:      "India",
		Phone:        "+91 22 6610 8888",
		Email:        "contact@grandhorizon.com",
		Status:       "active",
		StarRating:   5,
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
	}
	db.Create(&hotel)
	seededHotelID = hotel.ID

	// 2. Create Room Category
	cat := models.RoomCategory{
		HotelID:      hotel.ID,
		Name:         "Deluxe Sea View Room",
		Description:  "Panoramic sea view with king bed",
		DefaultRate:  5000,
		MaxOccupancy: 2,
		BedType:      "King",
		RoomSize:     35,
		Amenities:    "High-Speed WiFi, Air Conditioning, Smart TV",
		Images:       "https://images.unsplash.com/photo-deluxe",
	}
	db.Create(&cat)
	seededCatID = cat.ID

	// 3. Create Rooms
	room1 := models.Room{
		HotelID:    hotel.ID,
		CategoryID: cat.ID,
		RoomNumber: "101",
		Floor:      1,
		Rate:       5000,
		Status:     "available",
	}
	db.Create(&room1)
	seededRoomID = room1.ID

	room2 := models.Room{
		HotelID:    hotel.ID,
		CategoryID: cat.ID,
		RoomNumber: "102",
		Floor:      1,
		Rate:       5000,
		Status:     "available",
	}
	db.Create(&room2)

	// Router setup
	testRouter = gin.New()
	publicGroup := testRouter.Group("/api/public")
	{
		publicGroup.GET("/hotels/:hotelId", GetHotel)
		publicGroup.GET("/hotels/:hotelId/categories", GetRooms)
		publicGroup.GET("/hotels/:hotelId/rooms", GetRooms)
		publicGroup.GET("/hotels/:hotelId/availability", CheckAvailability)
		publicGroup.POST("/bookings", CreateBooking)
		publicGroup.GET("/bookings/:bookingReference", GetBookingByRef)
	}

	v1Public := testRouter.Group("/api/v1/public")
	{
		v1Public.GET("/hotel", GetHotel)
		v1Public.GET("/rooms", CheckAvailability)
		v1Public.GET("/rooms/:id", GetPublicRoomDetail)
		v1Public.GET("/categories", GetRooms)
		v1Public.GET("/categories/:id", GetPublicRoomDetail)
	}
}

func TestPublic_GetHotel(t *testing.T) {
	setupPublicTestDB(t)

	// With ID
	req, _ := http.NewRequest(http.MethodGet, "/api/public/hotels/"+seededHotelID, nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	hotelData := res["data"].(map[string]interface{})
	if hotelData["name"] != "Grand Horizon Hotel" {
		t.Errorf("expected Grand Horizon Hotel, got %v", hotelData["name"])
	}

	// Single hotel mode
	reqSingle, _ := http.NewRequest(http.MethodGet, "/api/v1/public/hotel", nil)
	wSingle := httptest.NewRecorder()
	testRouter.ServeHTTP(wSingle, reqSingle)

	if wSingle.Code != http.StatusOK {
		t.Fatalf("expected 200 on single hotel endpoint, got %d", wSingle.Code)
	}
}

func TestPublic_GetCategories(t *testing.T) {
	setupPublicTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/api/public/hotels/"+seededHotelID+"/categories", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	cats := res["data"].([]interface{})
	if len(cats) != 1 {
		t.Errorf("expected 1 category, got %d", len(cats))
	}
}

func TestPublic_CheckAvailability_And_DateOverlap(t *testing.T) {
	setupPublicTestDB(t)

	// Seed confirmed reservation for Room 101 from Dec 1 to Dec 4
	checkIn, _ := time.Parse("2006-01-02", "2024-12-01")
	checkOut, _ := time.Parse("2006-01-02", "2024-12-04")
	resv := models.Reservation{
		BookingReference: "BK-2024-11111",
		HotelID:          seededHotelID,
		RoomID:           &seededRoomID,
		GuestName:        "Existing Guest",
		GuestEmail:       "guest@example.com",
		GuestPhone:       "+91 99999 00000",
		CheckIn:          checkIn,
		CheckOut:         checkOut,
		Nights:           3,
		Status:           "confirmed",
	}
	database.DB.Create(&resv)

	// 1. Query Dec 2 to Dec 3 (Inside existing booking range) -> Room 101 must be EXCLUDED, Room 102 must be INCLUDED
	req, _ := http.NewRequest(http.MethodGet, "/api/public/hotels/"+seededHotelID+"/availability?checkIn=2024-12-02&checkOut=2024-12-03", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	rooms := res["data"].([]interface{})
	if len(rooms) != 1 {
		t.Fatalf("expected exactly 1 room available during overlap, got %d", len(rooms))
	}
	roomAvailable := rooms[0].(map[string]interface{})
	if roomAvailable["roomNumber"] != "102" {
		t.Errorf("expected Room 102 to be available, but got %v", roomAvailable["roomNumber"])
	}

	// 2. Query Dec 4 to Dec 7 (CheckIn on the same day as previous CheckOut) -> Room 101 MUST be available
	reqTurnaround, _ := http.NewRequest(http.MethodGet, "/api/public/hotels/"+seededHotelID+"/availability?checkIn=2024-12-04&checkOut=2024-12-07", nil)
	wTurnaround := httptest.NewRecorder()
	testRouter.ServeHTTP(wTurnaround, reqTurnaround)

	var resTurnaround map[string]interface{}
	json.Unmarshal(wTurnaround.Body.Bytes(), &resTurnaround)
	roomsTurnaround := resTurnaround["data"].([]interface{})
	if len(roomsTurnaround) != 2 {
		t.Errorf("expected both rooms (101 & 102) to be available on same-day turnaround, got %d", len(roomsTurnaround))
	}

	// 3. Bad date order (checkOut < checkIn) -> 400
	reqBad, _ := http.NewRequest(http.MethodGet, "/api/public/hotels/"+seededHotelID+"/availability?checkIn=2024-12-05&checkOut=2024-12-01", nil)
	wBad := httptest.NewRecorder()
	testRouter.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for inverted dates, got %d", wBad.Code)
	}
}

func TestPublic_CreateBooking_And_GST(t *testing.T) {
	setupPublicTestDB(t)

	now := time.Now()
	testCheckIn := now.AddDate(0, 0, 10).Format("2006-01-02")
	testCheckOut := now.AddDate(0, 0, 13).Format("2006-01-02")

	// 1. Create a 3-night booking for Room 101 @ ₹5,000/night
	bookingPayload := map[string]interface{}{
		"hotelId":         seededHotelID,
		"roomId":          seededRoomID,
		"guestName":       "Priya Sharma",
		"guestEmail":      "priya.sharma@example.com",
		"guestPhone":      "+91 98700 12345",
		"checkIn":         testCheckIn,
		"checkOut":        testCheckOut,
		"numGuests":       2,
		"specialRequests": "Upper floor sea view",
	}
	body, _ := json.Marshal(bookingPayload)

	req, _ := http.NewRequest(http.MethodPost, "/api/public/bookings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on booking create, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	booking := res["data"].(map[string]interface{})

	// Assert booking reference
	bookingRef := booking["bookingReference"].(string)
	if len(bookingRef) == 0 || bookingRef[:3] != "BK-" {
		t.Errorf("expected booking reference starting with BK-, got %s", bookingRef)
	}

	// Assert nights and rates
	if booking["nights"].(float64) != 3 {
		t.Errorf("expected 3 nights, got %v", booking["nights"])
	}
	if booking["subtotal"].(float64) != 15000 {
		t.Errorf("expected subtotal 15000 (3 * 5000), got %v", booking["subtotal"])
	}

	// Assert 12% GST in INR
	if booking["taxRate"].(float64) != 0.12 {
		t.Errorf("expected 12%% GST tax rate, got %v", booking["taxRate"])
	}
	if booking["taxAmount"].(float64) != 1800 {
		t.Errorf("expected 1800 tax (15000 * 0.12), got %v", booking["taxAmount"])
	}
	if booking["totalAmount"].(float64) != 16800 {
		t.Errorf("expected 16800 totalAmount (15000 + 1800), got %v", booking["totalAmount"])
	}

	// Assert Bill creation in DB
	var bill models.Bill
	err := database.DB.Where("guest_email = ?", "priya.sharma@example.com").First(&bill).Error
	if err != nil {
		t.Errorf("expected bill to be created in DB, got error: %v", err)
	}
	if bill.TotalAmount != 16800 {
		t.Errorf("expected bill total 16800, got %v", bill.TotalAmount)
	}

	// 2. Attempt duplicate booking for same room on overlapping dates -> 409 Conflict
	dupReq, _ := http.NewRequest(http.MethodPost, "/api/public/bookings", bytes.NewBuffer(body))
	dupReq.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	testRouter.ServeHTTP(wDup, dupReq)

	if wDup.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for double-booking attempt, got %d", wDup.Code)
	}

	// 3. Retrieve confirmed booking by reference
	reqLookup, _ := http.NewRequest(http.MethodGet, "/api/public/bookings/"+bookingRef, nil)
	wLookup := httptest.NewRecorder()
	testRouter.ServeHTTP(wLookup, reqLookup)

	if wLookup.Code != http.StatusOK {
		t.Fatalf("expected 200 on booking lookup by ref, got %d", wLookup.Code)
	}

	var resLookup map[string]interface{}
	json.Unmarshal(wLookup.Body.Bytes(), &resLookup)
	lookupData := resLookup["data"].(map[string]interface{})
	if lookupData["guestName"] != "Priya Sharma" {
		t.Errorf("expected Priya Sharma, got %v", lookupData["guestName"])
	}
	if lookupData["roomNumber"] != "101" {
		t.Errorf("expected room 101, got %v", lookupData["roomNumber"])
	}
}

func TestPublic_GetBookingByRef_NotFound(t *testing.T) {
	setupPublicTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/api/public/bookings/BK-9999-99999", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent booking ref, got %d", w.Code)
	}
}

func TestPublic_GetRoomDetail(t *testing.T) {
	setupPublicTestDB(t)

	// 1. Query by category ID via /api/v1/public/categories/:id
	reqCat, _ := http.NewRequest(http.MethodGet, "/api/v1/public/categories/"+seededCatID, nil)
	wCat := httptest.NewRecorder()
	testRouter.ServeHTTP(wCat, reqCat)

	if wCat.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for category detail, got %d, body: %s", wCat.Code, wCat.Body.String())
	}

	var resCat struct {
		Success bool `json:"success"`
		Data    struct {
			Name        string  `json:"name"`
			DefaultRate float64 `json:"defaultRate"`
			RoomNumber  string  `json:"roomNumber"`
		} `json:"data"`
	}
	json.Unmarshal(wCat.Body.Bytes(), &resCat)
	if resCat.Data.Name != "Deluxe Sea View Room" {
		t.Errorf("expected 'Deluxe Sea View Room', got: %s", resCat.Data.Name)
	}
	if resCat.Data.RoomNumber != "101" && resCat.Data.RoomNumber != "102" {
		t.Errorf("expected room number '101' or '102', got: %s", resCat.Data.RoomNumber)
	}

	// 2. Query by physical room ID via /api/v1/public/rooms/:id
	reqRoom, _ := http.NewRequest(http.MethodGet, "/api/v1/public/rooms/"+seededRoomID, nil)
	wRoom := httptest.NewRecorder()
	testRouter.ServeHTTP(wRoom, reqRoom)

	if wRoom.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for room detail, got %d, body: %s", wRoom.Code, wRoom.Body.String())
	}

	var resRoom struct {
		Success bool `json:"success"`
		Data    struct {
			RoomNumber string `json:"roomNumber"`
			Category   struct {
				Name string `json:"name"`
			} `json:"category"`
		} `json:"data"`
	}
	json.Unmarshal(wRoom.Body.Bytes(), &resRoom)
	if resRoom.Data.RoomNumber != "101" {
		t.Errorf("expected room 101, got: %s", resRoom.Data.RoomNumber)
	}
	if resRoom.Data.Category.Name != "Deluxe Sea View Room" {
		t.Errorf("expected category 'Deluxe Sea View Room', got: %s", resRoom.Data.Category.Name)
	}
}

func TestPublic_Validation_Guards(t *testing.T) {
	setupPublicTestDB(t)

	now := time.Now()
	testCheckIn := now.AddDate(0, 0, 10).Format("2006-01-02")
	testCheckOut := now.AddDate(0, 0, 12).Format("2006-01-02")

	// 1. Past check-in date -> 400
	pastPayload := map[string]interface{}{
		"hotelId":    seededHotelID,
		"roomId":     seededRoomID,
		"guestName":  "Past Guest",
		"guestEmail": "past@example.com",
		"guestPhone": "+91 99999 88888",
		"checkIn":    "2020-01-01",
		"checkOut":   "2020-01-03",
		"numGuests":  1,
	}
	bodyPast, _ := json.Marshal(pastPayload)
	reqPast, _ := http.NewRequest(http.MethodPost, "/api/public/bookings", bytes.NewBuffer(bodyPast))
	reqPast.Header.Set("Content-Type", "application/json")
	wPast := httptest.NewRecorder()
	testRouter.ServeHTTP(wPast, reqPast)

	if wPast.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for past booking date, got %d", wPast.Code)
	}

	// 2. Overcapacity (Deluxe category has MaxOccupancy = 2, request 6 guests) -> 400
	overOccPayload := map[string]interface{}{
		"hotelId":    seededHotelID,
		"roomId":     seededRoomID,
		"guestName":  "Huge Party",
		"guestEmail": "party@example.com",
		"guestPhone": "+91 99999 88888",
		"checkIn":    testCheckIn,
		"checkOut":   testCheckOut,
		"numGuests":  6,
	}
	bodyOcc, _ := json.Marshal(overOccPayload)
	reqOcc, _ := http.NewRequest(http.MethodPost, "/api/public/bookings", bytes.NewBuffer(bodyOcc))
	reqOcc.Header.Set("Content-Type", "application/json")
	wOcc := httptest.NewRecorder()
	testRouter.ServeHTTP(wOcc, reqOcc)

	if wOcc.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for exceeding max occupancy, got %d", wOcc.Code)
	}

	// 3. Invalid phone (< 7 digits) -> 400
	badPhonePayload := map[string]interface{}{
		"hotelId":    seededHotelID,
		"roomId":     seededRoomID,
		"guestName":  "Bad Phone",
		"guestEmail": "phone@example.com",
		"guestPhone": "123",
		"checkIn":    testCheckIn,
		"checkOut":   testCheckOut,
		"numGuests":  1,
	}
	bodyPhone, _ := json.Marshal(badPhonePayload)
	reqPhone, _ := http.NewRequest(http.MethodPost, "/api/public/bookings", bytes.NewBuffer(bodyPhone))
	reqPhone.Header.Set("Content-Type", "application/json")
	wPhone := httptest.NewRecorder()
	testRouter.ServeHTTP(wPhone, reqPhone)

	if wPhone.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for short phone number, got %d", wPhone.Code)
	}
}
