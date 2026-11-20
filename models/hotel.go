package models

// Hotel represents a hotel property entity
type Hotel struct {
	BaseModel
	Name         string `gorm:"size:255;not null" json:"name"`
	Description  string `gorm:"type:text" json:"description"`
	Address      string `gorm:"size:255;not null" json:"address"`
	City         string `gorm:"size:100;not null" json:"city"`
	State        string `gorm:"size:100;not null" json:"state"`
	Country      string `gorm:"size:100;not null;default:'India'" json:"country"`
	Phone        string `gorm:"size:50;not null" json:"phone"`
	Email        string `gorm:"size:150;not null" json:"email"`
	Website      string  `gorm:"size:255" json:"website"`
	ImageURL     string  `gorm:"type:text" json:"imageUrl"`
	Image        string  `gorm:"-" json:"image,omitempty"`
	Status       string  `gorm:"size:20;not null;default:'active';index" json:"status"` // active, inactive
	StarRating   int     `gorm:"default:4" json:"starRating"`
	Rating       float64 `gorm:"-" json:"rating,omitempty"`
	CheckInTime  string  `gorm:"size:20;default:'14:00'" json:"checkInTime"`
	CheckOutTime string  `gorm:"size:20;default:'12:00'" json:"checkOutTime"`

	// Relationships
	Categories []RoomCategory `gorm:"foreignKey:HotelID;constraint:OnDelete:CASCADE" json:"categories,omitempty"`
	Rooms      []Room         `gorm:"foreignKey:HotelID;constraint:OnDelete:CASCADE" json:"rooms,omitempty"`
}

func (Hotel) TableName() string {
	return "hotels"
}
