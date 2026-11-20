package admin

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

// BillDTO formats billing folios
type BillDTO struct {
	ID               string           `json:"id"`
	HotelID          string           `json:"hotelId"`
	ReservationID    string           `json:"reservationId"`
	BookingReference string           `json:"bookingReference,omitempty"`
	GuestName        string           `json:"guestName"`
	GuestEmail       string           `json:"guestEmail"`
	GuestPhone       string           `json:"guestPhone"`
	RoomNumber       string           `json:"roomNumber"`
	StayDates        string           `json:"stayDates"`
	CheckIn          string           `json:"checkIn"`
	CheckOut         string           `json:"checkOut"`
	Nights           int              `json:"nights"`
	Subtotal         float64          `json:"subtotal"`
	TaxRate          float64          `json:"taxRate"`
	TaxAmount        float64          `json:"taxAmount"`
	Discount         float64          `json:"discount"`
	TotalAmount      float64          `json:"totalAmount"`
	AmountPaid       float64          `json:"amountPaid"`
	Balance          float64          `json:"balance"`
	Status           string           `json:"status"` // paid, unpaid, partial
	PaidAt           *time.Time       `json:"paidAt,omitempty"`
	LineItems        []models.BillItem `json:"lineItems"`
	Payments         []PaymentDTO     `json:"payments,omitempty"`
	CreatedAt        time.Time        `json:"createdAt"`
}

// PaymentDTO formats payment transactions
type PaymentDTO struct {
	ID               string     `json:"id"`
	HotelID          string     `json:"hotelId"`
	BillID           string     `json:"billId"`
	BookingReference string     `json:"bookingReference,omitempty"`
	GuestName        string     `json:"guestName,omitempty"`
	Amount           float64    `json:"amount"`
	Method           string     `json:"method"` // credit_card, cash, upi, bank_transfer
	Status           string     `json:"status"` // completed, pending, failed
	TransactionID    string     `json:"transactionId"`
	PaidAt           *time.Time `json:"paidAt"`
	CreatedAt        time.Time  `json:"createdAt"`
}

func buildBillDTO(bill *models.Bill) BillDTO {
	stayDates := fmt.Sprintf("%s – %s", bill.CheckIn.Format("02 Jan"), bill.CheckOut.Format("02 Jan 2006"))
	bookingRef := ""

	if bill.ReservationID != "" {
		var resv models.Reservation
		if err := database.DB.Where("id = ?", bill.ReservationID).First(&resv).Error; err == nil {
			bookingRef = resv.BookingReference
		}
	}

	payments := make([]PaymentDTO, 0, len(bill.Payments))
	for _, p := range bill.Payments {
		payments = append(payments, PaymentDTO{
			ID:               p.ID,
			HotelID:          p.HotelID,
			BillID:           p.BillID,
			BookingReference: bookingRef,
			GuestName:        bill.GuestName,
			Amount:           p.Amount,
			Method:           p.Method,
			Status:           p.Status,
			TransactionID:    p.TransactionID,
			PaidAt:           p.PaidAt,
			CreatedAt:        p.CreatedAt,
		})
	}

	items := bill.LineItems
	if items == nil {
		items = make([]models.BillItem, 0)
	}

	return BillDTO{
		ID:               bill.ID,
		HotelID:          bill.HotelID,
		ReservationID:    bill.ReservationID,
		BookingReference: bookingRef,
		GuestName:        bill.GuestName,
		GuestEmail:       bill.GuestEmail,
		GuestPhone:       bill.GuestPhone,
		RoomNumber:       bill.RoomNumber,
		StayDates:        stayDates,
		CheckIn:          bill.CheckIn.Format("2006-01-02"),
		CheckOut:         bill.CheckOut.Format("2006-01-02"),
		Nights:           bill.Nights,
		Subtotal:         bill.Subtotal,
		TaxRate:          bill.TaxRate,
		TaxAmount:        bill.TaxAmount,
		Discount:         bill.Discount,
		TotalAmount:      bill.TotalAmount,
		AmountPaid:       bill.AmountPaid,
		Balance:          bill.Balance,
		Status:           bill.Status,
		PaidAt:           bill.PaidAt,
		LineItems:        items,
		Payments:         payments,
		CreatedAt:        bill.CreatedAt,
	}
}

func buildPaymentDTO(p *models.Payment) PaymentDTO {
	guestName := ""
	bookingRef := ""

	var bill models.Bill
	if err := database.DB.Where("id = ?", p.BillID).First(&bill).Error; err == nil {
		guestName = bill.GuestName
		var resv models.Reservation
		if errResv := database.DB.Where("id = ?", bill.ReservationID).First(&resv).Error; errResv == nil {
			bookingRef = resv.BookingReference
		}
	}

	return PaymentDTO{
		ID:               p.ID,
		HotelID:          p.HotelID,
		BillID:           p.BillID,
		BookingReference: bookingRef,
		GuestName:        guestName,
		Amount:           p.Amount,
		Method:           p.Method,
		Status:           p.Status,
		TransactionID:    p.TransactionID,
		PaidAt:           p.PaidAt,
		CreatedAt:        p.CreatedAt,
	}
}

func generateTxnID() string {
	now := time.Now().Format("20060102")
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("TXN-%s-%04d", now, 1000+n.Int64())
}

// GetBills returns all bills for the authenticated property
// GET /api/admin/bills
func GetBills(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))

	query := database.DB.Model(&models.Bill{}).
		Preload("LineItems").
		Preload("Payments").
		Where("hotel_id = ?", hotelID)

	if status != "" && !strings.EqualFold(status, "all") {
		query = query.Where("status = ?", strings.ToLower(status))
	}

	if search != "" {
		s := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(guest_name) LIKE ? OR LOWER(guest_email) LIKE ? OR room_number LIKE ?", s, s, s)
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var bills []models.Bill
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&bills).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Failed to retrieve billing records")
		return
	}

	dtos := make([]BillDTO, 0, len(bills))
	for i := range bills {
		dtos = append(dtos, buildBillDTO(&bills[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dtos,
		"bills":   dtos,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetBillByID returns a single bill with line items and payments
// GET /api/admin/bills/:id
func GetBillByID(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var bill models.Bill
	if err := database.DB.Preload("LineItems").Preload("Payments").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill not found")
		return
	}

	dto := buildBillDTO(&bill)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"bill":    dto,
	})
}

// CreateBillRequest defines input for creating a bill
type CreateBillRequest struct {
	ReservationID string  `json:"reservationId" binding:"required"`
	GuestName     string  `json:"guestName"`
	RoomNumber    string  `json:"roomNumber"`
	Subtotal      float64 `json:"subtotal"`
}

// CreateBill creates a new bill folio
// POST /api/admin/bills
func CreateBill(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CreateBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	var resv models.Reservation
	if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", req.ReservationID, hotelID).First(&resv).Error; err != nil {
		util.SendError(c, http.StatusBadRequest, "The specified reservation does not exist for this property")
		return
	}

	subtotal := req.Subtotal
	if subtotal <= 0 {
		subtotal = math.Round(resv.RoomRate*float64(resv.Nights)*100) / 100
	}
	taxRate := 0.12
	taxAmount := math.Round(subtotal*taxRate*100) / 100
	total := math.Round((subtotal+taxAmount)*100) / 100

	roomNum := "TBA"
	if resv.Room != nil {
		roomNum = resv.Room.RoomNumber
	}

	bill := models.Bill{
		HotelID:       hotelID,
		ReservationID: resv.ID,
		GuestName:     resv.GuestName,
		GuestEmail:    resv.GuestEmail,
		GuestPhone:    resv.GuestPhone,
		RoomNumber:    roomNum,
		CheckIn:       resv.CheckIn,
		CheckOut:      resv.CheckOut,
		Nights:        resv.Nights,
		Subtotal:      subtotal,
		TaxRate:       taxRate,
		TaxAmount:     taxAmount,
		TotalAmount:   total,
		Balance:       total,
		Status:        "unpaid",
	}

	if err := database.DB.Create(&bill).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to create bill folio. Please try again.")
		return
	}

	dto := buildBillDTO(&bill)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Bill successfully generated",
		"data":    dto,
		"bill":    dto,
	})
}

// AddLineItemRequest defines payload for extra facility charges (room service, spa, minibar)
type AddLineItemRequest struct {
	Description string  `json:"description" binding:"required"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Amount      float64 `json:"amount"`
}

// AddLineItem adds an extra charge and automatically recalculates tax, total, and balance
// POST /api/admin/bills/:id/items
func AddLineItem(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var bill models.Bill
	if err := database.DB.Preload("LineItems").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill not found")
		return
	}

	if bill.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Cannot add charges to a cancelled bill folio")
		return
	}

	var req AddLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	qty := req.Quantity
	if qty < 1 {
		qty = 1
	}

	itemAmt := req.Amount
	if itemAmt <= 0 && req.UnitPrice > 0 {
		itemAmt = math.Round(req.UnitPrice*float64(qty)*100) / 100
	}
	if itemAmt <= 0 {
		util.SendError(c, http.StatusBadRequest, "Charge amount must be greater than zero")
		return
	}

	unitPrice := req.UnitPrice
	if unitPrice <= 0 {
		unitPrice = math.Round((itemAmt/float64(qty))*100) / 100
	}

	item := models.BillItem{
		BillID:      bill.ID,
		Description: strings.TrimSpace(req.Description),
		Quantity:    qty,
		UnitPrice:   unitPrice,
		Amount:      itemAmt,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to record bill line item. Please try again.")
		return
	}

	// Recalculate bill
	var sum float64
	database.DB.Model(&models.BillItem{}).Where("bill_id = ?", bill.ID).Select("COALESCE(SUM(amount), 0)").Scan(&sum)

	bill.Subtotal = sum
	bill.TaxAmount = math.Round(bill.Subtotal*bill.TaxRate*100) / 100
	bill.TotalAmount = math.Round((bill.Subtotal+bill.TaxAmount-bill.Discount)*100) / 100
	bill.Balance = math.Max(0, math.Round((bill.TotalAmount-bill.AmountPaid)*100)/100)

	if bill.Balance == 0 && bill.AmountPaid > 0 {
		bill.Status = "paid"
	} else if bill.AmountPaid > 0 {
		bill.Status = "partial"
	} else {
		bill.Status = "unpaid"
	}

	database.DB.Save(&bill)

	database.DB.Preload("LineItems").Preload("Payments").First(&bill, "id = ?", bill.ID)
	dto := buildBillDTO(&bill)

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Line item added and totals recalculated",
		"data":     dto,
		"bill":     dto,
		"lineItem": item,
	})
}

// DiscountRequest defines discount payload
type DiscountRequest struct {
	Discount float64 `json:"discount"`
}

// ApplyDiscount applies discount and updates total and balance
// PATCH /api/admin/bills/:id/discount
func ApplyDiscount(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var bill models.Bill
	if err := database.DB.Preload("LineItems").Preload("Payments").
		Where("id = ? AND hotel_id = ?", id, hotelID).
		First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill not found")
		return
	}

	var req DiscountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	if bill.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Cannot apply discount to a cancelled bill folio")
		return
	}
	if req.Discount < 0 {
		util.SendError(c, http.StatusBadRequest, "Discount amount cannot be negative")
		return
	}
	maxDiscount := math.Round((bill.Subtotal+bill.TaxAmount)*100) / 100
	if req.Discount > maxDiscount {
		util.SendError(c, http.StatusBadRequest, fmt.Sprintf("Discount cannot exceed bill total before discount (₹%.2f)", maxDiscount))
		return
	}

	bill.Discount = math.Max(0, req.Discount)
	bill.TotalAmount = math.Round((bill.Subtotal+bill.TaxAmount-bill.Discount)*100) / 100
	bill.Balance = math.Max(0, math.Round((bill.TotalAmount-bill.AmountPaid)*100)/100)

	if bill.Balance == 0 && bill.AmountPaid > 0 {
		bill.Status = "paid"
	} else if bill.AmountPaid > 0 {
		bill.Status = "partial"
	} else {
		bill.Status = "unpaid"
	}

	database.DB.Save(&bill)
	dto := buildBillDTO(&bill)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Discount applied and totals updated",
		"data":    dto,
		"bill":    dto,
	})
}

// CreatePaymentRequest defines payload for recording a payment
type CreatePaymentRequest struct {
	BillID string  `json:"billId" binding:"required"`
	Amount float64 `json:"amount" binding:"required"`
	Method string  `json:"method"` // credit_card, cash, upi, bank_transfer
}

// CreatePayment records payment against a bill and updates folio status
// POST /api/admin/payments
func CreatePayment(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	if req.Amount <= 0 {
		util.SendError(c, http.StatusBadRequest, "Payment amount must be greater than zero")
		return
	}

	var bill models.Bill
	if err := database.DB.Where("id = ? AND hotel_id = ?", req.BillID, hotelID).First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill folio not found")
		return
	}

	if bill.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Cannot record payment on a cancelled bill folio")
		return
	}
	if bill.Status == "paid" && bill.Balance == 0 {
		util.SendError(c, http.StatusBadRequest, "Bill folio is already settled and fully paid")
		return
	}

	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "credit_card"
	}

	now := time.Now()
	txnID := generateTxnID()

	payment := models.Payment{
		HotelID:       hotelID,
		BillID:        bill.ID,
		Amount:        req.Amount,
		Method:        method,
		Status:        "completed",
		TransactionID: txnID,
		PaidAt:        &now,
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to record payment transaction. Please try again.")
		return
	}

	// Update bill paid amount, balance, and status
	bill.AmountPaid = math.Round((bill.AmountPaid+req.Amount)*100) / 100
	bill.Balance = math.Max(0, math.Round((bill.TotalAmount-bill.AmountPaid)*100)/100)
	if bill.Balance == 0 {
		bill.Status = "paid"
		bill.PaidAt = &now
	} else {
		bill.Status = "partial"
	}
	database.DB.Save(&bill)

	// Activity log
	var uid *string
	if val, ok := c.Get("userID"); ok {
		if s, isStr := val.(string); isStr {
			uid = &s
		}
	}
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "payment",
		Details: fmt.Sprintf("Payment of ₹%.2f received for %s (%s)", req.Amount, bill.GuestName, txnID),
	})

	dto := buildPaymentDTO(&payment)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Payment recorded successfully",
		"data":    dto,
		"payment": dto,
		"bill":    buildBillDTO(&bill),
	})
}

// GetPayments lists payment transactions for the property
// GET /api/admin/payments
func GetPayments(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	method := strings.TrimSpace(c.Query("method"))

	query := database.DB.Model(&models.Payment{}).Where("hotel_id = ?", hotelID)
	if method != "" && !strings.EqualFold(method, "all") {
		query = query.Where("method = ?", strings.ToLower(method))
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * limit
	var payments []models.Payment
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&payments).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Failed to retrieve payments")
		return
	}

	dtos := make([]PaymentDTO, 0, len(payments))
	for i := range payments {
		dtos = append(dtos, buildPaymentDTO(&payments[i]))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"data":     dtos,
		"payments": dtos,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetPaymentByID retrieves a single payment
// GET /api/admin/payments/:id
func GetPaymentByID(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	id := c.Param("id")

	var payment models.Payment
	if err := database.DB.Where("id = ? AND hotel_id = ?", id, hotelID).First(&payment).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Payment not found")
		return
	}

	dto := buildPaymentDTO(&payment)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto,
		"payment": dto,
	})
}

// GetPrintBill returns bill folio enriched with property letterhead and contact info for printing
// GET /api/admin/bills/:id/print
// GET /api/v1/hotel/bills/:id/print
func GetPrintBill(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	billID := c.Param("id")
	var bill models.Bill
	if err := database.DB.Preload("LineItems").Preload("Payments").
		Where("(id = ? OR reservation_id = ?) AND hotel_id = ?", billID, billID, hotelID).
		First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill folio not found")
		return
	}

	dto := buildBillDTO(&bill)

	// Fetch hotel letterhead data
	var hotel models.Hotel
	hotelInfo := gin.H{
		"name":         "Grand Horizon Hotel",
		"address":      "Marine Drive, Nariman Point",
		"city":         "Mumbai",
		"state":        "Maharashtra",
		"country":      "India",
		"phone":        "+91 22 6610 8888",
		"email":        "billing@grandhorizon.in",
		"checkInTime":  "14:00",
		"checkOutTime": "11:00",
	}

	if err := database.DB.Where("id = ?", hotelID).First(&hotel).Error; err == nil {
		hotelInfo["name"] = hotel.Name
		hotelInfo["address"] = hotel.Address
		hotelInfo["city"] = hotel.City
		hotelInfo["state"] = hotel.State
		hotelInfo["country"] = hotel.Country
		hotelInfo["phone"] = hotel.Phone
		hotelInfo["email"] = hotel.Email
		if hotel.CheckInTime != "" {
			hotelInfo["checkInTime"] = hotel.CheckInTime
		}
		if hotel.CheckOutTime != "" {
			hotelInfo["checkOutTime"] = hotel.CheckOutTime
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"data": gin.H{
			"id":               dto.ID,
			"reservationId":    dto.ReservationID,
			"bookingReference": dto.BookingReference,
			"guestName":        dto.GuestName,
			"guestEmail":       dto.GuestEmail,
			"guestPhone":       dto.GuestPhone,
			"roomNumber":       dto.RoomNumber,
			"stayDates":        dto.StayDates,
			"checkIn":          dto.CheckIn,
			"checkOut":         dto.CheckOut,
			"nights":           dto.Nights,
			"subtotal":         dto.Subtotal,
			"taxRate":          dto.TaxRate,
			"taxAmount":        dto.TaxAmount,
			"discount":         dto.Discount,
			"totalAmount":      dto.TotalAmount,
			"amountPaid":       dto.AmountPaid,
			"balance":          dto.Balance,
			"status":           dto.Status,
			"lineItems":        dto.LineItems,
			"items":            dto.LineItems,
			"payments":         dto.Payments,
			"createdAt":        dto.CreatedAt,
			"hotel":            hotelInfo,
		},
		"bill":  dto,
		"hotel": hotelInfo,
	})
}

// CreateBillPaymentByBillID records a payment using the bill ID from URL parameter
// POST /api/admin/bills/:id/payments
// POST /api/v1/hotel/bills/:id/payments
func CreateBillPaymentByBillID(c *gin.Context) {
	billID := c.Param("id")

	// Decode request body
	var req struct {
		Amount          float64 `json:"amount" binding:"required,gt=0"`
		Method          string  `json:"method"`
		PaymentMethod   string  `json:"paymentMethod"`
		ReferenceNumber string  `json:"referenceNumber"`
		TransactionID   string  `json:"transactionId"`
		Notes           string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	method := req.Method
	if method == "" {
		method = req.PaymentMethod
	}
	if method == "" {
		method = "cash"
	}

	refNum := req.ReferenceNumber
	if refNum == "" {
		refNum = req.TransactionID
	}
	if refNum == "" {
		refNum = fmt.Sprintf("TXN-%d", time.Now().Unix())
	}

	// Create and process payment with RecordPayment logic
	hotelID := c.GetString("hotelID")
	var bill models.Bill
	if err := database.DB.Where("id = ? AND hotel_id = ?", billID, hotelID).First(&bill).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Bill folio not found")
		return
	}

	if bill.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Cannot record payment on a cancelled bill folio")
		return
	}
	if bill.Status == "paid" && bill.Balance == 0 {
		util.SendError(c, http.StatusBadRequest, "Bill folio is already settled and fully paid")
		return
	}

	// Build Payment record
	payment := models.Payment{
		BillID:          bill.ID,
		HotelID:         hotelID,
		GuestName:       bill.GuestName,
		Amount:          req.Amount,
		Method:          method,
		ReferenceNumber: refNum,
		TransactionID:   refNum,
		Status:          "completed",
	}

	now := time.Now()
	payment.PaidAt = &now

	if err := database.DB.Create(&payment).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to record payment transaction. Please try again.")
		return
	}

	// Update bill balance
	bill.AmountPaid = math.Round((bill.AmountPaid+req.Amount)*100) / 100
	bill.Balance = math.Round((bill.TotalAmount-bill.AmountPaid)*100) / 100
	if bill.Balance <= 0 {
		bill.Balance = 0
		bill.Status = "paid"
		bill.PaidAt = &now
	} else if bill.AmountPaid > 0 {
		bill.Status = "partial"
	}
	database.DB.Save(&bill)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Payment recorded successfully",
		"data": gin.H{
			"id":              payment.ID,
			"billId":          payment.BillID,
			"amount":          payment.Amount,
			"method":          payment.Method,
			"referenceNumber": payment.ReferenceNumber,
			"transactionId":   payment.TransactionID,
			"status":          payment.Status,
			"billStatus":      bill.Status,
			"remainingBalance": bill.Balance,
		},
		"bill": buildBillDTO(&bill),
	})
}

