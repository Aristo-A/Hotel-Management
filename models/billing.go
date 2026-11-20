package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Bill represents the customer invoice/folio
type Bill struct {
	BaseModel
	HotelID       string     `gorm:"type:uuid;not null;index" json:"hotelId"`
	ReservationID string     `gorm:"type:uuid;not null;index" json:"reservationId"`
	GuestName     string     `gorm:"size:150;not null" json:"guestName"`
	GuestEmail    string     `gorm:"size:150" json:"guestEmail"`
	GuestPhone    string     `gorm:"size:50" json:"guestPhone"`
	RoomNumber    string     `gorm:"size:20" json:"roomNumber"`
	CheckIn       time.Time  `gorm:"type:date;not null" json:"checkIn"`
	CheckOut      time.Time  `gorm:"type:date;not null" json:"checkOut"`
	Nights        int        `gorm:"default:1;not null" json:"nights"`
	Subtotal      float64    `gorm:"type:numeric(10,2);not null" json:"subtotal"`
	TaxRate       float64    `gorm:"type:numeric(5,2);not null;default:0.12" json:"taxRate"` // 12% GST
	TaxAmount     float64    `gorm:"type:numeric(10,2);not null;default:0.00" json:"taxAmount"`
	Discount      float64    `gorm:"type:numeric(10,2);not null;default:0.00" json:"discount"`
	TotalAmount   float64    `gorm:"type:numeric(10,2);not null" json:"totalAmount"`
	AmountPaid    float64    `gorm:"type:numeric(10,2);not null;default:0.00" json:"amountPaid"`
	Balance       float64    `gorm:"type:numeric(10,2);not null" json:"balance"`
	Status        string     `gorm:"size:20;not null;default:'unpaid';index" json:"status"` // paid, unpaid, partial
	PaidAt        *time.Time `json:"paidAt,omitempty"`

	// Relationships
	LineItems []BillItem `gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE" json:"lineItems,omitempty"`
	Payments  []Payment  `gorm:"foreignKey:BillID;constraint:OnDelete:CASCADE" json:"payments,omitempty"`
}

func (Bill) TableName() string {
	return "bills"
}

// BillItem represents a single charge line on an invoice
type BillItem struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	BillID      string    `gorm:"type:uuid;not null;index" json:"billId"`
	Description string    `gorm:"size:255;not null" json:"description"`
	Quantity    int       `gorm:"default:1;not null" json:"quantity"`
	UnitPrice   float64   `gorm:"type:numeric(10,2);not null" json:"unitPrice"`
	Amount      float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (b *BillItem) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

func (BillItem) TableName() string {
	return "bill_items"
}

// Payment records a financial transaction applied towards a bill
type Payment struct {
	ID               string     `gorm:"type:uuid;primaryKey" json:"id"`
	BillID           string     `gorm:"type:uuid;not null;index" json:"billId"`
	HotelID          string     `gorm:"type:uuid;not null;index" json:"hotelId"`
	BookingReference string     `gorm:"size:50" json:"bookingReference,omitempty"`
	GuestName        string     `gorm:"size:150" json:"guestName,omitempty"`
	Amount           float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	Method           string     `gorm:"size:50;not null" json:"method"` // credit_card, cash, upi, bank_transfer, online
	ReferenceNumber  string     `gorm:"size:100" json:"referenceNumber"`
	TransactionID    string     `gorm:"size:100" json:"transactionId,omitempty"`
	PaymentDate      time.Time  `gorm:"autoCreateTime;index" json:"paymentDate"`
	PaidAt           *time.Time `json:"paidAt,omitempty"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	Status           string     `gorm:"size:20;not null;default:'completed'" json:"status"` // completed, pending, failed
}

func (b *Payment) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

func (Payment) TableName() string {
	return "payments"
}

// FacilityCharge records staff-recorded incidental charges (restaurant, minibar, spa, etc.)
type FacilityCharge struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	HotelID         string    `gorm:"type:uuid;not null;index" json:"hotelId"`
	ReservationID   string    `gorm:"type:uuid;not null;index" json:"reservationId"`
	ChargedByUserID *string   `gorm:"type:uuid" json:"chargedByUserId,omitempty"`
	ChargedBy       string    `gorm:"size:150" json:"chargedBy,omitempty"`
	Category        string    `gorm:"size:50;not null;index" json:"category"` // restaurant, minibar, laundry, spa, room_service, other
	Description     string    `gorm:"size:255;not null" json:"description"`
	Amount          float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	ChargedAt       time.Time `gorm:"autoCreateTime" json:"chargedAt"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (f *FacilityCharge) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

func (FacilityCharge) TableName() string {
	return "facility_charges"
}
