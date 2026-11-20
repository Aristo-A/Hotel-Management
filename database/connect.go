package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"hotel-backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global active database connection
var DB *gorm.DB

// Connect initializes database connection pool and applies migrations
func Connect() {
	var err error

	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		driver = "postgres"
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	if driver == "sqlite" {
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "hotel.db"
		}
		log.Printf("Connecting to SQLite database at: %s", dbPath)
		DB, err = gorm.Open(sqlite.Open(dbPath), gormConfig)
	} else {
		databaseURL := os.Getenv("DATABASE_URL")
		if databaseURL == "" {
			host := os.Getenv("DB_HOST")
			if host == "" {
				host = "localhost"
			}
			port := os.Getenv("DB_PORT")
			if port == "" {
				port = "5432"
			}
			user := os.Getenv("DB_USER")
			if user == "" {
				user = "hoteluser"
			}
			password := os.Getenv("DB_PASSWORD")
			if password == "" {
				password = "hotelpassword"
			}
			dbname := os.Getenv("DB_NAME")
			if dbname == "" {
				dbname = "hoteldb"
			}
			sslmode := os.Getenv("DB_SSLMODE")
			if sslmode == "" {
				sslmode = "disable"
			}
			databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
		}
		log.Printf("Connecting to PostgreSQL database at %s...", databaseURL)
		DB, err = gorm.Open(postgres.Open(databaseURL), gormConfig)
		if err != nil {
			log.Printf("Warning: Failed to connect to PostgreSQL (%v).", err)
			log.Println("Falling back to local SQLite 'hotel.db' for development...")
			DB, err = gorm.Open(sqlite.Open("hotel.db"), gormConfig)
		}
	}

	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to retrieve underlying sql.DB: %v", err)
	}

	maxOpen := 25
	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxOpen = n
		}
	}

	maxIdle := 10
	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			maxIdle = n
		}
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	log.Println("Database connection pool established successfully")

	// Execute migrations for all core entities
	err = DB.AutoMigrate(
		&models.Hotel{},
		&models.User{},
		&models.HotelAdminAssignment{},
		&models.RoomCategory{},
		&models.Room{},
		&models.Guest{},
		&models.Reservation{},
		&models.Bill{},
		&models.BillItem{},
		&models.Payment{},
		&models.FacilityCharge{},
		&models.HousekeepingLog{},
		&models.ActivityLog{},
	)
	if err != nil {
		log.Fatalf("Database auto-migration failed: %v", err)
	}

	log.Println("Database auto-migrations executed successfully (all 12 entities synchronized)")
}
