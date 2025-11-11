package handlers

import (
	"fmt"
	"strconv"
	"time"

	"event-management-backend/internal/middleware"
	"event-management-backend/internal/models"
	"event-management-backend/internal/services"
	"event-management-backend/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// VerificationHandler menangani semua HTTP request terkait verifikasi
type VerificationHandler struct {
	verificationService services.VerificationService
}

// NewVerificationHandler membuat instance baru VerificationHandler
func NewVerificationHandler(verificationService services.VerificationService) *VerificationHandler {
	return &VerificationHandler{
		verificationService: verificationService,
	}
}

// VerifyRequest represents the request payload for verification
type VerifyRequest struct {
	QRCodeData string `json:"qr_code_data" validate:"required"`
	ActionCode string `json:"action_code" validate:"required"`
}

// VerificationResponse represents the successful verification response
type VerificationResponse struct {
	Success         bool      `json:"success"`
	Message         string    `json:"message"`
	VerificationID  string    `json:"verification_id,omitempty"`
	ParticipantName string    `json:"participant_name,omitempty"`
	EventName       string    `json:"event_name,omitempty"`
	ActionName      string    `json:"action_name,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// VerificationHistoryResponse represents verification history response
type VerificationHistoryResponse struct {
	Verifications []VerificationDetail `json:"verifications"`
	Total         int64                `json:"total"`
	Page          int                  `json:"page"`
	PageSize      int                  `json:"page_size"`
	TotalPages    int                  `json:"total_pages"`
}

// VerificationDetail represents detailed verification information
type VerificationDetail struct {
	ID              string    `json:"id"`
	ParticipantID   string    `json:"participant_id"`
	ParticipantName string    `json:"participant_name"`
	ActionName      string    `json:"action_name"`
	ActionCode      string    `json:"action_code"`
	VerifiedBy      string    `json:"verified_by"`
	VerifiedAt      time.Time `json:"verified_at"`
	EventName       string    `json:"event_name"`
}

// VerificationStatsResponse represents verification statistics
type VerificationStatsResponse struct {
	EventID                   string    `json:"event_id"`
	EventTitle                string    `json:"event_title"`
	TotalVerifications        int64     `json:"total_verifications"`
	UniqueParticipants        int64     `json:"unique_participants"`
	TotalParticipants         int64     `json:"total_participants"`
	VerificationRate          float64   `json:"verification_rate"`
	MostVerifiedAction        string    `json:"most_verified_action"`
	TopVerifier               string    `json:"top_verifier"`
	LastVerification          time.Time `json:"last_verification"`
	TodayVerifications        int64     `json:"today_verifications"`
	AverageDailyVerifications float64   `json:"average_daily_verifications"`
}

// VerifyAction handles participant action verification
// @Summary Verify participant action
// @Description Verify a participant's action using QR code and action code
// @Tags Verification
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyRequest true "Verification request"
// @Success 200 {object} utils.Response{data=VerificationResponse}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 409 {object} utils.Response
// @Router /verify [post]
func (h *VerificationHandler) VerifyAction(c *fiber.Ctx) error {
	// Get verifier ID from JWT token
	verifierID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return utils.Error(c, "Authentication required", fiber.StatusUnauthorized)
	}

	var req VerifyRequest
	if err := middleware.ValidateBody(&req)(c); err != nil {
		return err
	}

	// Prepare verification request
	verifyReq := services.VerifyRequest{
		QRCodeData: req.QRCodeData,
		ActionCode: req.ActionCode,
		VerifierID: verifierID,
	}

	// Perform verification
	result, err := h.verificationService.VerifyParticipantAction(verifyReq)
	if err != nil {
		return h.handleVerificationError(c, err)
	}

	// Build success response
	response := VerificationResponse{
		Success:         result.Success,
		Message:         result.Message,
		VerificationID:  result.ActionLog.ID.String(),
		ParticipantName: result.Participant.Name,
		EventName:       "",
		ActionName:      result.EventAction.Name,
		Timestamp:       result.Timestamp,
	}

	return utils.Success(c, response, "Verification successful")
}

// GetParticipantVerifications retrieves verification history for a participant
// @Summary Get participant verification history
// @Description Get all verification records for a specific participant
// @Tags Verification
// @Produce json
// @Security BearerAuth
// @Param id path string true "Participant ID"
// @Success 200 {object} utils.Response{data=[]VerificationDetail}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /participants/{id}/verifications [get]
func (h *VerificationHandler) GetParticipantVerifications(c *fiber.Ctx) error {
	participantID := c.Params("id")
	if _, err := uuid.Parse(participantID); err != nil {
		return utils.Error(c, "Invalid participant ID format", fiber.StatusBadRequest)
	}

	verifications, err := h.verificationService.GetParticipantVerificationHistory(participantID)
	if err != nil {
		return h.handleVerificationError(c, err)
	}

	// Transform to response format
	response := h.transformVerificationsToDetail(verifications)

	return utils.Success(c, response, "Verification history retrieved successfully")
}

// GetEventVerifications retrieves paginated verification records for an event
// @Summary Get event verifications
// @Description Get paginated verification records for a specific event with optional filters
// @Tags Verification
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param date_from query string false "Start date (RFC3339)"
// @Param date_to query string false "End date (RFC3339)"
// @Param action_id query string false "Filter by action ID"
// @Param verifier_id query string false "Filter by verifier ID"
// @Success 200 {object} utils.Response{data=VerificationHistoryResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /events/{id}/verifications [get]
func (h *VerificationHandler) GetEventVerifications(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID format", fiber.StatusBadRequest)
	}

	// Parse query parameters
	filters, err := h.parseVerificationFilters(c)
	if err != nil {
		return utils.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	// Get verifications
	verificationList, err := h.verificationService.GetEventVerifications(eventID, filters)
	if err != nil {
		return h.handleVerificationError(c, err)
	}

	// Transform to response format
	response := h.transformToVerificationHistoryResponse(verificationList)

	return utils.Success(c, response, "Event verifications retrieved successfully")
}

// GetVerificationStats retrieves verification statistics for an event
// @Summary Get verification statistics
// @Description Get comprehensive verification statistics for a specific event
// @Tags Verification
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Success 200 {object} utils.Response{data=VerificationStatsResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /events/{id}/verifications/stats [get]
func (h *VerificationHandler) GetVerificationStats(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID format", fiber.StatusBadRequest)
	}

	stats, err := h.verificationService.GetVerificationStats(eventID)
	if err != nil {
		return h.handleVerificationError(c, err)
	}

	// Transform to response format
	response := h.transformToStatsResponse(stats)

	return utils.Success(c, response, "Verification statistics retrieved successfully")
}

// CheckVerificationEligibility checks if a participant can be verified for an action
// @Summary Check verification eligibility
// @Description Check if a participant is eligible for verification for a specific action
// @Tags Verification
// @Produce json
// @Security BearerAuth
// @Param participant_id query string true "Participant ID"
// @Param action_id query string true "Action ID"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Router /verify/eligibility [get]
func (h *VerificationHandler) CheckVerificationEligibility(c *fiber.Ctx) error {
	participantID := c.Query("participant_id")
	actionID := c.Query("action_id")

	if participantID == "" || actionID == "" {
		return utils.Error(c, "Participant ID and Action ID are required", fiber.StatusBadRequest)
	}

	if _, err := uuid.Parse(participantID); err != nil {
		return utils.Error(c, "Invalid participant ID format", fiber.StatusBadRequest)
	}

	if _, err := uuid.Parse(actionID); err != nil {
		return utils.Error(c, "Invalid action ID format", fiber.StatusBadRequest)
	}

	eligible, err := h.verificationService.CanVerifyParticipant(participantID, actionID)
	if err != nil {
		return h.handleVerificationError(c, err)
	}

	response := map[string]interface{}{
		"eligible":       eligible,
		"participant_id": participantID,
		"action_id":      actionID,
		"checked_at":     time.Now(),
	}

	message := "Participant is eligible for verification"
	if !eligible {
		message = "Participant is not eligible for verification"
	}

	return utils.Success(c, response, message)
}

// RevertVerification allows admin to revert a verification
// @Summary Revert verification
// @Description Admin endpoint to revert a verification (soft delete)
// @Tags Verification
// @Security BearerAuth
// @Param id path string true "Verification ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/verifications/{id}/revert [post]
func (h *VerificationHandler) RevertVerification(c *fiber.Ctx) error {
	// Only admin can revert verifications
	userRole := c.Locals("user_role")
	if userRole != "admin" {
		return utils.Error(c, "Admin access required", fiber.StatusForbidden)
	}

	verificationID := c.Params("id")
	adminID, err := middleware.GetUserIDFromContext(c)
	if err != nil {
		return utils.Error(c, "Authentication required", fiber.StatusUnauthorized)
	}

	if _, err := uuid.Parse(verificationID); err != nil {
		return utils.Error(c, "Invalid verification ID format", fiber.StatusBadRequest)
	}

	if err := h.verificationService.RevertVerification(verificationID, adminID); err != nil {
		return h.handleVerificationError(c, err)
	}

	return utils.Success(c, nil, "Verification reverted successfully")
}

// GetDailyVerifications retrieves daily verification counts
// @Summary Get daily verification counts
// @Description Get daily verification counts for an event for the specified number of days
// @Tags Verification
// @Produce json
// @Security BearerAuth
// @Param id path string true "Event ID"
// @Param days query int false "Number of days" default(30)
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /events/{id}/verifications/daily [get]
func (h *VerificationHandler) GetDailyVerifications(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if _, err := uuid.Parse(eventID); err != nil {
		return utils.Error(c, "Invalid event ID format", fiber.StatusBadRequest)
	}

	days, _ := strconv.Atoi(c.Query("days", "30"))
	if days <= 0 || days > 365 {
		return utils.Error(c, "Days must be between 1 and 365", fiber.StatusBadRequest)
	}

	return utils.Error(c, "Not implemented", fiber.StatusNotImplemented)
}

// Private helper methods

// handleVerificationError handles verification service errors and maps to appropriate HTTP status
func (h *VerificationHandler) handleVerificationError(c *fiber.Ctx, err error) error {
	if verr, ok := err.(*services.VerificationError); ok {
		switch verr.Code {
		case services.ErrInvalidInput, services.ErrInvalidQRCode:
			return utils.Error(c, verr.Message, fiber.StatusBadRequest)
		case services.ErrParticipantNotFound, services.ErrActionNotFound, services.ErrEventNotFound:
			return utils.Error(c, verr.Message, fiber.StatusNotFound)
		case services.ErrVerifierNotFound:
			return utils.Error(c, verr.Message, fiber.StatusUnauthorized)
		case services.ErrPaymentRequired, services.ErrAlreadyVerified, services.ErrActionInactive:
			return utils.Error(c, verr.Message, fiber.StatusConflict)
		case services.ErrEventMismatch, services.ErrEventNotStarted:
			return utils.Error(c, verr.Message, fiber.StatusForbidden)
		case services.ErrPermissionDenied:
			return utils.Error(c, verr.Message, fiber.StatusForbidden)
		case services.ErrNotImplemented:
			return utils.Error(c, verr.Message, fiber.StatusNotImplemented)
		default:
			return utils.Error(c, verr.Message, fiber.StatusInternalServerError)
		}
	}

	// Generic error
	return utils.Error(c, "Internal server error", fiber.StatusInternalServerError)
}

// parseVerificationFilters parses query parameters into verification filters
func (h *VerificationHandler) parseVerificationFilters(c *fiber.Ctx) (*services.VerificationFilters, error) {
	filters := &services.VerificationFilters{}

	// Pagination
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	filters.Page = page
	filters.PageSize = pageSize

	// Date filters
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		parsedDate, err := time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid date_from format")
		}
		filters.DateFrom = parsedDate
	}

	if dateTo := c.Query("date_to"); dateTo != "" {
		parsedDate, err := time.Parse(time.RFC3339, dateTo)
		if err != nil {
			return nil, fmt.Errorf("invalid date_to format")
		}
		filters.DateTo = parsedDate
	}

	// Other filters
	if actionID := c.Query("action_id"); actionID != "" {
		if _, err := uuid.Parse(actionID); err != nil {
			return nil, fmt.Errorf("invalid action_id format")
		}
		filters.ActionID = actionID
	}

	if verifierID := c.Query("verifier_id"); verifierID != "" {
		if _, err := uuid.Parse(verifierID); err != nil {
			return nil, fmt.Errorf("invalid verifier_id format")
		}
		filters.VerifierID = verifierID
	}

	return filters, nil
}

// transformVerificationsToDetail transforms ActionLog models to VerificationDetail responses
func (h *VerificationHandler) transformVerificationsToDetail(verifications []*models.ActionLog) []VerificationDetail {
	var details []VerificationDetail

	for _, log := range verifications {
		detail := VerificationDetail{
			ID:              log.ID.String(),
			ParticipantID:   log.ParticipantID.String(),
			ParticipantName: log.Participant.Name,
			ActionName:      log.Action.Name,
			ActionCode:      log.Action.Code,
			VerifiedBy:      log.Verifier.Email,
			VerifiedAt:      log.VerifiedAt,
			EventName:       log.Participant.Event.Title,
		}
		details = append(details, detail)
	}

	return details
}

// transformToVerificationHistoryResponse transforms service response to HTTP response
func (h *VerificationHandler) transformToVerificationHistoryResponse(list *services.VerificationList) *VerificationHistoryResponse {
	var verifications []VerificationDetail

	for _, log := range list.Verifications {
		detail := VerificationDetail{
			ID:              log.ID.String(),
			ParticipantID:   log.ParticipantID.String(),
			ParticipantName: log.Participant.Name,
			ActionName:      log.Action.Name,
			ActionCode:      log.Action.Code,
			VerifiedBy:      log.Verifier.Email,
			VerifiedAt:      log.VerifiedAt,
			EventName:       log.Participant.Event.Title,
		}
		verifications = append(verifications, detail)
	}

	return &VerificationHistoryResponse{
		Verifications: verifications,
		Total:         list.TotalCount,
		Page:          list.Page,
		PageSize:      list.PageSize,
		TotalPages:    list.TotalPages,
	}
}

// transformToStatsResponse transforms service stats to HTTP response
func (h *VerificationHandler) transformToStatsResponse(stats *services.VerificationStats) *VerificationStatsResponse {
	return &VerificationStatsResponse{
		EventID:                   stats.EventID,
		EventTitle:                stats.EventTitle,
		TotalVerifications:        stats.TotalVerifications,
		UniqueParticipants:        stats.UniqueParticipants,
		TotalParticipants:         0,
		VerificationRate:          stats.VerificationRate,
		MostVerifiedAction:        stats.MostVerifiedAction,
		TopVerifier:               stats.TopVerifier,
		LastVerification:          stats.LastVerification,
		TodayVerifications:        stats.TodayVerifications,
		AverageDailyVerifications: 0,
	}
}

// RegisterVerificationRoutes mendaftarkan semua routes verifikasi
func (h *VerificationHandler) RegisterVerificationRoutes(router fiber.Router, authMiddleware fiber.Handler) {
	// Verification routes (protected)
	verification := router.Group("/verify", authMiddleware)
	{
		verification.Post("/", h.VerifyAction)
		verification.Get("/eligibility", h.CheckVerificationEligibility)
	}

	// Participant verification history (protected)
	participants := router.Group("/participants", authMiddleware)
	{
		participants.Get("/:id/verifications", h.GetParticipantVerifications)
	}

	// Event verification routes (protected)
	events := router.Group("/events", authMiddleware)
	{
		events.Get("/:id/verifications", h.GetEventVerifications)
		events.Get("/:id/verifications/stats", h.GetVerificationStats)
		events.Get("/:id/verifications/daily", h.GetDailyVerifications)
	}

	// Admin verification routes (admin only)
	admin := router.Group("/admin/verifications", authMiddleware)
	admin.Use(func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole != "admin" {
			return utils.Error(c, "Admin access required", fiber.StatusForbidden)
		}
		return c.Next()
	})
	{
		admin.Post("/:id/revert", h.RevertVerification)
	}
}
