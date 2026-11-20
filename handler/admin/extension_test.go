package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotel-backend/database"
	"hotel-backend/middleware"
	"hotel-backend/models"

	"github.com/gin-gonic/gin"
)

func setupExtensionRouter() {
	testRouter = gin.New()
	adminGroup := testRouter.Group("/api/admin")
	adminGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		// Housekeeping
		adminGroup.GET("/housekeeping", GetHousekeepingTasks)
		adminGroup.PATCH("/housekeeping/:roomId", UpdateHousekeepingTask)

		// Facility Charges
		adminGroup.GET("/facility-charges", GetFacilityCharges)
		adminGroup.POST("/facility-charges", CreateFacilityCharge)
		adminGroup.DELETE("/facility-charges/:id", DeleteFacilityCharge)

		// Billing & Payments
		adminGroup.GET("/bills/:id", GetBillByID)
		adminGroup.GET("/bills/:id/print", GetPrintBill)
		adminGroup.POST("/bills/:id/payments", CreateBillPaymentByBillID)

		// Reports
		adminGroup.GET("/reports/room-status", GetRoomStatusReport)
	}

	v1HotelGroup := testRouter.Group("/api/v1/hotel")
	v1HotelGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		v1HotelGroup.GET("/housekeeping", GetHousekeepingTasks)
		v1HotelGroup.PATCH("/housekeeping/:roomId", UpdateHousekeepingTask)
		v1HotelGroup.GET("/facility-charges", GetFacilityCharges)
		v1HotelGroup.POST("/facility-charges", CreateFacilityCharge)
		v1HotelGroup.DELETE("/facility-charges/:id", DeleteFacilityCharge)
		v1HotelGroup.GET("/bills/:id/print", GetPrintBill)
		v1HotelGroup.POST("/bills/:id/payments", CreateBillPaymentByBillID)
		v1HotelGroup.GET("/reports/room-status", GetRoomStatusReport)
	}
}

func TestHousekeeping_Tasks_And_StatusUpdate(t *testing.T) {
	setupInventoryTestDB(t)
	setupExtensionRouter()

	// Initially room status is available; mark room dirty
	database.DB.Model(&models.Room{}).Where("id = ?", seededRoomAID).Update("status", "dirty")

	// 1. GET /api/admin/housekeeping
	req, _ := http.NewRequest("GET", "/api/admin/housekeeping", nil)
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for housekeeping tasks, got %d, body: %s", w.Code, w.Body.String())
	}

	var getResp struct {
		Success bool                  `json:"success"`
		Data    []HousekeepingTaskDTO `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &getResp)
	if len(getResp.Data) == 0 {
		t.Fatalf("expected at least 1 housekeeping task, got 0")
	}

	// 2. PATCH /api/admin/housekeeping/:roomId -> update status to cleaning
	updatePayload := map[string]string{
		"taskStatus": "in_progress",
		"assignedTo": "Meena Devi",
		"notes":      "Deep clean in progress",
	}
	body, _ := json.Marshal(updatePayload)
	reqPatch, _ := http.NewRequest("PATCH", "/api/admin/housekeeping/"+seededRoomAID, bytes.NewBuffer(body))
	reqPatch.Header.Set("Authorization", "Bearer "+hotelAJWT)
	reqPatch.Header.Set("Content-Type", "application/json")
	wPatch := httptest.NewRecorder()
	testRouter.ServeHTTP(wPatch, reqPatch)

	if wPatch.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for housekeeping patch, got %d, body: %s", wPatch.Code, wPatch.Body.String())
	}

	var roomCheck models.Room
	database.DB.Where("id = ?", seededRoomAID).First(&roomCheck)
	if roomCheck.Status != "cleaning" {
		t.Errorf("expected room status to transition to 'cleaning', got: %s", roomCheck.Status)
	}

	// 3. PATCH /api/v1/hotel/housekeeping/:roomId -> mark completed
	completePayload := map[string]string{
		"taskStatus": "completed",
		"notes":      "Room cleaned and sanitized",
	}
	bodyComp, _ := json.Marshal(completePayload)
	reqComp, _ := http.NewRequest("PATCH", "/api/v1/hotel/housekeeping/"+seededRoomAID, bytes.NewBuffer(bodyComp))
	reqComp.Header.Set("Authorization", "Bearer "+hotelAJWT)
	reqComp.Header.Set("Content-Type", "application/json")
	wComp := httptest.NewRecorder()
	testRouter.ServeHTTP(wComp, reqComp)

	if wComp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for complete patch, got %d, body: %s", wComp.Code, wComp.Body.String())
	}

	database.DB.Where("id = ?", seededRoomAID).First(&roomCheck)
	if roomCheck.Status != "available" {
		t.Errorf("expected room status to transition to 'available' on completion, got: %s", roomCheck.Status)
	}
}

func TestFacilityCharges_Lifecycle_And_BillRecalculation(t *testing.T) {
	setupInventoryTestDB(t)
	setupExtensionRouter()

	// Create checked-in reservation with bill
	checkIn := time.Now().AddDate(0, 0, -1)
	checkOut := time.Now().AddDate(0, 0, 2)
	resv := models.Reservation{
		HotelID:          hotelAID,
		RoomID:           &seededRoomAID,
		BookingReference: "BK-2024-FAC01",
		GuestName:        "Test Guest",
		GuestEmail:       "test@example.com",
		GuestPhone:       "+91 99999 11111",
		CheckIn:          checkIn,
		CheckOut:         checkOut,
		Nights:           3,
		RoomRate:         5000.0,
		TotalAmount:      16800.0,
		Status:           "checked_in",
	}
	database.DB.Create(&resv)

	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: resv.ID,
		GuestName:     resv.GuestName,
		Subtotal:      15000.0,
		TaxRate:       0.12,
		TaxAmount:     1800.0,
		TotalAmount:   16800.0,
		Balance:       16800.0,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	// 1. POST /api/admin/facility-charges
	chargePayload := map[string]interface{}{
		"reservationId": resv.ID,
		"category":      "restaurant",
		"description":   "Buffet Breakfast x2",
		"amount":        1200.0,
	}
	body, _ := json.Marshal(chargePayload)
	req, _ := http.NewRequest("POST", "/api/admin/facility-charges", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for facility charge, got %d, body: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Success bool              `json:"success"`
		Data    FacilityChargeDTO `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	chargeID := createResp.Data.ID

	// Verify bill was recalculated: subtotal increased from 15000 to 16200
	var updatedBill models.Bill
	database.DB.Where("id = ?", bill.ID).First(&updatedBill)
	if updatedBill.Subtotal != 16200.0 {
		t.Errorf("expected bill subtotal to be 16200.0, got: %.2f", updatedBill.Subtotal)
	}

	// 2. GET /api/v1/hotel/facility-charges
	reqGet, _ := http.NewRequest("GET", "/api/v1/hotel/facility-charges?reservationId="+resv.ID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wGet := httptest.NewRecorder()
	testRouter.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for facility charge list, got %d", wGet.Code)
	}

	// 3. DELETE /api/admin/facility-charges/:id
	reqDel, _ := http.NewRequest("DELETE", "/api/admin/facility-charges/"+chargeID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDel := httptest.NewRecorder()
	testRouter.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for delete, got %d", wDel.Code)
	}
}

func TestPrintBill_Enrichment(t *testing.T) {
	setupInventoryTestDB(t)
	setupExtensionRouter()

	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: "res-print-1",
		GuestName:     "Priya Sharma",
		Subtotal:      10000.0,
		TaxRate:       0.12,
		TaxAmount:     1200.0,
		TotalAmount:   11200.0,
		AmountPaid:    11200.0,
		Balance:       0.0,
		Status:        "paid",
	}
	database.DB.Create(&bill)

	// GET /api/v1/hotel/bills/:id/print
	req, _ := http.NewRequest("GET", "/api/v1/hotel/bills/"+bill.ID+"/print", nil)
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for print bill, got %d, body: %s", w.Code, w.Body.String())
	}

	var printResp struct {
		Success bool `json:"success"`
		Data    struct {
			TotalAmount float64                `json:"totalAmount"`
			Hotel       map[string]interface{} `json:"hotel"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &printResp)

	if printResp.Data.TotalAmount != 11200.0 {
		t.Errorf("expected totalAmount 11200.0, got: %.2f", printResp.Data.TotalAmount)
	}
	if printResp.Data.Hotel["name"] == "" {
		t.Errorf("expected hotel name to be populated in print bill")
	}
}

func TestBillPayment_ByBillID(t *testing.T) {
	setupInventoryTestDB(t)
	setupExtensionRouter()

	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: "res-pay-1",
		GuestName:     "Rahul Verma",
		Subtotal:      10000.0,
		TaxRate:       0.12,
		TaxAmount:     1200.0,
		TotalAmount:   11200.0,
		AmountPaid:    0.0,
		Balance:       11200.0,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	// POST /api/v1/hotel/bills/:id/payments
	payPayload := map[string]interface{}{
		"amount":          5000.0,
		"method":          "upi",
		"referenceNumber": "UPI-REF-998877",
	}
	body, _ := json.Marshal(payPayload)
	req, _ := http.NewRequest("POST", "/api/v1/hotel/bills/"+bill.ID+"/payments", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for bill payment, got %d, body: %s", w.Code, w.Body.String())
	}

	var updatedBill models.Bill
	database.DB.Where("id = ?", bill.ID).First(&updatedBill)
	if updatedBill.AmountPaid != 5000.0 {
		t.Errorf("expected amountPaid 5000.0, got: %.2f", updatedBill.AmountPaid)
	}
	if updatedBill.Balance != 6200.0 {
		t.Errorf("expected balance 6200.0, got: %.2f", updatedBill.Balance)
	}
	if updatedBill.Status != "partial" {
		t.Errorf("expected status 'partial', got: %s", updatedBill.Status)
	}
}

func TestRoomStatusReport(t *testing.T) {
	setupInventoryTestDB(t)
	setupExtensionRouter()

	// GET /api/v1/hotel/reports/room-status
	req, _ := http.NewRequest("GET", "/api/v1/hotel/reports/room-status", nil)
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for room status report, got %d", w.Code)
	}

	var reportResp struct {
		Success bool `json:"success"`
		Data    struct {
			Available int64 `json:"available"`
			Total     int64 `json:"total"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &reportResp)
	if reportResp.Data.Total < 1 {
		t.Errorf("expected total rooms >= 1, got %d", reportResp.Data.Total)
	}
}
