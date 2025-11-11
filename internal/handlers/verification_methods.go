package handlers

import (
	"strconv"

	"event-management-backend/internal/middleware"
	"event-management-backend/internal/services"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type VerifyActionRequest struct {
	QRCode     string `json:"qr_code" validate:"required"`
	ActionCode string `json:"action_code" validate:"required"`
}

func (h *Handler) VerifyAction(c *fiber.Ctx) error {
	verifierID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusUnauthorized)
	}

	var req VerifyActionRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	verifyReq := services.VerifyRequest{
		QRCodeData: req.QRCode,
		ActionCode: req.ActionCode,
		VerifierID: verifierID,
	}

	result, err := h.verifySvc.VerifyParticipantAction(verifyReq)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	return utils.Success(c, result, "Action verified successfully")
}

func (h *Handler) GetParticipantVerifications(c *fiber.Ctx) error {
	participantID := c.Params("id")
	if _, err := uuid.Parse(participantID); err != nil {
		return utils.Error(c, "Invalid participant ID", fiber.StatusBadRequest)
	}

	verifications, err := h.verifySvc.GetParticipantVerificationHistory(participantID)
	if err != nil {
		return utils.Error(c, "Failed to fetch verifications", fiber.StatusInternalServerError)
	}

	return utils.Success(c, verifications, "Verifications retrieved successfully")
}

func (h *Handler) GetEventVerifications(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID", fiber.StatusBadRequest)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))

	filters := &services.VerificationFilters{
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.verifySvc.GetEventVerifications(eventID, filters)
	if err != nil {
		return utils.Error(c, "Failed to fetch verification logs", fiber.StatusInternalServerError)
	}

	meta := &utils.Meta{
		Page:      result.Page,
		PageSize:  result.PageSize,
		Total:     result.TotalCount,
		TotalPage: result.TotalPages,
	}

	return utils.SuccessWithMeta(c, result.Verifications, meta, "Verification logs retrieved successfully")
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	stats := fiber.Map{
		"total_events":        0,
		"total_participants":  0,
		"total_verifications": 0,
		"active_events":       0,
	}

	return utils.Success(c, stats, "Statistics retrieved successfully")
}
