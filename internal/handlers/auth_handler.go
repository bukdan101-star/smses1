package handlers

import (
	"event-management-backend/internal/middleware"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=admin organizer staff"`
}

// Login handles user authentication
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /auth/login [post]
func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	loginResp, err := h.authSvc.Authenticate(req.Email, req.Password)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusUnauthorized)
	}

	return utils.Success(c, loginResp, "Login successful")
}

// RegisterUser handles user registration (Admin only)
// @Summary Register new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterUserRequest true "User registration data"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Router /admin/users [post]
func (h *Handler) CreateUser(c *fiber.Ctx) error {
	var req RegisterUserRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	user, err := h.authSvc.CreateUser(req.Email, req.Password, req.Role)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, user, "User created successfully", fiber.StatusCreated)
}

// RegisterUser public registration (for staff/organizer signup if needed)
func (h *Handler) RegisterUser(c *fiber.Ctx) error {
	var req RegisterUserRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	// Only allow staff registration publicly, admin/organizer must be created by admin
	if req.Role != "staff" {
		return utils.Error(c, "Only staff role can be registered publicly", fiber.StatusForbidden)
	}

	user, err := h.authSvc.CreateUser(req.Email, req.Password, req.Role)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, user, "User registered successfully", fiber.StatusCreated)
}

// GetProfile returns current user profile
// @Summary Get user profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Router /profile [get]
func (h *Handler) GetProfile(c *fiber.Ctx) error {
	userID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusUnauthorized)
	}

	user, err := h.authSvc.GetUserProfile(userID)
	if err != nil {
		return utils.Error(c, "User not found", fiber.StatusNotFound)
	}

	return utils.Success(c, user, "Profile retrieved successfully")
}
