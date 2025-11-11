package handlers

import (
	"event-management-backend/internal/config"
	"event-management-backend/internal/services"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	authSvc        *services.AuthService
	eventSvc       *services.EventService
	participantSvc *services.ParticipantService
	verifySvc      services.VerificationService
	cfg            *config.Config
}

func NewHandler(
	authSvc *services.AuthService,
	eventSvc *services.EventService,
	participantSvc *services.ParticipantService,
	verifySvc services.VerificationService,
	cfg *config.Config,
) *Handler {
	return &Handler{
		authSvc:        authSvc,
		eventSvc:       eventSvc,
		participantSvc: participantSvc,
		verifySvc:      verifySvc,
		cfg:            cfg,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Public routes
	public := router.Group("/auth")
	{
		public.Post("/login", h.Login)
		public.Post("/register", h.RegisterUser)
	}

	// Event public routes
	events := router.Group("/events")
	{
		events.Get("/", h.ListEvents)
		events.Get("/:id", h.GetEvent)
		events.Get("/slug/:slug", h.GetEventBySlug)
	}

	// Participant public registration
	router.Post("/register", h.RegisterParticipant)

	// Protected routes (JWT required)
	protected := router.Group("", h.AuthMiddleware())
	{
		// User profile
		protected.Get("/profile", h.GetProfile)

		// Event management (Admin/Organizer only)
		eventsAdmin := protected.Group("/events")
		eventsAdmin.Use(h.OrganizerOrAdminMiddleware())
		{
			eventsAdmin.Post("/", h.CreateEvent)
			eventsAdmin.Post("/:id/days", h.AddEventDay)
			eventsAdmin.Post("/:id/days/:day_id/actions", h.AddEventAction)
			eventsAdmin.Get("/:id/participants", h.ListParticipants)
			eventsAdmin.Get("/:id/verifications", h.GetEventVerifications)
		}

		// Participant management
		participants := protected.Group("/participants")
		participants.Use(h.StaffOrAboveMiddleware())
		{
			participants.Post("/import", h.ImportParticipants)
			participants.Patch("/:id/payment-status", h.UpdatePaymentStatus)
			participants.Get("/:id/verifications", h.GetParticipantVerifications)
		}

		// Verification (Staff or above)
		verification := protected.Group("/verify")
		verification.Use(h.StaffOrAboveMiddleware())
		{
			verification.Post("/", h.VerifyAction)
		}

		// Admin only routes
		admin := protected.Group("/admin")
		admin.Use(h.AdminOnlyMiddleware())
		{
			admin.Get("/stats", h.GetStats)
			admin.Post("/users", h.CreateUser)
		}
	}
}

// ErrorHandler handles global errors
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Default to internal server error
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log internal errors
	if code >= 500 {
		// You can log to your logging service here
		println("Internal Error:", err.Error())
	}

	return utils.Error(c, message, code)
}

// Auth middleware
func (h *Handler) AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		if userID == nil {
			return utils.Error(c, "Authentication required", fiber.StatusUnauthorized)
		}
		return c.Next()
	}
}

// Role-based middlewares
func (h *Handler) AdminOnlyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole != "admin" {
			return utils.Error(c, "Admin access required", fiber.StatusForbidden)
		}
		return c.Next()
	}
}

func (h *Handler) OrganizerOrAdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole != "admin" && userRole != "organizer" {
			return utils.Error(c, "Organizer or admin access required", fiber.StatusForbidden)
		}
		return c.Next()
	}
}

func (h *Handler) StaffOrAboveMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole != "admin" && userRole != "organizer" && userRole != "staff" {
			return utils.Error(c, "Staff or above access required", fiber.StatusForbidden)
		}
		return c.Next()
	}
}
