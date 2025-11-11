package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"event-management-backend/internal/config"
	"event-management-backend/internal/handlers"
	"event-management-backend/internal/repositories"
	"event-management-backend/internal/services"
	"event-management-backend/pkg/database"
	"event-management-backend/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Initialize logger
	logger.Init()

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

	// Initialize repositories
	repo := repositories.NewRepository(db)

	// Initialize services
	authSvc := services.NewAuthService(repo, cfg)
	eventSvc := services.NewEventService(repo, cfg)
	participantSvc := services.NewParticipantService(repo, cfg)
	verificationSvc := services.NewVerificationService(
		repo.ActionRepo,
		repo.EventRepo,
		repo.UserRepo,
		repo.ParticipantRepo,
		cfg,
	)

	// Initialize handlers
	handler := handlers.NewHandler(authSvc, eventSvc, participantSvc, verificationSvc, cfg)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Event Management API",
		ErrorHandler: handlers.ErrorHandler,
	})

	// Global middlewares
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// Create upload directories
	if err := os.MkdirAll(cfg.QRDir, 0755); err != nil {
		log.Fatalf("Failed to create QR directory: %v", err)
	}
	if err := os.MkdirAll(cfg.LogoDir, 0755); err != nil {
		log.Fatalf("Failed to create logo directory: %v", err)
	}

	// Static file serving
	app.Static("/qrcodes", cfg.QRDir)
	app.Static("/logos", cfg.LogoDir)

	// Register routes
	api := app.Group("/api/v1")
	handler.RegisterRoutes(api)

	// Start server
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("🚀 Server starting on %s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped gracefully")
}