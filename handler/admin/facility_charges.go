package admin

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"hotel-backend/database"
	"hotel-backend/models"
	"hotel-backend/util"

	"github.com/gin-gonic/gin"
)

// FacilityChargeDTO formats facility charge items for the folio
type FacilityChargeDTO struct {
	ID               string    `json:"id"`
	ReservationID    string    `json:"reservationId"`
	BookingReference string    `json:"bookingReference"`
	GuestName        string    `json:"guestName"`
	RoomNumber       string    `json:"roomNumber"`
	Category         string    `json:"category"`
	Description      string    `json:"description"`
	Amount           float64   `json:"amount"`
	RecordedAt       time.Time `json:"recordedAt"`
	ChargedAt        time.Time `json:"chargedAt"`
	RecordedBy       string    `json:"recordedBy"`
	ChargedBy        string    `json:"chargedBy"`
	Currency         string    `json:"currency"`
}

// CreateFacilityChargeRequest payload
type CreateFacilityChargeRequest struct {
	ReservationID string  `json:"reservationId" binding:"required"`
	Category      string  `json:"category" binding:"required"`
	Description   string  `json:"description" binding:"required"`
	Amount        float64 `json:"amount" binding:"required,gt=0"`
}

// GetFacilityCharges lists running incidental charges for checked-in guests
// GET /api/admin/facility-charges
// GET /api/v1/hotel/facility-charges
func GetFacilityCharges(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	reservationID := strings.TrimSpace(c.Query("reservationId"))
	categoryFilter := strings.TrimSpace(c.Query("category"))
	search := strings.TrimSpace(c.Query("search"))

	query := database.DB.Where("hotel_id = ?", hotelID)
	if reservationID != "" {
		query = query.Where("reservation_id = ?", reservationID)
	}
	if categoryFilter != "" && categoryFilter != "all" && categoryFilter != "All" {
		query = query.Where("LOWER(category) = ?", strings.ToLower(categoryFilter))
	}
	if search != "" {
		query = query.Where("description LIKE ? OR category LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var charges []models.FacilityCharge
	if err := query.Order("charged_at desc").Find(&charges).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to retrieve facility charges. Please try again.")
		return
	}

	dtos := make([]FacilityChargeDTO, 0, len(charges))
	for _, fc := range charges {
		guestName := "Guest"
		bookingRef := ""
		roomNum := ""

		var resv models.Reservation
		if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", fc.ReservationID, hotelID).First(&resv).Error; err == nil {
			guestName = resv.GuestName
			bookingRef = resv.BookingReference
			if resv.Room != nil {
				roomNum = resv.Room.RoomNumber
			}
		}

		recordedBy := fc.ChargedBy
		if recordedBy == "" {
			recordedBy = "Front Desk Staff"
		}

		dtos = append(dtos, FacilityChargeDTO{
			ID:               fc.ID,
			ReservationID:    fc.ReservationID,
			BookingReference: bookingRef,
			GuestName:        guestName,
			RoomNumber:       roomNum,
			Category:         fc.Category,
			Description:      fc.Description,
			Amount:           fc.Amount,
			RecordedAt:       fc.ChargedAt,
			ChargedAt:        fc.ChargedAt,
			RecordedBy:       recordedBy,
			ChargedBy:        recordedBy,
			Currency:         "INR",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dtos,
		"charges": dtos,
		"total":   len(dtos),
	})
}

// CreateFacilityCharge posts a new facility charge and updates the reservation's running bill
// POST /api/admin/facility-charges
// POST /api/v1/hotel/facility-charges
func CreateFacilityCharge(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	var req CreateFacilityChargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.SendBindingError(c, err)
		return
	}

	// Verify reservation exists for this hotel
	var resv models.Reservation
	if err := database.DB.Preload("Room").Where("id = ? AND hotel_id = ?", req.ReservationID, hotelID).First(&resv).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Reservation not found in this hotel property")
		return
	}

	if resv.Status == "cancelled" {
		util.SendError(c, http.StatusBadRequest, "Cannot post facility charges to a cancelled reservation")
		return
	}
	if resv.Status == "checked_out" {
		util.SendError(c, http.StatusBadRequest, "Cannot post facility charges to an already checked-out reservation")
		return
	}

	cat := strings.ToLower(strings.TrimSpace(req.Category))
	desc := strings.TrimSpace(req.Description)
	if cat == "" || desc == "" {
		util.SendError(c, http.StatusBadRequest, "Charge category and description cannot be empty")
		return
	}

	chargedBy := "Front Desk"
	var uid *string
	if val, exists := c.Get("userID"); exists {
		if s, ok := val.(string); ok && s != "" {
			uid = &s
			var user models.User
			if err := database.DB.Where("id = ?", s).First(&user).Error; err == nil {
				chargedBy = user.Name
			}
		}
	}

	charge := models.FacilityCharge{
		HotelID:         hotelID,
		ReservationID:   resv.ID,
		ChargedByUserID: uid,
		ChargedBy:       chargedBy,
		Category:        strings.ToLower(strings.TrimSpace(req.Category)),
		Description:     strings.TrimSpace(req.Description),
		Amount:          math.Round(req.Amount*100) / 100,
		ChargedAt:       time.Now(),
	}

	if err := database.DB.Create(&charge).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to record facility charge. Please try again.")
		return
	}

	// If reservation has a bill in 'bills', add a line item and recalculate
	var bill models.Bill
	if err := database.DB.Where("reservation_id = ? AND hotel_id = ?", resv.ID, hotelID).First(&bill).Error; err == nil {
		itemDesc := fmt.Sprintf("%s (%s)", charge.Description, charge.Category)
		lineItem := models.BillItem{
			BillID:      bill.ID,
			Description: itemDesc,
			Quantity:    1,
			UnitPrice:   charge.Amount,
			Amount:      charge.Amount,
		}
		database.DB.Create(&lineItem)

		// Recalculate bill financial totals with 12% GST
		bill.Subtotal = math.Round((bill.Subtotal+charge.Amount)*100) / 100
		tax := math.Round(bill.Subtotal*bill.TaxRate*100) / 100
		bill.TaxAmount = tax
		bill.TotalAmount = math.Round((bill.Subtotal+tax-bill.Discount)*100) / 100
		bill.Balance = math.Round((bill.TotalAmount-bill.AmountPaid)*100) / 100
		database.DB.Save(&bill)
	}

	// Activity log
	database.DB.Create(&models.ActivityLog{
		HotelID: &hotelID,
		UserID:  uid,
		Action:  "FACILITY_CHARGE",
		Details: fmt.Sprintf("Recorded charge of ₹%.2f for %s (Ref: %s)", charge.Amount, charge.Category, resv.BookingReference),
	})

	roomNum := ""
	if resv.Room != nil {
		roomNum = resv.Room.RoomNumber
	}

	dto := FacilityChargeDTO{
		ID:               charge.ID,
		ReservationID:    resv.ID,
		BookingReference: resv.BookingReference,
		GuestName:        resv.GuestName,
		RoomNumber:       roomNum,
		Category:         charge.Category,
		Description:      charge.Description,
		Amount:           charge.Amount,
		RecordedAt:       charge.ChargedAt,
		ChargedAt:        charge.ChargedAt,
		RecordedBy:       chargedBy,
		ChargedBy:        chargedBy,
		Currency:         "INR",
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Facility charge added successfully",
		"data":    dto,
		"charge":  dto,
	})
}

// DeleteFacilityCharge deletes an erroneous incidental charge
// DELETE /api/admin/facility-charges/:id
// DELETE /api/v1/hotel/facility-charges/:id
func DeleteFacilityCharge(c *gin.Context) {
	hotelID := c.GetString("hotelID")
	if hotelID == "" {
		util.SendError(c, http.StatusUnauthorized, "Missing property tenant context")
		return
	}

	chargeID := c.Param("id")
	var charge models.FacilityCharge
	if err := database.DB.Where("id = ? AND hotel_id = ?", chargeID, hotelID).First(&charge).Error; err != nil {
		util.SendError(c, http.StatusNotFound, "Facility charge not found")
		return
	}

	// Delete from database
	if err := database.DB.Delete(&charge).Error; err != nil {
		util.SendError(c, http.StatusInternalServerError, "Unable to remove facility charge. Please try again.")
		return
	}

	// Remove matching line item from folio and recalculate totals
	var bill models.Bill
	if err := database.DB.Where("reservation_id = ? AND hotel_id = ?", charge.ReservationID, hotelID).First(&bill).Error; err == nil {
		itemDesc := fmt.Sprintf("%s (%s)", charge.Description, charge.Category)
		database.DB.Where("bill_id = ? AND description = ? AND amount = ?", bill.ID, itemDesc, charge.Amount).Delete(&models.BillItem{})

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
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Facility charge removed successfully",
	})
}
