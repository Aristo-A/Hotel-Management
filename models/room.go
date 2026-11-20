package models

// RoomCategory represents a category/tier of room (Deluxe, Suite, Standard, etc.)
type RoomCategory struct {
	BaseModel
	HotelID      string   `gorm:"type:uuid;not null;index;uniqueIndex:uq_hotel_cat_name" json:"hotelId"`
	Name         string   `gorm:"size:100;not null;uniqueIndex:uq_hotel_cat_name" json:"name"`
	Description  string   `gorm:"type:text" json:"description"`
	DefaultRate  float64  `gorm:"type:numeric(10,2);not null" json:"defaultRate"`
	BaseRate     float64  `gorm:"-" json:"baseRate,omitempty"`
	MaxOccupancy int      `gorm:"default:2;not null" json:"maxOccupancy"`
	BedType      string   `gorm:"size:50;default:'Queen'" json:"bedType"`
	RoomSize     int      `json:"roomSize"` // in sq meters
	Size         int      `gorm:"-" json:"size,omitempty"`
	Amenities    string   `gorm:"type:text" json:"amenities"` // JSON string representation
	Images       string   `gorm:"type:text" json:"images"`    // JSON string representation
	Image        string   `gorm:"-" json:"image,omitempty"`
	TotalRooms   int64    `gorm:"-" json:"totalRooms,omitempty"`
	AmenitiesList []string `gorm:"-" json:"amenitiesList,omitempty"`

	// Relationships
	Rooms []Room `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT" json:"rooms,omitempty"`
}

func (RoomCategory) TableName() string {
	return "room_categories"
}

// Room represents a physical room in a hotel
type Room struct {
	BaseModel
	HotelID      string        `gorm:"type:uuid;not null;index;uniqueIndex:uq_hotel_room_num" json:"hotelId"`
	CategoryID   string        `gorm:"type:uuid;not null;index" json:"categoryId"`
	RoomNumber   string        `gorm:"size:20;not null;uniqueIndex:uq_hotel_room_num" json:"roomNumber"`
	Floor        int           `gorm:"default:1;not null" json:"floor"`
	Rate         float64       `gorm:"type:numeric(10,2);not null" json:"rate"`
	Status       string        `gorm:"size:30;not null;default:'available';index" json:"status"` // available, occupied, reserved, dirty, cleaning, out_of_service
	Description  string        `gorm:"type:text" json:"description"`
	Notes        string        `gorm:"type:text" json:"notes"`
	CategoryName string        `gorm:"-" json:"categoryName,omitempty"`
	CurrentGuest *string       `gorm:"-" json:"currentGuest,omitempty"`

	// Relationships
	Category *RoomCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

func (Room) TableName() string {
	return "rooms"
}
