package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HousekeepingLog tracks room cleaning assignments and operational statuses
type HousekeepingLog struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	RoomID     string    `gorm:"type:uuid;not null;index" json:"roomId"`
	HotelID    string    `gorm:"type:uuid;not null;index" json:"hotelId"`
	HkStatus   string    `gorm:"size:30;not null" json:"hkStatus"` // clean, dirty, cleaning, maintenance
	AssignedTo string    `gorm:"size:150" json:"assignedTo"`
	Notes      string    `gorm:"type:text" json:"notes"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`

	// Relationships
	Room Room `gorm:"foreignKey:RoomID" json:"room,omitempty"`
}

func (b *HousekeepingLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

func (HousekeepingLog) TableName() string {
	return "housekeeping_logs"
}

// ActivityLog records user or system actions for auditing
type ActivityLog struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	HotelID   *string   `gorm:"type:uuid;index" json:"hotelId,omitempty"`
	UserID    *string   `gorm:"type:uuid" json:"userId,omitempty"`
	Action    string    `gorm:"size:100;not null" json:"action"`
	Details   string    `gorm:"type:text" json:"details"` // JSON string representation
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (b *ActivityLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	return nil
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}
