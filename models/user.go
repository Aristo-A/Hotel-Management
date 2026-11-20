package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents platform superadmins and property-level hotel admins
type User struct {
	BaseModel
	Name         string     `gorm:"size:150;not null" json:"name"`
	Email        string     `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Role         string     `gorm:"size:30;not null" json:"role"` // superadmin, hotel_admin
	Phone        string     `gorm:"size:50" json:"phone"`
	Status       string     `gorm:"size:20;not null;default:'active'" json:"status"` // active, inactive, suspended
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`

	HotelID   *string `gorm:"-" json:"hotelId,omitempty"`
	HotelName *string `gorm:"-" json:"hotelName,omitempty"`

	// Relation to tenant assignment
	Assignments []HotelAdminAssignment `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"assignments,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// HotelAdminAssignment links hotel_admin users to specific hotel properties
type HotelAdminAssignment struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string    `gorm:"type:uuid;not null;index;uniqueIndex:uq_user_hotel" json:"userId"`
	HotelID    string    `gorm:"type:uuid;not null;index;uniqueIndex:uq_user_hotel" json:"hotelId"`
	AssignedAt time.Time `gorm:"autoCreateTime" json:"assignedAt"`
	AssignedBy *string   `gorm:"type:uuid" json:"assignedBy,omitempty"`

	// Relationships
	User  User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Hotel Hotel `gorm:"foreignKey:HotelID" json:"hotel,omitempty"`
}

func (b *HotelAdminAssignment) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

func (HotelAdminAssignment) TableName() string {
	return "hotel_admin_assignments"
}
