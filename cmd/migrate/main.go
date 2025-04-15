package main

import (
	"fmt"
	"log"
	"os"
	"rtcs/internal/config"
	"rtcs/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Printf("Starting migrations...")

	// Initialize configuration
	cfg := config.Get()
	log.Printf("Configuration loaded")

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Printf("Connected to database")

	// Run migrations
	log.Printf("Running migrations...")
	if err := db.AutoMigrate(
		&model.User{},
		&model.Chat{},
		&model.ChatUser{},
		&model.Message{},
	); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Printf("Migrations completed")

	// Create admin user if it doesn't exist
	admin := &model.User{
		Username: "admin",
		Password: "$2a$10$X7J3Y5Z8A9B0C1D2E3F4G5H6I7J8K9L0M1N2O3P4Q5R6S7T8U9V0W1X2Y3Z4", // "admin123"
	}
	if err := db.FirstOrCreate(admin, model.User{Username: "admin"}).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}
	log.Printf("Admin user created")

	fmt.Println("Migrations completed successfully")
	os.Exit(0)
}
