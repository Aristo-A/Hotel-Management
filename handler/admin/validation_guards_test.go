package admin

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
)

func TestReservation_Validation_Guards(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	now := time.Now()
	yesterday := now.AddDate(0, 0, -2).Format("2006-01-02")
	pastCheckOut := now.AddDate(0, 0, -1).Format("2006-01-02")
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	// 1. Past check-in date should be rejected with 400
	pastPayload := map[string]interface{}{
		"roomId":     seededRoomAID,
		"guestName":  "Past Traveler",
		"guestEmail": "past@example.com",
		"guestPhone": "+91 99999 11111",
		"checkIn":    yesterday,
		"checkOut":   pastCheckOut,
		"numGuests":  1,
	}
	pastBody, _ := json.Marshal(pastPayload)
	reqPast, _ := http.NewRequest(http.MethodPost, "/api/admin/reservations", bytes.NewBuffer(pastBody))
	reqPast.Header.Set("Content-Type", "application/json")
	reqPast.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPast := httptest.NewRecorder()
	testRouter.ServeHTTP(wPast, reqPast)

	if wPast.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for past check-in date, got %d", wPast.Code)
	}

	// 2. Overcapacity (Deluxe Suite has MaxOccupancy = 2, request 5 guests) -> 400
	overOccPayload := map[string]interface{}{
		"roomId":     seededRoomAID,
		"guestName":  "Big Family",
		"guestEmail": "family@example.com",
		"guestPhone": "+91 99999 22222",
		"checkIn":    today,
		"checkOut":   tomorrow,
		"numGuests":  5,
	}
	occBody, _ := json.Marshal(overOccPayload)
	reqOcc, _ := http.NewRequest(http.MethodPost, "/api/admin/reservations", bytes.NewBuffer(occBody))
	reqOcc.Header.Set("Content-Type", "application/json")
	reqOcc.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wOcc := httptest.NewRecorder()
	testRouter.ServeHTTP(wOcc, reqOcc)

	if wOcc.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for exceeding room capacity, got %d", wOcc.Code)
	}

	// 3. Create valid confirmed reservation
	validPayload := map[string]interface{}{
		"roomId":     seededRoomAID,
		"guestName":  "Valid Guest",
		"guestEmail": "valid@example.com",
		"guestPhone": "+91 99999 33333",
		"checkIn":    today,
		"checkOut":   tomorrow,
		"numGuests":  2,
	}
	validBody, _ := json.Marshal(validPayload)
	reqValid, _ := http.NewRequest(http.MethodPost, "/api/admin/reservations", bytes.NewBuffer(validBody))
	reqValid.Header.Set("Content-Type", "application/json")
	reqValid.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wValid := httptest.NewRecorder()
	testRouter.ServeHTTP(wValid, reqValid)

	if wValid.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid reservation, got %d: %s", wValid.Code, wValid.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(wValid.Body.Bytes(), &res)
	resvID := res["data"].(map[string]interface{})["id"].(string)

	// 4. Try checkout on a 'confirmed' reservation (not yet checked in) -> 400
	reqCheckOutEarly, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-out", nil)
	reqCheckOutEarly.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wCheckOutEarly := httptest.NewRecorder()
	testRouter.ServeHTTP(wCheckOutEarly, reqCheckOutEarly)
	if wCheckOutEarly.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for checking out un-checked-in reservation, got %d", wCheckOutEarly.Code)
	}

	// 5. Check in the reservation -> 200
	reqIn, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-in", nil)
	reqIn.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wIn := httptest.NewRecorder()
	testRouter.ServeHTTP(wIn, reqIn)
	if wIn.Code != http.StatusOK {
		t.Fatalf("expected 200 for check-in, got %d", wIn.Code)
	}

	// 6. Try cancelling a checked-in reservation -> 400
	reqCancelIn, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/cancel", nil)
	reqCancelIn.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wCancelIn := httptest.NewRecorder()
	testRouter.ServeHTTP(wCancelIn, reqCancelIn)
	if wCancelIn.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when cancelling an active checked-in reservation, got %d", wCancelIn.Code)
	}

	// 7. Check out the reservation -> 200
	reqOut, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-out", nil)
	reqOut.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wOut := httptest.NewRecorder()
	testRouter.ServeHTTP(wOut, reqOut)
	if wOut.Code != http.StatusOK {
		t.Fatalf("expected 200 for check-out, got %d", wOut.Code)
	}

	// 8. Try modifying a checked-out reservation -> 400
	updatePayload := map[string]interface{}{"guestName": "Attempt Rename"}
	updateBody, _ := json.Marshal(updatePayload)
	reqUpd, _ := http.NewRequest(http.MethodPut, "/api/admin/reservations/"+resvID, bytes.NewBuffer(updateBody))
	reqUpd.Header.Set("Content-Type", "application/json")
	reqUpd.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wUpd := httptest.NewRecorder()
	testRouter.ServeHTTP(wUpd, reqUpd)
	if wUpd.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when updating a checked-out reservation, got %d", wUpd.Code)
	}

	// 9. Try cancelling a checked-out reservation -> 400
	reqCancelOut, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/cancel", nil)
	reqCancelOut.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wCancelOut := httptest.NewRecorder()
	testRouter.ServeHTTP(wCancelOut, reqCancelOut)
	if wCancelOut.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when cancelling a checked-out reservation, got %d", wCancelOut.Code)
	}
}

func TestBilling_Validation_Guards(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	// 1. Create a bill
	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: "resv-guard-1",
		GuestName:     "Test Guest",
		Subtotal:      5000,
		TaxRate:       0.12,
		TaxAmount:     600,
		TotalAmount:   5600,
		Balance:       5600,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	// 2. Negative discount -> 400
	discNeg := map[string]interface{}{"discount": -100}
	discNegBody, _ := json.Marshal(discNeg)
	reqDiscNeg, _ := http.NewRequest(http.MethodPatch, "/api/admin/bills/"+bill.ID+"/discount", bytes.NewBuffer(discNegBody))
	reqDiscNeg.Header.Set("Content-Type", "application/json")
	reqDiscNeg.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDiscNeg := httptest.NewRecorder()
	testRouter.ServeHTTP(wDiscNeg, reqDiscNeg)
	if wDiscNeg.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative discount, got %d", wDiscNeg.Code)
	}

	// 3. Discount exceeding total amount -> 400
	discExcess := map[string]interface{}{"discount": 10000}
	discExcessBody, _ := json.Marshal(discExcess)
	reqDiscExcess, _ := http.NewRequest(http.MethodPatch, "/api/admin/bills/"+bill.ID+"/discount", bytes.NewBuffer(discExcessBody))
	reqDiscExcess.Header.Set("Content-Type", "application/json")
	reqDiscExcess.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDiscExcess := httptest.NewRecorder()
	testRouter.ServeHTTP(wDiscExcess, reqDiscExcess)
	if wDiscExcess.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for excessive discount, got %d", wDiscExcess.Code)
	}

	// 4. Negative payment amount -> 400
	payNeg := map[string]interface{}{
		"billId": bill.ID,
		"amount": -500,
		"method": "credit_card",
	}
	payNegBody, _ := json.Marshal(payNeg)
	reqPayNeg, _ := http.NewRequest(http.MethodPost, "/api/admin/payments", bytes.NewBuffer(payNegBody))
	reqPayNeg.Header.Set("Content-Type", "application/json")
	reqPayNeg.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPayNeg := httptest.NewRecorder()
	testRouter.ServeHTTP(wPayNeg, reqPayNeg)
	if wPayNeg.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for negative payment, got %d", wPayNeg.Code)
	}

	// 5. Pay full balance (settle bill) -> 201
	payFull := map[string]interface{}{
		"billId": bill.ID,
		"amount": 5600,
		"method": "upi",
	}
	payFullBody, _ := json.Marshal(payFull)
	reqPayFull, _ := http.NewRequest(http.MethodPost, "/api/admin/payments", bytes.NewBuffer(payFullBody))
	reqPayFull.Header.Set("Content-Type", "application/json")
	reqPayFull.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPayFull := httptest.NewRecorder()
	testRouter.ServeHTTP(wPayFull, reqPayFull)
	if wPayFull.Code != http.StatusCreated {
		t.Fatalf("expected 201 for full payment, got %d: %s", wPayFull.Code, wPayFull.Body.String())
	}

	// Verify bill is settled in DB
	database.DB.First(&bill, "id = ?", bill.ID)
	if bill.Status != "paid" || bill.Balance != 0 {
		t.Errorf("expected bill status 'paid' and balance 0, got %s and %f", bill.Status, bill.Balance)
	}

	// 6. Attempt another payment on settled bill -> 400
	reqPayExtra, _ := http.NewRequest(http.MethodPost, "/api/admin/payments", bytes.NewBuffer(payFullBody))
	reqPayExtra.Header.Set("Content-Type", "application/json")
	reqPayExtra.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPayExtra := httptest.NewRecorder()
	testRouter.ServeHTTP(wPayExtra, reqPayExtra)
	if wPayExtra.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for payment on already settled bill, got %d", wPayExtra.Code)
	}
}

func TestHousekeeping_Occupied_Preservation(t *testing.T) {
	setupInventoryTestDB(t)

	// Setup housekeeping route
	testRouter = gin.New()
	adminGroup := testRouter.Group("/api/admin")
	adminGroup.Use(func(c *gin.Context) {
		c.Set("hotelID", hotelAID)
		c.Set("userID", "admin-1")
		c.Next()
	})
	{
		adminGroup.PATCH("/housekeeping/:roomId", UpdateHousekeepingTask)
	}

	// Set seededRoomA status to occupied
	database.DB.Model(&models.Room{}).Where("id = ?", seededRoomAID).Update("status", "occupied")

	// Housekeeping marks occupied room as completed/clean
	hkPayload := map[string]interface{}{
		"taskStatus": "completed",
		"notes":      "Daily stayover cleaning done",
	}
	hkBody, _ := json.Marshal(hkPayload)
	reqHk, _ := http.NewRequest(http.MethodPatch, "/api/admin/housekeeping/"+seededRoomAID, bytes.NewBuffer(hkBody))
	reqHk.Header.Set("Content-Type", "application/json")
	wHk := httptest.NewRecorder()
	testRouter.ServeHTTP(wHk, reqHk)

	if wHk.Code != http.StatusOK {
		t.Fatalf("expected 200 on housekeeping update, got %d: %s", wHk.Code, wHk.Body.String())
	}

	// Verify room is STILL occupied (not changed to available while guest is in house)
	var room models.Room
	database.DB.First(&room, "id = ?", seededRoomAID)
	if room.Status != "occupied" {
		t.Errorf("expected room to stay 'occupied' during stayover cleaning, got %s", room.Status)
	}
}

func TestFacilityCharges_Delete_Recalculation(t *testing.T) {
	setupInventoryTestDB(t)

	testRouter = gin.New()
	adminGroup := testRouter.Group("/api/admin")
	adminGroup.Use(func(c *gin.Context) {
		c.Set("hotelID", hotelAID)
		c.Set("userID", "admin-1")
		c.Next()
	})
	{
		adminGroup.POST("/facility-charges", CreateFacilityCharge)
		adminGroup.DELETE("/facility-charges/:id", DeleteFacilityCharge)
	}

	// Seed confirmed reservation
	resv := models.Reservation{
		HotelID:          hotelAID,
		BookingReference: "BK-2026-99999",
		GuestName:        "Charge Guest",
		CheckIn:          time.Now(),
		CheckOut:         time.Now().AddDate(0, 0, 2),
		Nights:           2,
		RoomRate:         5000,
		TotalAmount:      11200,
		Status:           "confirmed",
	}
	database.DB.Create(&resv)

	// Seed Bill
	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: resv.ID,
		GuestName:     resv.GuestName,
		Subtotal:      10000,
		TaxRate:       0.12,
		TaxAmount:     1200,
		TotalAmount:   11200,
		Balance:       11200,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)
	item1 := models.BillItem{
		BillID:      bill.ID,
		Description: "Room charges 2 nights",
		Quantity:    2,
		UnitPrice:   5000,
		Amount:      10000,
	}
	database.DB.Create(&item1)

	// 1. Post facility charge (Spa: ₹2,000)
	fcPayload := map[string]interface{}{
		"reservationId": resv.ID,
		"category":      "spa",
		"description":   "Ayurvedic Massage",
		"amount":        2000,
	}
	fcBody, _ := json.Marshal(fcPayload)
	reqFC, _ := http.NewRequest(http.MethodPost, "/api/admin/facility-charges", bytes.NewBuffer(fcBody))
	reqFC.Header.Set("Content-Type", "application/json")
	wFC := httptest.NewRecorder()
	testRouter.ServeHTTP(wFC, reqFC)

	if wFC.Code != http.StatusCreated {
		t.Fatalf("expected 201 on charge create, got %d: %s", wFC.Code, wFC.Body.String())
	}

	var resFC map[string]interface{}
	json.Unmarshal(wFC.Body.Bytes(), &resFC)
	chargeID := resFC["data"].(map[string]interface{})["id"].(string)

	// Check bill updated (Subtotal: 12000, Tax: 1440, Total: 13440)
	database.DB.First(&bill, "id = ?", bill.ID)
	if bill.Subtotal != 12000 || bill.TotalAmount != 13440 {
		t.Errorf("expected bill total 13440 after charge, got %f", bill.TotalAmount)
	}

	// 2. Delete facility charge
	reqDel, _ := http.NewRequest(http.MethodDelete, "/api/admin/facility-charges/"+chargeID, nil)
	wDel := httptest.NewRecorder()
	testRouter.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on charge delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Verify bill recalculated back (Subtotal: 10000, Tax: 1200, Total: 11200)
	database.DB.First(&bill, "id = ?", bill.ID)
	if bill.Subtotal != 10000 || bill.TotalAmount != 11200 {
		t.Errorf("expected bill total 11200 after charge removal, got %f", bill.TotalAmount)
	}
}
