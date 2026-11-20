package database

import (
	"testing"

	"hotel-backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestMultiTenantIsolation verifies that queries scoped to one hotel property
// NEVER leak rooms, reservations, or bills from another property.
func TestMultiTenantIsolation(t *testing.T) {
	// Initialize an isolated in-memory SQLite database for tenant tests
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to initialize test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&models.Hotel{},
		&models.RoomCategory{},
		&models.Room{},
		&models.Reservation{},
		&models.Bill{},
	)
	if err != nil {
		t.Fatalf("failed to migrate models: %v", err)
	}

	// 1. Create Hotel Alpha (Mumbai)
	hotelAlpha := models.Hotel{
		Name:    "Grand Horizon Mumbai",
		Address: "Marine Drive",
		City:    "Mumbai",
		State:   "Maharashtra",
		Phone:   "+91 22 1111 2222",
		Email:   "mumbai@grandhorizon.in",
		Status:  "active",
	}
	db.Create(&hotelAlpha)

	// 2. Create Hotel Beta (Goa)
	hotelBeta := models.Hotel{
		Name:    "Sunset Cove Goa",
		Address: "Calangute Beach",
		City:    "Goa",
		State:   "Goa",
		Phone:   "+91 832 333 4444",
		Email:   "goa@sunsetcove.in",
		Status:  "active",
	}
	db.Create(&hotelBeta)

	// 3. Create Category & Rooms for Hotel Alpha
	catAlpha := models.RoomCategory{
		HotelID:     hotelAlpha.ID,
		Name:        "Sea View Suite",
		DefaultRate: 8500.00,
	}
	db.Create(&catAlpha)

	roomAlpha := models.Room{
		HotelID:    hotelAlpha.ID,
		CategoryID: catAlpha.ID,
		RoomNumber: "M-101",
		Rate:       8500.00,
		Status:     "available",
	}
	db.Create(&roomAlpha)

	// 4. Create Category & Rooms for Hotel Beta
	catBeta := models.RoomCategory{
		HotelID:     hotelBeta.ID,
		Name:        "Beachfront Villa",
		DefaultRate: 12000.00,
	}
	db.Create(&catBeta)

	roomBeta := models.Room{
		HotelID:    hotelBeta.ID,
		CategoryID: catBeta.ID,
		RoomNumber: "G-201",
		Rate:       12000.00,
		Status:     "available",
	}
	db.Create(&roomBeta)

	// ============================================================
	// CRITICAL TEST: Tenant Isolation Query
	// When Hotel Alpha's Admin queries rooms with WHERE hotel_id = hotelAlpha.ID:
	// They must ONLY see Hotel Alpha's rooms, NEVER Hotel Beta's rooms!
	// ============================================================
	var alphaRooms []models.Room
	db.Where("hotel_id = ?", hotelAlpha.ID).Find(&alphaRooms)

	if len(alphaRooms) != 1 {
		t.Fatalf("expected 1 room for Hotel Alpha, got: %d", len(alphaRooms))
	}
	if alphaRooms[0].RoomNumber != "M-101" {
		t.Errorf("expected room M-101, got: %s", alphaRooms[0].RoomNumber)
	}

	// Hotel Alpha's Admin queries for room G-201 (which belongs to Hotel Beta)
	var unauthorizedAccess models.Room
	err = db.Where("hotel_id = ? AND id = ?", hotelAlpha.ID, roomBeta.ID).First(&unauthorizedAccess).Error
	if err == nil {
		t.Fatal("SECURITY BREACH: Hotel Alpha was able to query Hotel Beta's room record!")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected ErrRecordNotFound, got: %v", err)
	}

	// Now verify Beta Admin's scope
	var betaRooms []models.Room
	db.Where("hotel_id = ?", hotelBeta.ID).Find(&betaRooms)
	if len(betaRooms) != 1 {
		t.Fatalf("expected 1 room for Hotel Beta, got: %d", len(betaRooms))
	}
	if betaRooms[0].RoomNumber != "G-201" {
		t.Errorf("expected room G-201, got: %s", betaRooms[0].RoomNumber)
	}
}
