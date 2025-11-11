package main

import (
	"log"

	"event-management-backend/internal/config"
	"event-management-backend/internal/models"
	"event-management-backend/internal/repositories"
	"event-management-backend/internal/utils"
	"event-management-backend/pkg/database"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Load configuration
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Database connection error: %v", err)
	}

	// Run migrations
	if err := repositories.AutoMigrate(db); err != nil {
		log.Fatalf("Migration error: %v", err)
	}

	log.Println("✅ Database migrations completed successfully")

	// Create default admin user if not exists
	if err := createDefaultAdmin(db, cfg); err != nil {
		log.Fatalf("Failed to create default admin: %v", err)
	}

	log.Println("✅ Default admin user created (if not exists)")
	log.Println("🎉 Migration process completed!")
}

func createDefaultAdmin(db *gorm.DB, cfg *config.Config) error {
	adminEmail := "admin@event.com"
	adminPassword := "admin123"

	// Check if admin already exists
	var existingAdmin models.User
	if err := db.Where("email = ?", adminEmail).First(&existingAdmin).Error; err == nil {
		log.Println("ℹ️  Default admin user already exists")
		return nil
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(adminPassword)
	if err != nil {
		return err
	}

	// Create admin user
	admin := &models.User{
		Email:    adminEmail,
		Password: hashedPassword,
		Role:     "admin",
	}

	if err := db.Create(admin).Error; err != nil {
		return err
	}

	log.Printf("✅ Default admin user created:")
	log.Printf("   Email: %s", adminEmail)
	log.Printf("   Password: %s", adminPassword)
	log.Printf("   Role: %s", admin.Role)

	return nil
}
