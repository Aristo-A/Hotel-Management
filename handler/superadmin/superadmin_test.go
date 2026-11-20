package superadmin

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
	testRouter    *gin.Engine
	superAdminJWT string
	hotelAdminJWT string
	seededHotelID string
	seededAdminID string
)

func setupSuperAdminTestDB(t *testing.T) {
	os.Setenv("JWT_SECRET", "superadmin_test_secret_key_1234567890")
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

	// 1. Create Hotel
	hotel := models.Hotel{
		Name:         "Grand Horizon Hotel",
		Description:  "Luxury 5-star hotel facing Marine Drive",
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

	// 2. Hash Password
	hashed, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	// 3. SuperAdmin user
	superUser := models.User{
		Name:         "Super Admin User",
		Email:        "superadmin@hotelportal.com",
		PasswordHash: string(hashed),
		Role:         "superadmin",
		Status:       "active",
	}
	db.Create(&superUser)

	// 4. HotelAdmin user
	hotelAdminUser := models.User{
		Name:         "Arjun Sharma",
		Email:        "admin@grandhorizon.com",
		PasswordHash: string(hashed),
		Role:         "hotel_admin",
		Status:       "active",
	}
	db.Create(&hotelAdminUser)
	seededAdminID = hotelAdminUser.ID

	// Assign admin to hotel
	assignment := models.HotelAdminAssignment{
		UserID:  hotelAdminUser.ID,
		HotelID: hotel.ID,
	}
	db.Create(&assignment)

	// 5. Seed Category & Rooms
	cat := models.RoomCategory{
		HotelID:      hotel.ID,
		Name:         "Deluxe Suite",
		DefaultRate:  5000,
		MaxOccupancy: 2,
	}
	db.Create(&cat)

	room1 := models.Room{
		HotelID:    hotel.ID,
		CategoryID: cat.ID,
		RoomNumber: "101",
		Rate:       5000,
		Status:     "occupied",
	}
	db.Create(&room1)

	room2 := models.Room{
		HotelID:    hotel.ID,
		CategoryID: cat.ID,
		RoomNumber: "102",
		Rate:       5000,
		Status:     "available",
	}
	db.Create(&room2)

	// 6. Seed Payment
	payment := models.Payment{
		HotelID: hotel.ID,
		Amount:  11200,
		Status:  "completed",
		Method:  "credit_card",
	}
	db.Create(&payment)

	// Generate JWTs
	sToken, err := util.GenerateToken(superUser.ID, superUser.Name, superUser.Email, superUser.Role, nil, "")
	if err != nil {
		t.Fatalf("failed to generate superadmin token: %v", err)
	}
	superAdminJWT = sToken

	hToken, err := util.GenerateToken(hotelAdminUser.ID, hotelAdminUser.Name, hotelAdminUser.Email, hotelAdminUser.Role, &hotel.ID, hotel.Name)
	if err != nil {
		t.Fatalf("failed to generate hoteladmin token: %v", err)
	}
	hotelAdminJWT = hToken

	// Setup Router with superadmin routes
	testRouter = gin.New()
	superGroup := testRouter.Group("/api/superadmin")
	superGroup.Use(middleware.RequireAuth(), middleware.RequireRole("superadmin"))
	{
		superGroup.GET("/dashboard", GetDashboard)
		superGroup.GET("/hotels", GetHotels)
		superGroup.POST("/hotels", CreateHotel)
		superGroup.GET("/hotels/:id", GetHotelByID)
		superGroup.PUT("/hotels/:id", UpdateHotel)
		superGroup.DELETE("/hotels/:id", DeleteHotel)
		superGroup.PATCH("/hotels/:id/status", UpdateHotelStatus)
		superGroup.GET("/hotels/:id/admins", GetHotelAdmins)

		superGroup.GET("/admins", GetAdmins)
		superGroup.POST("/admins", CreateAdmin)
		superGroup.GET("/admins/:id", GetAdminByID)
		superGroup.PUT("/admins/:id", UpdateAdmin)
		superGroup.PATCH("/admins/:id/status", UpdateAdminStatus)
		superGroup.DELETE("/admins/:id", DeleteAdmin)
	}
}

func TestSuperAdmin_Dashboard(t *testing.T) {
	setupSuperAdminTestDB(t)

	req, _ := http.NewRequest(http.MethodGet, "/api/superadmin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)

	if res["success"] != true {
		t.Errorf("expected success: true, got %v", res["success"])
	}

	totalHotels, ok := res["totalHotels"].(float64)
	if !ok || totalHotels != 1 {
		t.Errorf("expected totalHotels 1, got %v", res["totalHotels"])
	}

	totalAdmins, ok := res["totalAdmins"].(float64)
	if !ok || totalAdmins != 1 {
		t.Errorf("expected totalAdmins 1, got %v", res["totalAdmins"])
	}

	occupancyRate, ok := res["occupancyRate"].(float64)
	if !ok || occupancyRate != 50.0 {
		t.Errorf("expected occupancyRate 50.0%% (1/2 rooms), got %v", res["occupancyRate"])
	}

	totalRevenue, ok := res["totalRevenue"].(float64)
	if !ok || totalRevenue != 11200 {
		t.Errorf("expected totalRevenue 11200, got %v", res["totalRevenue"])
	}
}

func TestSuperAdmin_GetHotels(t *testing.T) {
	setupSuperAdminTestDB(t)

	// 1. Get all hotels
	req, _ := http.NewRequest(http.MethodGet, "/api/superadmin/hotels", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)

	hotelsList, ok := res["data"].([]interface{})
	if !ok || len(hotelsList) != 1 {
		t.Fatalf("expected 1 hotel in list, got %v", len(hotelsList))
	}

	hotel0 := hotelsList[0].(map[string]interface{})
	if hotel0["name"] != "Grand Horizon Hotel" {
		t.Errorf("expected hotel name 'Grand Horizon Hotel', got %v", hotel0["name"])
	}
	if hotel0["totalRooms"].(float64) != 2 {
		t.Errorf("expected totalRooms 2, got %v", hotel0["totalRooms"])
	}
	if hotel0["availableRooms"].(float64) != 1 {
		t.Errorf("expected availableRooms 1, got %v", hotel0["availableRooms"])
	}
	if hotel0["adminName"] != "Arjun Sharma" {
		t.Errorf("expected adminName 'Arjun Sharma', got %v", hotel0["adminName"])
	}

	// 2. Search filter matching
	reqSearch, _ := http.NewRequest(http.MethodGet, "/api/superadmin/hotels?search=Mumbai", nil)
	reqSearch.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wSearch := httptest.NewRecorder()
	testRouter.ServeHTTP(wSearch, reqSearch)
	if wSearch.Code != http.StatusOK {
		t.Fatalf("search expected 200, got %d", wSearch.Code)
	}

	// 3. Search filter non-matching
	reqSearchNone, _ := http.NewRequest(http.MethodGet, "/api/superadmin/hotels?search=NonExistentCity", nil)
	reqSearchNone.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wSearchNone := httptest.NewRecorder()
	testRouter.ServeHTTP(wSearchNone, reqSearchNone)
	json.Unmarshal(wSearchNone.Body.Bytes(), &res)
	noneList := res["data"].([]interface{})
	if len(noneList) != 0 {
		t.Errorf("expected 0 hotels for non-matching search, got %d", len(noneList))
	}
}

func TestSuperAdmin_CreateHotel(t *testing.T) {
	setupSuperAdminTestDB(t)

	// Valid creation
	payload := map[string]interface{}{
		"name":        "Seaside Paradise Resort",
		"description": "Beachfront resort with ocean views",
		"address":     "Calangute Beach Road",
		"city":        "Goa",
		"state":       "Goa",
		"country":     "India",
		"phone":       "+91 832 227 1234",
		"email":       "stay@seasideparadise.in",
		"rating":      5,
		"adminId":     seededAdminID,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/api/superadmin/hotels", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	hotelData := res["data"].(map[string]interface{})
	if hotelData["name"] != "Seaside Paradise Resort" {
		t.Errorf("expected hotel name Seaside Paradise Resort, got %v", hotelData["name"])
	}
	if hotelData["adminName"] != "Arjun Sharma" {
		t.Errorf("expected admin assigned, got %v", hotelData["adminName"])
	}

	// Invalid creation (missing required address and city)
	invalidPayload := map[string]interface{}{
		"name": "Incomplete Hotel",
	}
	invBody, _ := json.Marshal(invalidPayload)
	reqInv, _ := http.NewRequest(http.MethodPost, "/api/superadmin/hotels", bytes.NewBuffer(invBody))
	reqInv.Header.Set("Content-Type", "application/json")
	reqInv.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wInv := httptest.NewRecorder()
	testRouter.ServeHTTP(wInv, reqInv)
	if wInv.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid hotel payload, got %d", wInv.Code)
	}
}

func TestSuperAdmin_GetHotelByID(t *testing.T) {
	setupSuperAdminTestDB(t)

	// Existing hotel
	req, _ := http.NewRequest(http.MethodGet, "/api/superadmin/hotels/"+seededHotelID, nil)
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Non-existing hotel
	reqBad, _ := http.NewRequest(http.MethodGet, "/api/superadmin/hotels/non-existent-uuid-12345", nil)
	reqBad.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wBad := httptest.NewRecorder()
	testRouter.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", wBad.Code)
	}
}

func TestSuperAdmin_UpdateHotel_And_Status(t *testing.T) {
	setupSuperAdminTestDB(t)

	// 1. Update Hotel Info
	updatePayload := map[string]interface{}{
		"name":        "Grand Horizon Palace & Spa",
		"description": "Newly renovated 5-star palace",
		"address":     "Marine Drive, Nariman Point",
		"city":        "Mumbai",
		"phone":       "+91 22 9999 8888",
		"email":       "contact@grandhorizon.com",
	}
	body, _ := json.Marshal(updatePayload)
	req, _ := http.NewRequest(http.MethodPut, "/api/superadmin/hotels/"+seededHotelID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	hotelData := res["data"].(map[string]interface{})
	if hotelData["name"] != "Grand Horizon Palace & Spa" {
		t.Errorf("expected updated name, got %v", hotelData["name"])
	}

	// 2. Toggle Status to inactive
	statusPayload := map[string]interface{}{
		"status": "inactive",
	}
	stBody, _ := json.Marshal(statusPayload)
	reqSt, _ := http.NewRequest(http.MethodPatch, "/api/superadmin/hotels/"+seededHotelID+"/status", bytes.NewBuffer(stBody))
	reqSt.Header.Set("Content-Type", "application/json")
	reqSt.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wSt := httptest.NewRecorder()
	testRouter.ServeHTTP(wSt, reqSt)

	if wSt.Code != http.StatusOK {
		t.Fatalf("expected 200 on status patch, got %d", wSt.Code)
	}

	var stRes map[string]interface{}
	json.Unmarshal(wSt.Body.Bytes(), &stRes)
	if stRes["status"] != "inactive" {
		t.Errorf("expected status 'inactive', got %v", stRes["status"])
	}
}

func TestSuperAdmin_DeleteHotel(t *testing.T) {
	setupSuperAdminTestDB(t)

	// 1. Add active reservation to hotel
	resv := models.Reservation{
		BookingReference: "BK-TEST-ACTIVE",
		HotelID:          seededHotelID,
		GuestName:        "Suresh Patel",
		GuestEmail:       "suresh@example.com",
		GuestPhone:       "+91 98000 11111",
		CheckIn:          time.Now(),
		CheckOut:         time.Now().Add(48 * time.Hour),
		Nights:           2,
		Status:           "confirmed",
	}
	database.DB.Create(&resv)

	// Attempt delete should fail with 400
	req, _ := http.NewRequest(http.MethodDelete, "/api/superadmin/hotels/"+seededHotelID, nil)
	req.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when deleting hotel with active bookings, got %d: %s", w.Code, w.Body.String())
	}

	// Cancel reservation and retry
	database.DB.Model(&resv).Update("status", "cancelled")
	req2, _ := http.NewRequest(http.MethodDelete, "/api/superadmin/hotels/"+seededHotelID, nil)
	req2.Header.Set("Authorization", "Bearer "+superAdminJWT)
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 when deleting hotel without active bookings, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestSuperAdmin_Admins_Lifecycle(t *testing.T) {
	setupSuperAdminTestDB(t)

	// 1. Create duplicate admin email -> fail
	dupPayload := map[string]interface{}{
		"name":     "Duplicate Admin",
		"email":    "admin@grandhorizon.com",
		"phone":    "+91 99999 11111",
		"password": "Password123!",
		"hotelId":  seededHotelID,
	}
	dupBody, _ := json.Marshal(dupPayload)
	reqDup, _ := http.NewRequest(http.MethodPost, "/api/superadmin/admins", bytes.NewBuffer(dupBody))
	reqDup.Header.Set("Content-Type", "application/json")
	reqDup.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wDup := httptest.NewRecorder()
	testRouter.ServeHTTP(wDup, reqDup)
	if wDup.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate email, got %d", wDup.Code)
	}

	// 2. Create new Admin with short password -> fail
	shortPayload := map[string]interface{}{
		"name":     "Short Pass Admin",
		"email":    "short@example.com",
		"phone":    "+91 99999 11111",
		"password": "pass",
		"hotelId":  seededHotelID,
	}
	shortBody, _ := json.Marshal(shortPayload)
	reqShort, _ := http.NewRequest(http.MethodPost, "/api/superadmin/admins", bytes.NewBuffer(shortBody))
	reqShort.Header.Set("Content-Type", "application/json")
	reqShort.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wShort := httptest.NewRecorder()
	testRouter.ServeHTTP(wShort, reqShort)
	if wShort.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", wShort.Code)
	}

	// 3. Create Admin Successfully
	validPayload := map[string]interface{}{
		"name":     "Priya Nair",
		"email":    "priya@grandhorizon.com",
		"phone":    "+91 98333 44444",
		"password": "SecurePassword123!",
		"hotelId":  seededHotelID,
		"status":   "active",
	}
	valBody, _ := json.Marshal(validPayload)
	reqVal, _ := http.NewRequest(http.MethodPost, "/api/superadmin/admins", bytes.NewBuffer(valBody))
	reqVal.Header.Set("Content-Type", "application/json")
	reqVal.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wVal := httptest.NewRecorder()
	testRouter.ServeHTTP(wVal, reqVal)

	if wVal.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", wVal.Code, wVal.Body.String())
	}

	var resVal map[string]interface{}
	json.Unmarshal(wVal.Body.Bytes(), &resVal)
	adminData := resVal["data"].(map[string]interface{})
	newAdminID := adminData["id"].(string)

	if adminData["hotelName"] != "Grand Horizon Hotel" {
		t.Errorf("expected hotelName 'Grand Horizon Hotel', got %v", adminData["hotelName"])
	}

	// Verify password hash is never exposed
	if _, exists := adminData["passwordHash"]; exists {
		t.Errorf("security violation: passwordHash exposed in admin JSON")
	}

	// 4. Get Admins List
	reqList, _ := http.NewRequest(http.MethodGet, "/api/superadmin/admins", nil)
	reqList.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wList := httptest.NewRecorder()
	testRouter.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wList.Code)
	}

	// 5. Update Admin
	updPayload := map[string]interface{}{
		"name":  "Priya N. Sharma",
		"phone": "+91 98333 55555",
	}
	updBody, _ := json.Marshal(updPayload)
	reqUpd, _ := http.NewRequest(http.MethodPut, "/api/superadmin/admins/"+newAdminID, bytes.NewBuffer(updBody))
	reqUpd.Header.Set("Content-Type", "application/json")
	reqUpd.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wUpd := httptest.NewRecorder()
	testRouter.ServeHTTP(wUpd, reqUpd)
	if wUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 on update admin, got %d", wUpd.Code)
	}

	// 6. Toggle Status
	reqToggle, _ := http.NewRequest(http.MethodPatch, "/api/superadmin/admins/"+newAdminID+"/status", nil)
	reqToggle.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wToggle := httptest.NewRecorder()
	testRouter.ServeHTTP(wToggle, reqToggle)
	if wToggle.Code != http.StatusOK {
		t.Fatalf("expected 200 on status toggle, got %d", wToggle.Code)
	}

	var toggleRes map[string]interface{}
	json.Unmarshal(wToggle.Body.Bytes(), &toggleRes)
	if toggleRes["status"] != "inactive" {
		t.Errorf("expected admin status 'inactive', got %v", toggleRes["status"])
	}

	// 7. Delete Admin
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/superadmin/admins/"+newAdminID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+superAdminJWT)
	wDel := httptest.NewRecorder()
	testRouter.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete admin, got %d", wDel.Code)
	}
}

func TestSuperAdmin_RBAC_Protection(t *testing.T) {
	setupSuperAdminTestDB(t)

	// 1. HotelAdmin attempts SuperAdmin endpoint -> 403 Forbidden
	req, _ := http.NewRequest(http.MethodGet, "/api/superadmin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+hotelAdminJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for hotel_admin calling superadmin, got %d", w.Code)
	}

	// 2. Unauthenticated request -> 401 Unauthorized
	reqAnon, _ := http.NewRequest(http.MethodGet, "/api/superadmin/dashboard", nil)
	wAnon := httptest.NewRecorder()
	testRouter.ServeHTTP(wAnon, reqAnon)

	if wAnon.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing token, got %d", wAnon.Code)
	}
}
