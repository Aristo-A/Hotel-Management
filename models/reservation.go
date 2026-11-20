package models

import (
	"time"
)

// Guest represents an optional saved profile for guests
type Guest struct {
	BaseModel
	Name        string `gorm:"size:150;not null" json:"name"`
	Email       string `gorm:"size:150;not null;index" json:"email"`
	Phone       string `gorm:"size:50;not null" json:"phone"`
	Nationality string `gorm:"size:100" json:"nationality"`
	IDType      string `gorm:"size:30;default:'Passport'" json:"idType"`
	IDNumber    string `gorm:"size:100" json:"idNumber"`
	Address     string `gorm:"type:text" json:"address"`
}

func (Guest) TableName() string {
	return "guests"
}

// Reservation represents a booking created either publicly or by the front desk
type Reservation struct {
	BaseModel
	BookingReference string    `gorm:"size:50;uniqueIndex;not null" json:"bookingReference"`
	HotelID          string    `gorm:"type:uuid;not null;index" json:"hotelId"`
	RoomID           *string   `gorm:"type:uuid;index" json:"roomId"`
	GuestID          *string   `gorm:"type:uuid" json:"guestId,omitempty"`
	GuestName        string    `gorm:"size:150;not null" json:"guestName"`
	GuestEmail       string    `gorm:"size:150;not null" json:"guestEmail"`
	GuestPhone       string    `gorm:"size:50;not null" json:"guestPhone"`
	CheckIn          time.Time `gorm:"type:date;not null;index" json:"checkIn"`
	CheckOut         time.Time `gorm:"type:date;not null;index" json:"checkOut"`
	Nights           int       `gorm:"not null" json:"nights"`
	NumGuests        int       `gorm:"default:1;not null" json:"numGuests"`
	RoomRate         float64   `gorm:"type:numeric(10,2);not null" json:"roomRate"`
	TotalAmount      float64   `gorm:"type:numeric(10,2);not null" json:"totalAmount"`
	Status           string    `gorm:"size:30;not null;default:'confirmed';index" json:"status"` // pending, confirmed, checked_in, checked_out, cancelled
	SpecialRequests  string    `gorm:"type:text" json:"specialRequests"`

	Subtotal     float64 `gorm:"-" json:"subtotal,omitempty"`
	TaxRate      float64 `gorm:"-" json:"taxRate,omitempty"`
	TaxAmount    float64 `gorm:"-" json:"taxAmount,omitempty"`
	GrandTotal   float64 `gorm:"-" json:"grandTotal,omitempty"`
	RoomNumber   string  `gorm:"-" json:"roomNumber,omitempty"`
	RoomCategory string  `gorm:"-" json:"roomCategory,omitempty"`
	CategoryName string  `gorm:"-" json:"categoryName,omitempty"`
	HotelName    string  `gorm:"-" json:"hotelName,omitempty"`

	// Relationships
	Hotel Hotel  `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
	Room  *Room  `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	Guest *Guest `gorm:"foreignKey:GuestID" json:"guest,omitempty"`
}

func (Reservation) TableName() string {
	return "reservations"
}
