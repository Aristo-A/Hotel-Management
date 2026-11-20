package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"hotel-backend/database"
	"hotel-backend/middleware"
	"hotel-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var testRouter *gin.Engine

func setupAuthTestDB(t *testing.T) {
	os.Setenv("JWT_SECRET", "super_secure_auth_test_secret_key_12345")
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
		&models.ActivityLog{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test tables: %v", err)
	}

	// 1. Create demo Hotel
	hotel := models.Hotel{
		Name:    "Grand Horizon Hotel",
		Address: "Marine Drive",
		City:    "Mumbai",
		State:   "Maharashtra",
		Phone:   "+91 22 5555 6666",
		Email:   "mumbai@grandhorizon.in",
		Status:  "active",
	}
	db.Create(&hotel)

	// Hash password
	hashedPass, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	// 2. Create SuperAdmin
	superAdmin := models.User{
		Name:         "Super Admin User",
		Email:        "superadmin@hotelgroup.com",
		PasswordHash: string(hashedPass),
		Role:         "superadmin",
		Status:       "active",
	}
	db.Create(&superAdmin)

	// 3. Create Hotel Admin
	hotelAdmin := models.User{
		Name:         "Arjun Sharma",
		Email:        "admin@grandhorizon.com",
		PasswordHash: string(hashedPass),
		Role:         "hotel_admin",
		Status:       "active",
	}
	db.Create(&hotelAdmin)

	// Assign to hotel
	assignment := models.HotelAdminAssignment{
		UserID:  hotelAdmin.ID,
		HotelID: hotel.ID,
	}
	db.Create(&assignment)

	// 4. Create Deactivated User
	deactivated := models.User{
		Name:         "Deactivated Admin",
		Email:        "deactivated@hotel.com",
		PasswordHash: string(hashedPass),
		Role:         "hotel_admin",
		Status:       "inactive",
	}
	db.Create(&deactivated)

	// Setup routes
	testRouter = gin.New()
	authGroup := testRouter.Group("/api/auth")
	{
		authGroup.POST("/login", Login)
		authGroup.POST("/logout", middleware.RequireAuth(), Logout)
		authGroup.GET("/me", middleware.RequireAuth(), Me)
		authGroup.PUT("/profile", middleware.RequireAuth(), UpdateProfile)
		authGroup.PATCH("/password", middleware.RequireAuth(), ChangePassword)
		authGroup.POST("/refresh", middleware.RequireAuth(), Refresh)
	}
}

func TestLogin_HotelAdmin_Success(t *testing.T) {
	setupAuthTestDB(t)

	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d, body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected token in response")
	}

	userMap, ok := resp["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object in response")
	}

	if userMap["email"] != "admin@grandhorizon.com" {
		t.Errorf("expected admin@grandhorizon.com, got: %v", userMap["email"])
	}
	if userMap["role"] != "hotel_admin" {
		t.Errorf("expected role hotel_admin, got: %v", userMap["role"])
	}
	if userMap["hotelName"] != "Grand Horizon Hotel" {
		t.Errorf("expected hotelName Grand Horizon Hotel, got: %v", userMap["hotelName"])
	}
	if userMap["hotelId"] == nil || userMap["hotelId"] == "" {
		t.Error("expected non-empty hotelId for hotel_admin")
	}
}

func TestLogin_SuperAdmin_Success(t *testing.T) {
	setupAuthTestDB(t)

	payload := map[string]string{
		"email":    "superadmin@hotelgroup.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got: %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	userMap := resp["user"].(map[string]interface{})
	if userMap["role"] != "superadmin" {
		t.Errorf("expected role superadmin, got: %v", userMap["role"])
	}
	if userMap["hotelId"] != nil {
		t.Errorf("expected nil hotelId for superadmin, got: %v", userMap["hotelId"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	setupAuthTestDB(t)

	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "WrongPassword999!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for wrong password, got: %d", w.Code)
	}
}

func TestLogin_NonExistentEmail(t *testing.T) {
	setupAuthTestDB(t)

	payload := map[string]string{
		"email":    "nobody@nowhere.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unknown email, got: %d", w.Code)
	}
}

func TestLogin_DeactivatedAccount(t *testing.T) {
	setupAuthTestDB(t)

	payload := map[string]string{
		"email":    "deactivated@hotel.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for deactivated account, got: %d", w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	setupAuthTestDB(t)

	// 1. First login to obtain token
	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// 2. Call /api/auth/me with Bearer token
	meReq, _ := http.NewRequest("GET", "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meW := httptest.NewRecorder()
	testRouter.ServeHTTP(meW, meReq)

	if meW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/me, got: %d", meW.Code)
	}

	var meResp map[string]interface{}
	json.Unmarshal(meW.Body.Bytes(), &meResp)

	if meResp["name"] != "Arjun Sharma" {
		t.Errorf("expected name Arjun Sharma, got: %v", meResp["name"])
	}
	if meResp["role"] != "hotel_admin" {
		t.Errorf("expected role hotel_admin, got: %v", meResp["role"])
	}
	if meResp["hotelName"] != "Grand Horizon Hotel" {
		t.Errorf("expected hotelName Grand Horizon Hotel, got: %v", meResp["hotelName"])
	}
}

func TestMe_MissingToken(t *testing.T) {
	setupAuthTestDB(t)

	req, _ := http.NewRequest("GET", "/api/auth/me", nil)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for missing token, got: %d", w.Code)
	}
}

func TestLogout_Success(t *testing.T) {
	setupAuthTestDB(t)

	// Obtain token
	payload := map[string]string{
		"email":    "superadmin@hotelgroup.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	logoutReq, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutW := httptest.NewRecorder()
	testRouter.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Errorf("expected 200 OK for logout, got: %d", logoutW.Code)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	setupAuthTestDB(t)

	// Obtain token
	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// Update profile
	updatePayload := map[string]string{
		"name":  "Arjun K. Sharma",
		"phone": "+91 99999 88888",
	}
	upBody, _ := json.Marshal(updatePayload)
	req, _ := http.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(upBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for update profile, got: %d, body: %s", w.Code, w.Body.String())
	}

	var updateResp struct {
		Success bool `json:"success"`
		Data    struct {
			Name  string `json:"name"`
			Phone string `json:"phone"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	if updateResp.Data.Name != "Arjun K. Sharma" {
		t.Errorf("expected updated name 'Arjun K. Sharma', got: %s", updateResp.Data.Name)
	}
}

func TestChangePassword_Success(t *testing.T) {
	setupAuthTestDB(t)

	// Obtain token
	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// Change password
	pwPayload := map[string]string{
		"currentPassword": "Password123!",
		"newPassword":     "BrandNewSecurePass999!",
	}
	pwBody, _ := json.Marshal(pwPayload)
	req, _ := http.NewRequest("PATCH", "/api/auth/password", bytes.NewBuffer(pwBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for change password, got: %d, body: %s", w.Code, w.Body.String())
	}

	// Verify login with new password succeeds
	newLoginPayload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "BrandNewSecurePass999!",
	}
	nlBody, _ := json.Marshal(newLoginPayload)
	nlReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(nlBody))
	nlReq.Header.Set("Content-Type", "application/json")
	nlW := httptest.NewRecorder()
	testRouter.ServeHTTP(nlW, nlReq)

	if nlW.Code != http.StatusOK {
		t.Errorf("expected 200 OK with new password, got: %d", nlW.Code)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	setupAuthTestDB(t)

	// Obtain token
	payload := map[string]string{
		"email":    "superadmin@hotelgroup.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	req, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for refresh, got: %d", w.Code)
	}

	var refreshResp struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &refreshResp)
	if refreshResp.Token == "" {
		t.Errorf("expected non-empty refreshed token")
	}
}

func TestAuth_Validation_Guards(t *testing.T) {
	setupAuthTestDB(t)

	// Obtain token
	payload := map[string]string{
		"email":    "admin@grandhorizon.com",
		"password": "Password123!",
	}
	body, _ := json.Marshal(payload)
	loginReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	var loginResp map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	token := loginResp["token"].(string)

	// 1. Same new and current password -> 400
	samePwPayload := map[string]string{
		"currentPassword": "Password123!",
		"newPassword":     "Password123!",
	}
	samePwBody, _ := json.Marshal(samePwPayload)
	reqSame, _ := http.NewRequest("PATCH", "/api/auth/password", bytes.NewBuffer(samePwBody))
	reqSame.Header.Set("Authorization", "Bearer "+token)
	reqSame.Header.Set("Content-Type", "application/json")
	wSame := httptest.NewRecorder()
	testRouter.ServeHTTP(wSame, reqSame)

	if wSame.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when setting new password identical to current, got %d", wSame.Code)
	}

	// 2. Profile update with empty or 1-char name -> 400
	shortNamePayload := map[string]string{
		"name": "A",
	}
	snBody, _ := json.Marshal(shortNamePayload)
	reqSN, _ := http.NewRequest("PUT", "/api/auth/profile", bytes.NewBuffer(snBody))
	reqSN.Header.Set("Authorization", "Bearer "+token)
	reqSN.Header.Set("Content-Type", "application/json")
	wSN := httptest.NewRecorder()
	testRouter.ServeHTTP(wSN, reqSN)

	if wSN.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for name shorter than 2 chars, got %d", wSN.Code)
	}
}
