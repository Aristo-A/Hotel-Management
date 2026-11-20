package main

import (
	"log"
	"os"

	"hotel-backend/database"
	"hotel-backend/routes"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("==================================================")
	log.Println(" Hotel Management System — Go REST API Backend    ")
	log.Println("==================================================")

	// 1. Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Notice: No .env file found, relying on system environment variables")
	}

	// 2. Initialize Database and Auto-Migrations
	database.Connect()

	// 3. Seed Initial Demo Data
	database.SeedInitialData()

	// 4. Setup Routes and Middleware
	router := routes.SetupRoutes()

	// 5. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s (API Base: http://localhost:%s/api)\n", port, port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Fatal: Server failed to start: %v", err)
	}
}
