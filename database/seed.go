package database

import (
	"log"

	"hotel-backend/models"

	"golang.org/x/crypto/bcrypt"
)

// SeedInitialData sets up initial demo users or hotels when developing
func SeedInitialData() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return // Database already initialized
	}

	log.Println("Seeding default demo superadmin and hotel data...")

	// 1. Create Demo Hotel
	demoHotel := models.Hotel{
		Name:         "Grand Horizon Hotel",
		Description:  "Flagship 5-star property facing the Arabian Sea",
		Address:      "Marine Drive, Nariman Point",
		City:         "Mumbai",
		State:        "Maharashtra",
		Country:      "India",
		Phone:        "+91 22-6655-4433",
		Email:        "contact@grandhorizon.com",
		Status:       "active",
		StarRating:   5,
		CheckInTime:  "14:00",
		CheckOutTime: "12:00",
	}
	if err := DB.Create(&demoHotel).Error; err != nil {
		log.Printf("Seeding hotel skipped or failed: %v", err)
		return
	}

	// 2. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SecurePassword123!"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return
	}

	// 3. Create Super Admin
	superAdmin := models.User{
		Name:         "Platform SuperAdmin",
		Email:        "superadmin@hotelgroup.com",
		PasswordHash: string(hashedPassword),
		Role:         "superadmin",
		Status:       "active",
	}
	DB.Create(&superAdmin)

	// 4. Create Hotel Admin
	hotelAdmin := models.User{
		Name:         "Arjun Sharma",
		Email:        "admin@grandhorizon.com",
		PasswordHash: string(hashedPassword),
		Role:         "hotel_admin",
		Status:       "active",
	}
	DB.Create(&hotelAdmin)

	// 5. Assign Hotel Admin to Hotel
	assignment := models.HotelAdminAssignment{
		UserID:  hotelAdmin.ID,
		HotelID: demoHotel.ID,
	}
	DB.Create(&assignment)

	log.Println("Default demo data successfully seeded (admin@grandhorizon.com / SecurePassword123!)")
}
