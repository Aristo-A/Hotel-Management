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

func setupOperationsRouter() {
	testRouter = gin.New()
	adminGroup := testRouter.Group("/api/admin")
	adminGroup.Use(middleware.RequireAuth(), middleware.RequireRole("hotel_admin"), middleware.RequireTenant())
	{
		// Operational Dashboard
		adminGroup.GET("/dashboard", GetDashboard)

		// Reservations & Operations
		adminGroup.GET("/reservations", GetReservations)
		adminGroup.POST("/reservations", CreateReservation)
		adminGroup.GET("/reservations/today/check-ins", GetTodayCheckIns)
		adminGroup.GET("/reservations/today/check-outs", GetTodayCheckOuts)
		adminGroup.GET("/reservations/:id", GetReservationByID)
		adminGroup.PUT("/reservations/:id", UpdateReservation)
		adminGroup.PATCH("/reservations/:id/cancel", CancelReservation)
		adminGroup.PATCH("/reservations/:id/check-in", CheckIn)
		adminGroup.POST("/reservations/:id/checkin", CheckIn)
		adminGroup.PATCH("/reservations/:id/check-out", CheckOut)
		adminGroup.POST("/reservations/:id/checkout", CheckOut)

		// Billing & Invoicing
		adminGroup.GET("/bills", GetBills)
		adminGroup.POST("/bills", CreateBill)
		adminGroup.GET("/bills/:id", GetBillByID)
		adminGroup.POST("/bills/:id/items", AddLineItem)
		adminGroup.PATCH("/bills/:id/discount", ApplyDiscount)

		// Payments
		adminGroup.GET("/payments", GetPayments)
		adminGroup.POST("/payments", CreatePayment)
		adminGroup.GET("/payments/:id", GetPaymentByID)

		// Analytics & Reports
		adminGroup.GET("/reports/occupancy", GetOccupancyReport)
		adminGroup.GET("/reports/revenue", GetRevenueReport)
		adminGroup.GET("/reports/housekeeping", GetHousekeepingReport)
		adminGroup.GET("/reports/bookings", GetBookingReport)
	}
}

func TestReservation_Lifecycle_And_FrontDesk(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	now := time.Now()
	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	// 1. Staff creates reservation for Room 101
	payload := map[string]interface{}{
		"roomId":          seededRoomAID,
		"guestName":       "Vikram Malhotra",
		"guestEmail":      "vikram@example.com",
		"guestPhone":      "+91 98222 33445",
		"checkIn":         today,
		"checkOut":        tomorrow,
		"numGuests":       1,
		"specialRequests": "Early check-in requested",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/api/admin/reservations", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 on reservation create, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &res)
	resvData := res["data"].(map[string]interface{})
	resvID := resvData["id"].(string)

	// 2. Check-in
	reqIn, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-in", nil)
	reqIn.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wIn := httptest.NewRecorder()
	testRouter.ServeHTTP(wIn, reqIn)

	if wIn.Code != http.StatusOK {
		t.Fatalf("expected 200 on checkin, got %d: %s", wIn.Code, wIn.Body.String())
	}

	// Verify room is now occupied
	var room models.Room
	database.DB.First(&room, "id = ?", seededRoomAID)
	if room.Status != "occupied" {
		t.Errorf("expected room status 'occupied', got %s", room.Status)
	}

	// 3. Duplicate check-in should fail
	reqInDup, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-in", nil)
	reqInDup.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wInDup := httptest.NewRecorder()
	testRouter.ServeHTTP(wInDup, reqInDup)
	if wInDup.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate check-in, got %d", wInDup.Code)
	}

	// 4. Check-out
	reqOut, _ := http.NewRequest(http.MethodPatch, "/api/admin/reservations/"+resvID+"/check-out", nil)
	reqOut.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wOut := httptest.NewRecorder()
	testRouter.ServeHTTP(wOut, reqOut)

	if wOut.Code != http.StatusOK {
		t.Fatalf("expected 200 on checkout, got %d: %s", wOut.Code, wOut.Body.String())
	}

	// Verify room is now dirty and housekeeping log created
	database.DB.First(&room, "id = ?", seededRoomAID)
	if room.Status != "dirty" {
		t.Errorf("expected room status 'dirty' after checkout, got %s", room.Status)
	}

	var hkCount int64
	database.DB.Model(&models.HousekeepingLog{}).Where("room_id = ? AND hk_status = 'dirty'", seededRoomAID).Count(&hkCount)
	if hkCount == 0 {
		t.Errorf("expected housekeeping log on checkout")
	}
}

func TestBilling_Lifecycle_And_Payments(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	now := time.Now()
	// Create bill
	bill := models.Bill{
		HotelID:       hotelAID,
		ReservationID: "res-test-1",
		GuestName:     "Kavita Krishnan",
		GuestEmail:    "kavita@example.com",
		RoomNumber:    "101",
		CheckIn:       now,
		CheckOut:      now.AddDate(0, 0, 2),
		Nights:        2,
		Subtotal:      10000,
		TaxRate:       0.12,
		TaxAmount:     1200,
		TotalAmount:   11200,
		AmountPaid:    0,
		Balance:       11200,
		Status:        "unpaid",
	}
	database.DB.Create(&bill)

	item1 := models.BillItem{
		BillID:      bill.ID,
		Description: "Deluxe Suite – 2 nights @ ₹5,000/night",
		Quantity:    2,
		UnitPrice:   5000,
		Amount:      10000,
	}
	database.DB.Create(&item1)

	// 1. Add extra facility charge (e.g. Room Service / Restaurant ₹1,000)
	addItemPayload := map[string]interface{}{
		"description": "Room Service – Dinner",
		"quantity":    1,
		"amount":      1000,
	}
	bodyItem, _ := json.Marshal(addItemPayload)
	reqItem, _ := http.NewRequest(http.MethodPost, "/api/admin/bills/"+bill.ID+"/items", bytes.NewBuffer(bodyItem))
	reqItem.Header.Set("Content-Type", "application/json")
	reqItem.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wItem := httptest.NewRecorder()
	testRouter.ServeHTTP(wItem, reqItem)

	if wItem.Code != http.StatusOK {
		t.Fatalf("expected 200 on add line item, got %d: %s", wItem.Code, wItem.Body.String())
	}

	var resItem map[string]interface{}
	json.Unmarshal(wItem.Body.Bytes(), &resItem)
	updatedBill := resItem["data"].(map[string]interface{})

	// Subtotal = 10000 + 1000 = 11000
	if updatedBill["subtotal"].(float64) != 11000 {
		t.Errorf("expected subtotal 11000, got %v", updatedBill["subtotal"])
	}
	// Tax = 11000 * 0.12 = 1320
	if updatedBill["taxAmount"].(float64) != 1320 {
		t.Errorf("expected tax 1320, got %v", updatedBill["taxAmount"])
	}
	// Total = 11000 + 1320 = 12320
	if updatedBill["totalAmount"].(float64) != 12320 {
		t.Errorf("expected total 12320, got %v", updatedBill["totalAmount"])
	}

	// 2. Apply discount of ₹320
	discPayload := map[string]interface{}{
		"discount": 320,
	}
	bodyDisc, _ := json.Marshal(discPayload)
	reqDisc, _ := http.NewRequest(http.MethodPatch, "/api/admin/bills/"+bill.ID+"/discount", bytes.NewBuffer(bodyDisc))
	reqDisc.Header.Set("Content-Type", "application/json")
	reqDisc.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDisc := httptest.NewRecorder()
	testRouter.ServeHTTP(wDisc, reqDisc)

	var resDisc map[string]interface{}
	json.Unmarshal(wDisc.Body.Bytes(), &resDisc)
	discBill := resDisc["data"].(map[string]interface{})
	// Total = 12320 - 320 = 12000
	if discBill["totalAmount"].(float64) != 12000 {
		t.Errorf("expected total 12000 after discount, got %v", discBill["totalAmount"])
	}
	if discBill["balance"].(float64) != 12000 {
		t.Errorf("expected balance 12000 after discount, got %v", discBill["balance"])
	}

	// 3. Make Partial Payment of ₹5,000 via UPI
	payPayload := map[string]interface{}{
		"billId": bill.ID,
		"amount": 5000,
		"method": "upi",
	}
	bodyPay, _ := json.Marshal(payPayload)
	reqPay, _ := http.NewRequest(http.MethodPost, "/api/admin/payments", bytes.NewBuffer(bodyPay))
	reqPay.Header.Set("Content-Type", "application/json")
	reqPay.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPay := httptest.NewRecorder()
	testRouter.ServeHTTP(wPay, reqPay)

	if wPay.Code != http.StatusCreated {
		t.Fatalf("expected 201 on payment, got %d: %s", wPay.Code, wPay.Body.String())
	}

	var resPay map[string]interface{}
	json.Unmarshal(wPay.Body.Bytes(), &resPay)
	payBill := resPay["bill"].(map[string]interface{})
	if payBill["status"] != "partial" {
		t.Errorf("expected status 'partial', got %v", payBill["status"])
	}
	if payBill["balance"].(float64) != 7000 {
		t.Errorf("expected balance 7000, got %v", payBill["balance"])
	}

	// 4. Pay remaining ₹7,000 via credit_card -> bill status becomes 'paid'
	payPayload2 := map[string]interface{}{
		"billId": bill.ID,
		"amount": 7000,
		"method": "credit_card",
	}
	bodyPay2, _ := json.Marshal(payPayload2)
	reqPay2, _ := http.NewRequest(http.MethodPost, "/api/admin/payments", bytes.NewBuffer(bodyPay2))
	reqPay2.Header.Set("Content-Type", "application/json")
	reqPay2.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wPay2 := httptest.NewRecorder()
	testRouter.ServeHTTP(wPay2, reqPay2)

	var resPay2 map[string]interface{}
	json.Unmarshal(wPay2.Body.Bytes(), &resPay2)
	fullBill := resPay2["bill"].(map[string]interface{})
	if fullBill["status"] != "paid" {
		t.Errorf("expected status 'paid', got %v", fullBill["status"])
	}
	if fullBill["balance"].(float64) != 0 {
		t.Errorf("expected balance 0, got %v", fullBill["balance"])
	}
}

func TestReports_And_Dashboard(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	// 1. Dashboard stats
	reqDash, _ := http.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	reqDash.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wDash := httptest.NewRecorder()
	testRouter.ServeHTTP(wDash, reqDash)

	if wDash.Code != http.StatusOK {
		t.Fatalf("expected 200 on dashboard, got %d: %s", wDash.Code, wDash.Body.String())
	}

	var resDash map[string]interface{}
	json.Unmarshal(wDash.Body.Bytes(), &resDash)
	if resDash["totalRooms"].(float64) != 1 {
		t.Errorf("expected totalRooms 1, got %v", resDash["totalRooms"])
	}

	// 2. Reports
	reports := []string{"occupancy", "revenue", "housekeeping", "bookings"}
	for _, r := range reports {
		reqRep, _ := http.NewRequest(http.MethodGet, "/api/admin/reports/"+r, nil)
		reqRep.Header.Set("Authorization", "Bearer "+hotelAJWT)
		wRep := httptest.NewRecorder()
		testRouter.ServeHTTP(wRep, reqRep)

		if wRep.Code != http.StatusOK {
			t.Errorf("expected 200 on report %s, got %d", r, wRep.Code)
		}
	}
}

func TestOperations_MultiTenant_Isolation(t *testing.T) {
	setupInventoryTestDB(t)
	setupOperationsRouter()

	// Seed bill in Hotel A
	billA := models.Bill{
		HotelID:       hotelAID,
		ReservationID: "res-A",
		GuestName:     "Tenant A Guest",
		TotalAmount:   5000,
		Balance:       5000,
	}
	database.DB.Create(&billA)

	// Hotel B Admin attempts to fetch Hotel A's bill -> 404
	req, _ := http.NewRequest(http.MethodGet, "/api/admin/bills/"+billA.ID, nil)
	req.Header.Set("Authorization", "Bearer "+hotelBJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("security violation: Hotel B admin accessed Hotel A bill, got code %d", w.Code)
	}

	// Hotel B Admin attempts to add line item to Hotel A bill -> 404
	payload := map[string]interface{}{"description": "Hacked Line Item", "amount": 100}
	body, _ := json.Marshal(payload)
	reqMut, _ := http.NewRequest(http.MethodPost, "/api/admin/bills/"+billA.ID+"/items", bytes.NewBuffer(body))
	reqMut.Header.Set("Content-Type", "application/json")
	reqMut.Header.Set("Authorization", "Bearer "+hotelBJWT)
	wMut := httptest.NewRecorder()
	testRouter.ServeHTTP(wMut, reqMut)

	if wMut.Code != http.StatusNotFound {
		t.Errorf("security violation: Hotel B admin mutated Hotel A bill, got code %d", wMut.Code)
	}
}
