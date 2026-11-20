package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"hotel-backend/database"
	"hotel-backend/middleware"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	testRouter     *gin.Engine
	hotelAJWT      string
	hotelBJWT      string
	hotelAID       string
	hotelBID       string
	seededCatAID   string
	seededRoomAID  string
)

func setupInventoryTestDB(t *testing.T) {
	os.Setenv("JWT_SECRET", "inventory_test_secret_key_1234567890")
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
		&models.FacilityCharge{},
		&models.HousekeepingLog{},
		&models.ActivityLog{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	// 1. Hotel A (Grand Horizon)
	hotelA := models.Hotel{
		Name:    "Grand Horizon Hotel",
		Address: "Marine Drive",
		City:    "Mumbai",
		Status:  "active",
	}
	db.Create(&hotelA)
	hotelAID = hotelA.ID

	adminA := models.User{
		Name:         "Arjun Sharma",
		Email:        "admin@grandhorizon.com",
		PasswordHash: string(hashed),
		Role:         "hotel_admin",
		Status:       "active",
	}
	db.Create(&adminA)
	db.Create(&models.HotelAdminAssignment{UserID: adminA.ID, HotelID: hotelA.ID})

	tokenA, _ := util.GenerateToken(adminA.ID, adminA.Name, adminA.Email, adminA.Role, &hotelA.ID, hotelA.Name)
	hotelAJWT = tokenA

	// 2. Hotel B (Seaside Paradise)
	hotelB := models.Hotel{
		Name:    "Seaside Paradise Resort",
		Address: "Calangute Beach",
		City:    "Goa",
		Status:  "active",
	}
	db.Create(&hotelB)
	hotelBID = hotelB.ID

	adminB := models.User{
		Name:         "Priya Nair",
		Email:        "admin@seasideparadise.com",
		PasswordHash: string(hashed),
		Role:         "hotel_admin",
		Status:       "active",
	}
	db.Create(&adminB)
	db.Create(&models.HotelAdminAssignment{UserID: adminB.ID, HotelID: hotelB.ID})

	tokenB, _ := util.GenerateToken(adminB.ID, adminB.Name, adminB.Email, adminB.Role, &hotelB.ID, hotelB.Name)
	hotelBJWT = tokenB

	// 3. Seed Category & Room for Hotel A
	catA := models.RoomCategory{
		HotelID:      hotelA.ID,
		Name:         "Deluxe Suite",
		Description:  "Spacious deluxe suite with sea view",
		DefaultRate:  5500,
		MaxOccupancy: 2,
		BedType:      "King",
		RoomSize:     38,
		Amenities:    `["WiFi","Sea View","Air Conditioning"]`,
	}
	db.Create(&catA)
	seededCatAID = catA.ID

	roomA := models.Room{
		HotelID:    hotelA.ID,
		CategoryID: catA.ID,
		RoomNumber: "101",
		Floor:      1,
		Rate:       5500,
		Status:     "available",
		Notes:      "Sea-facing room",
	}
	db.Create(&roomA)
	seededRoomAID = roomA.ID

	// Setup router
	testRouter = gin.New()
	adminGroup := testRouter.Group("/api/admin")
	adminGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		// Categories
		adminGroup.GET("/room-categories", GetCategories)
		adminGroup.POST("/room-categories", CreateCategory)
		adminGroup.GET("/room-categories/:id", GetCategoryByID)
		adminGroup.PUT("/room-categories/:id", UpdateCategory)
		adminGroup.DELETE("/room-categories/:id", DeleteCategory)

		// Rooms
		adminGroup.GET("/rooms", GetRooms)
		adminGroup.POST("/rooms", CreateRoom)
		adminGroup.GET("/rooms/:id", GetRoomByID)
		adminGroup.PUT("/rooms/:id", UpdateRoom)
		adminGroup.DELETE("/rooms/:id", DeleteRoom)
		adminGroup.PATCH("/rooms/:id/status", UpdateRoomStatus)
	}
}

func TestCategory_Lifecycle(t *testing.T) {
	setupInventoryTestDB(t)

	// 1. Get Categories (Hotel A should have 1 category)
	req, _ := http.NewRequest(http.MethodGet, "/api/admin/room-categories", nil)
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	cats := res["data"].([]interface{})
	if len(cats) != 1 {
		t.Errorf("expected 1 category, got %d", len(cats))
	}

	// 2. Create Duplicate Category Name in Hotel A -> Reject 400
	dupPayload := map[string]interface{}{
		"name":        "Deluxe Suite",
		"defaultRate": 6000,
	}
	dupBody, _ := json.Marshal(dupPayload)
	reqDup, _ := http.NewRequest(http.MethodPost, "/api/admin/room-categories", bytes.NewBuffer(dupBody))
	reqDup.Header.Set("Content-Type", "application/json")
	reqDup.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDup := httptest.NewRecorder()
	testRouter.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate category name, got %d", wDup.Code)
	}

	// 3. Create New Category in Hotel A -> Success 201
	newPayload := map[string]interface{}{
		"name":        "Executive Suite",
		"description": "Top floor suite with boardroom",
		"defaultRate": 12000,
		"bedType":     "King",
		"roomSize":    75,
		"amenities":   []string{"WiFi", "Butler", "Boardroom"},
		"image":       "https://images.unsplash.com/photo-suite",
	}
	newBody, _ := json.Marshal(newPayload)
	reqNew, _ := http.NewRequest(http.MethodPost, "/api/admin/room-categories", bytes.NewBuffer(newBody))
	reqNew.Header.Set("Content-Type", "application/json")
	reqNew.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wNew := httptest.NewRecorder()
	testRouter.ServeHTTP(wNew, reqNew)

	if wNew.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", wNew.Code, wNew.Body.String())
	}

	var newRes map[string]interface{}
	json.Unmarshal(wNew.Body.Bytes(), &newRes)
	createdCat := newRes["data"].(map[string]interface{})
	newCatID := createdCat["id"].(string)

	if createdCat["baseRate"].(float64) != 12000 {
		t.Errorf("expected baseRate 12000, got %v", createdCat["baseRate"])
	}

	// 4. Update Category
	updPayload := map[string]interface{}{
		"name":        "Presidential Executive Suite",
		"defaultRate": 15000,
	}
	updBody, _ := json.Marshal(updPayload)
	reqUpd, _ := http.NewRequest(http.MethodPut, "/api/admin/room-categories/"+newCatID, bytes.NewBuffer(updBody))
	reqUpd.Header.Set("Content-Type", "application/json")
	reqUpd.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wUpd := httptest.NewRecorder()
	testRouter.ServeHTTP(wUpd, reqUpd)

	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", wUpd.Code, wUpd.Body.String())
	}

	// 5. Try deleting category that has rooms attached (Deluxe Suite has Room 101) -> Reject 409
	reqDelBlocked, _ := http.NewRequest(http.MethodDelete, "/api/admin/room-categories/"+seededCatAID, nil)
	reqDelBlocked.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDelBlocked := httptest.NewRecorder()
	testRouter.ServeHTTP(wDelBlocked, reqDelBlocked)

	if wDelBlocked.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict when deleting category with attached rooms, got %d", wDelBlocked.Code)
	}

	// 6. Delete category without rooms attached -> Success 200
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/admin/room-categories/"+newCatID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDel := httptest.NewRecorder()
	testRouter.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Errorf("expected 200 on deleting unreferenced category, got %d", wDel.Code)
	}
}

func TestRoom_Lifecycle(t *testing.T) {
	setupInventoryTestDB(t)

	// 1. Create duplicate room number in Hotel A -> Reject 400
	dupPayload := map[string]interface{}{
		"roomNumber": "101",
		"floor":      1,
		"categoryId": seededCatAID,
		"rate":       5500,
	}
	dupBody, _ := json.Marshal(dupPayload)
	reqDup, _ := http.NewRequest(http.MethodPost, "/api/admin/rooms", bytes.NewBuffer(dupBody))
	reqDup.Header.Set("Content-Type", "application/json")
	reqDup.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDup := httptest.NewRecorder()
	testRouter.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate room number, got %d", wDup.Code)
	}

	// 2. Create valid new room
	newPayload := map[string]interface{}{
		"roomNumber": "102",
		"floor":      1,
		"categoryId": seededCatAID,
		"rate":       5500,
		"status":     "available",
		"notes":      "Near garden entrance",
	}
	newBody, _ := json.Marshal(newPayload)
	reqNew, _ := http.NewRequest(http.MethodPost, "/api/admin/rooms", bytes.NewBuffer(newBody))
	reqNew.Header.Set("Content-Type", "application/json")
	reqNew.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wNew := httptest.NewRecorder()
	testRouter.ServeHTTP(wNew, reqNew)

	if wNew.Code != http.StatusCreated {
		t.Fatalf("expected 201 on room create, got %d: %s", wNew.Code, wNew.Body.String())
	}

	var newRes map[string]interface{}
	json.Unmarshal(wNew.Body.Bytes(), &newRes)
	roomData := newRes["data"].(map[string]interface{})
	newRoomID := roomData["id"].(string)

	if roomData["categoryName"] != "Deluxe Suite" {
		t.Errorf("expected categoryName 'Deluxe Suite', got %v", roomData["categoryName"])
	}

	// 3. Update Room
	updPayload := map[string]interface{}{
		"roomNumber": "102-A",
		"floor":      1,
		"categoryId": seededCatAID,
		"rate":       5800,
		"notes":      "Renovated with balcony",
	}
	updBody, _ := json.Marshal(updPayload)
	reqUpd, _ := http.NewRequest(http.MethodPut, "/api/admin/rooms/"+newRoomID, bytes.NewBuffer(updBody))
	reqUpd.Header.Set("Content-Type", "application/json")
	reqUpd.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wUpd := httptest.NewRecorder()
	testRouter.ServeHTTP(wUpd, reqUpd)

	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 on room update, got %d: %s", wUpd.Code, wUpd.Body.String())
	}

	// 4. Update Room Status (Housekeeping lifecycle: available -> dirty -> cleaning -> available)
	statusPayload := map[string]interface{}{
		"status":     "dirty",
		"notes":      "Guest checked out",
		"assignedTo": "Housekeeping Team A",
	}
	stBody, _ := json.Marshal(statusPayload)
	reqSt, _ := http.NewRequest(http.MethodPatch, "/api/admin/rooms/"+newRoomID+"/status", bytes.NewBuffer(stBody))
	reqSt.Header.Set("Content-Type", "application/json")
	reqSt.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wSt := httptest.NewRecorder()
	testRouter.ServeHTTP(wSt, reqSt)

	if wSt.Code != http.StatusOK {
		t.Fatalf("expected 200 on status patch, got %d: %s", wSt.Code, wSt.Body.String())
	}

	// Verify housekeeping log was created
	var hkLogs []models.HousekeepingLog
	database.DB.Where("room_id = ?", newRoomID).Find(&hkLogs)
	if len(hkLogs) == 0 {
		t.Errorf("expected housekeeping log to be generated on room status change")
	}

	// 5. Delete Room with active booking -> Reject 409
	resv := models.Reservation{
		BookingReference: "BK-ACTIVE-ROOM",
		HotelID:          hotelAID,
		RoomID:           &newRoomID,
		GuestName:        "Rahul Verma",
		GuestEmail:       "rahul@example.com",
		GuestPhone:       "+91 98111 22334",
		CheckIn:          time.Now(),
		CheckOut:         time.Now().Add(24 * time.Hour),
		Nights:           1,
		Status:           "confirmed",
	}
	database.DB.Create(&resv)

	reqDelConflict, _ := http.NewRequest(http.MethodDelete, "/api/admin/rooms/"+newRoomID, nil)
	reqDelConflict.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDelConflict := httptest.NewRecorder()
	testRouter.ServeHTTP(wDelConflict, reqDelConflict)

	if wDelConflict.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict when deleting room with active reservations, got %d", wDelConflict.Code)
	}

	// Cancel reservation and delete
	database.DB.Model(&resv).Update("status", "cancelled")
	reqDelSuccess, _ := http.NewRequest(http.MethodDelete, "/api/admin/rooms/"+newRoomID, nil)
	reqDelSuccess.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDelSuccess := httptest.NewRecorder()
	testRouter.ServeHTTP(wDelSuccess, reqDelSuccess)

	if wDelSuccess.Code != http.StatusOK {
		t.Errorf("expected 200 on deleting room without active reservations, got %d", wDelSuccess.Code)
	}
}

func TestInventory_MultiTenant_Isolation(t *testing.T) {
	setupInventoryTestDB(t)

	// Hotel B Admin attempts to access Hotel A's Room -> 404
	reqRoom, _ := http.NewRequest(http.MethodGet, "/api/admin/rooms/"+seededRoomAID, nil)
	reqRoom.Header.Set("Authorization", "Bearer "+hotelBJWT)
	wRoom := httptest.NewRecorder()
	testRouter.ServeHTTP(wRoom, reqRoom)

	if wRoom.Code != http.StatusNotFound {
		t.Errorf("security violation: Hotel B admin could access Hotel A's room, got code %d", wRoom.Code)
	}

	// Hotel B Admin attempts to update Hotel A's Category -> 404
	payload := map[string]interface{}{"name": "Hacked Category"}
	body, _ := json.Marshal(payload)
	reqCat, _ := http.NewRequest(http.MethodPut, "/api/admin/room-categories/"+seededCatAID, bytes.NewBuffer(body))
	reqCat.Header.Set("Content-Type", "application/json")
	reqCat.Header.Set("Authorization", "Bearer "+hotelBJWT)
	wCat := httptest.NewRecorder()
	testRouter.ServeHTTP(wCat, reqCat)

	if wCat.Code != http.StatusNotFound {
		t.Errorf("security violation: Hotel B admin could mutate Hotel A's category, got code %d", wCat.Code)
	}

	// Hotel B queries rooms -> should only see 0 rooms (none created for Hotel B yet)
	reqList, _ := http.NewRequest(http.MethodGet, "/api/admin/rooms", nil)
	reqList.Header.Set("Authorization", "Bearer "+hotelBJWT)
	wList := httptest.NewRecorder()
	testRouter.ServeHTTP(wList, reqList)

	var resList map[string]interface{}
	json.Unmarshal(wList.Body.Bytes(), &resList)
	bRooms := resList["data"].([]interface{})
	if len(bRooms) != 0 {
		t.Errorf("security violation: Hotel B saw %d rooms belonging to Hotel A", len(bRooms))
	}
}
