package handlers

import (
	"strconv"
	"time"

	"event-management-backend/internal/middleware"
	"event-management-backend/internal/services"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CreateEventRequest struct {
	Title       string  `json:"title" validate:"required"`
	Slug        string  `json:"slug" validate:"required,alphanum"`
	Description string  `json:"description"`
	StartsAt    string  `json:"starts_at" validate:"required"`
	EndsAt      string  `json:"ends_at" validate:"required"`
	TicketPrice float64 `json:"ticket_price" validate:"gte=0"`
	TicketQuota *int    `json:"ticket_quota" validate:"omitempty,gt=0"`
}

type AddEventDayRequest struct {
	DayNumber int    `json:"day_number" validate:"required,gt=0"`
	Label     string `json:"label" validate:"required"`
	Date      string `json:"date" validate:"required"`
}

type AddEventActionRequest struct {
	Name string `json:"name" validate:"required"`
	Code string `json:"code" validate:"required,alphanum"`
}

// CreateEvent creates a new event
// @Summary Create event
// @Tags Events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateEventRequest true "Event data"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /events [post]
func (h *Handler) CreateEvent(c *fiber.Ctx) error {
	var req CreateEventRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	// Parse dates
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return utils.Error(c, "Invalid starts_at format", fiber.StatusBadRequest)
	}

	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return utils.Error(c, "Invalid ends_at format", fiber.StatusBadRequest)
	}

	if endsAt.Before(startsAt) {
		return utils.Error(c, "End date must be after start date", fiber.StatusBadRequest)
	}

	// Handle file upload
	logoPath := ""
	file, err := c.FormFile("logo")
	if err == nil && file != nil {
		if err := utils.ValidateImageFile(file); err != nil {
			return utils.Error(c, err.Error(), fiber.StatusBadRequest)
		}

		filename := utils.GenerateUniqueFilename(file.Filename)
		if err := utils.SaveUploadedFile(file, h.cfg.LogoDir, filename); err != nil {
			return utils.Error(c, "Failed to save logo", fiber.StatusInternalServerError)
		}
		logoPath = "/logos/" + filename
	}

	// Create event
	eventReq := services.CreateEventRequest{
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
		StartsAt:    startsAt,
		EndsAt:      endsAt,
		LogoPath:    logoPath,
		TicketPrice: req.TicketPrice,
		TicketQuota: req.TicketQuota,
	}

	event, err := h.eventSvc.CreateEvent(eventReq)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, event, "Event created successfully", fiber.StatusCreated)
}

// ListEvents returns paginated list of events
// @Summary List events
// @Tags Events
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} utils.Response
// @Router /events [get]
func (h *Handler) ListEvents(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	events, total, totalPages, err := h.eventSvc.ListEvents(page, pageSize)
	if err != nil {
		return utils.Error(c, "Failed to fetch events", fiber.StatusInternalServerError)
	}

	meta := &utils.Meta{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: totalPages,
	}

	return utils.SuccessWithMeta(c, events, meta, "Events retrieved successfully")
}

// GetEvent returns event by ID
// @Summary Get event by ID
// @Tags Events
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /events/{id} [get]
func (h *Handler) GetEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}

	event, err := h.eventSvc.GetEvent(eventID)
	if err != nil {
		return utils.Error(c, "Event not found", fiber.StatusNotFound)
	}

	return utils.Success(c, event, "Event retrieved successfully")
}

// GetEventBySlug returns event by slug
// @Summary Get event by slug
// @Tags Events
// @Produce json
// @Param slug path string true "Event slug"
// @Success 200 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /events/slug/{slug} [get]
func (h *Handler) GetEventBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	event, err := h.eventSvc.GetEventBySlug(slug)
	if err != nil {
		return utils.Error(c, "Event not found", fiber.StatusNotFound)
	}

	return utils.Success(c, event, "Event retrieved successfully")
}

// AddEventDay adds a day to an event
// @Summary Add event day
// @Tags Events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param request body AddEventDayRequest true "Event day data"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /events/{id}/days [post]
func (h *Handler) AddEventDay(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}

	var req AddEventDayRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		return utils.Error(c, "Invalid date format", fiber.StatusBadRequest)
	}

	day, err := h.eventSvc.AddEventDay(eventID, req.DayNumber, req.Label, date)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, day, "Event day added successfully", fiber.StatusCreated)
}

// AddEventAction adds an action to an event day
// @Summary Add event action
// @Tags Events
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param day_id path string true "Event Day ID"
// @Param request body AddEventActionRequest true "Event action data"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /events/{id}/days/{day_id}/actions [post]
func (h *Handler) AddEventAction(c *fiber.Ctx) error {
	eventID := c.Params("id")
	dayID := c.Params("day_id")

	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}
	if _, err := uuid.Parse(dayID); err != nil {
		return utils.Error(c, "Invalid day ID", fiber.StatusBadRequest)
	}

	var req AddEventActionRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	action, err := h.eventSvc.AddEventAction(eventID, dayID, req.Name, req.Code)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, action, "Event action added successfully", fiber.StatusCreated)
}
