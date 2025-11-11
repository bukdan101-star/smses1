package handlers

import (
	"encoding/csv"
	"strconv"

	"event-management-backend/internal/middleware"
	"event-management-backend/internal/services"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type RegisterParticipantRequest struct {
	EventID  string `json:"event_id" validate:"required,uuid"`
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"required"`
	Division string `json:"division"`
	Address  string `json:"address"`
}

type UpdatePaymentStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=unpaid pending paid"`
}

// RegisterParticipant handles participant registration
// @Summary Register participant
// @Tags Participants
// @Accept json
// @Produce json
// @Param request body RegisterParticipantRequest true "Participant data"
// @Success 201 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /register [post]
func (h *Handler) RegisterParticipant(c *fiber.Ctx) error {
	var req RegisterParticipantRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	participantReq := services.RegisterParticipantRequest{
		EventID:  req.EventID,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Division: req.Division,
		Address:  req.Address,
	}

	result, err := h.participantSvc.RegisterParticipant(participantReq)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, result, "Participant registered successfully", fiber.StatusCreated)
}

// ListParticipants returns paginated list of participants for an event
// @Summary List participants
// @Tags Participants
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} utils.Response
// @Router /events/{id}/participants [get]
func (h *Handler) ListParticipants(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	participants, total, totalPages, err := h.participantSvc.ListParticipants(eventID, page, pageSize)
	if err != nil {
		return utils.Error(c, "Failed to fetch participants", fiber.StatusInternalServerError)
	}

	meta := &utils.Meta{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: totalPages,
	}

	return utils.SuccessWithMeta(c, participants, meta, "Participants retrieved successfully")
}

// ImportParticipants imports participants from CSV
// @Summary Import participants
// @Tags Participants
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param event_id formData string true "Event ID"
// @Param file formData file true "CSV file"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /participants/import [post]
func (h *Handler) ImportParticipants(c *fiber.Ctx) error {
	eventID := c.FormValue("event_id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return utils.Error(c, "File is required", fiber.StatusBadRequest)
	}

	// Validate file size
	if file.Size > h.cfg.MaxUploadSize {
		return utils.Error(c, "File too large", fiber.StatusBadRequest)
	}

	// Validate file type
	if file.Header.Get("Content-Type") != "text/csv" {
		return utils.Error(c, "Only CSV files are allowed", fiber.StatusBadRequest)
	}

	// Read CSV file
	src, err := file.Open()
	if err != nil {
		return utils.Error(c, "Failed to read file", fiber.StatusInternalServerError)
	}
	defer src.Close()

	reader := csv.NewReader(src)
	rows, err := reader.ReadAll()
	if err != nil {
		return utils.Error(c, "Invalid CSV format", fiber.StatusBadRequest)
	}

	if len(rows) < 2 {
		return utils.Error(c, "CSV file is empty or missing header", fiber.StatusBadRequest)
	}

	// Skip header row
	success, fail, errors, err := h.participantSvc.ImportParticipantsCSV(eventID, rows[1:])
	if err != nil {
		return utils.Error(c, "Failed to import participants", fiber.StatusInternalServerError)
	}

	result := fiber.Map{
		"success": success,
		"failed":  fail,
		"errors":  errors,
	}

	return utils.Success(c, result, "Import completed")
}

// UpdatePaymentStatus updates participant payment status
// @Summary Update payment status
// @Tags Participants
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Participant ID"
// @Param request body UpdatePaymentStatusRequest true "Payment status"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /participants/{id}/payment-status [patch]
func (h *Handler) UpdatePaymentStatus(c *fiber.Ctx) error {
	participantID := c.Params("id")
	if _, err := uuid.Parse(participantID); err != nil {
		return utils.Error(c, "Invalid participant ID", fiber.StatusBadRequest)
	}

	var req UpdatePaymentStatusRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	if err := h.participantSvc.UpdatePaymentStatus(participantID, req.Status); err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, nil, "Payment status updated successfully")
}
