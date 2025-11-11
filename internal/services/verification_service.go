package services

import (
	"errors"
	"fmt"
	"time"

	"event-management-backend/internal/config"
	"event-management-backend/internal/models"
	"event-management-backend/internal/repositories"
	"event-management-backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VerificationService handles all business logic related to participant verification
type VerificationService interface {
	VerifyParticipantAction(req VerifyRequest) (*VerificationResult, error)
	GetParticipantVerificationHistory(participantID string) ([]*models.ActionLog, error)
	GetEventVerifications(eventID string, filters *VerificationFilters) (*VerificationList, error)
	GetVerificationStats(eventID string) (*VerificationStats, error)
	CanVerifyParticipant(participantID, actionID string) (bool, error)
	RevertVerification(verificationID, adminID string) error
}

type VerifyRequest struct {
	QRCodeData string `json:"qr_code_data" validate:"required"`
	ActionCode string `json:"action_code" validate:"required"`
	VerifierID string `json:"-"`
}

type VerificationResult struct {
	Success     bool                `json:"success"`
	Message     string              `json:"message"`
	ActionLog   *models.ActionLog   `json:"action_log,omitempty"`
	Participant *models.Participant `json:"participant,omitempty"`
	EventAction *models.EventAction `json:"event_action,omitempty"`
	Timestamp   time.Time           `json:"timestamp"`
}

type VerificationFilters struct {
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	DateFrom   time.Time `json:"date_from"`
	DateTo     time.Time `json:"date_to"`
	ActionID   string    `json:"action_id"`
	VerifierID string    `json:"verifier_id"`
}

type VerificationList struct {
	Verifications []*models.ActionLog `json:"verifications"`
	TotalCount    int64               `json:"total_count"`
	Page          int                 `json:"page"`
	PageSize      int                 `json:"page_size"`
	TotalPages    int                 `json:"total_pages"`
}

type VerificationStats struct {
	EventID            string    `json:"event_id"`
	EventTitle         string    `json:"event_title"`
	TotalVerifications int64     `json:"total_verifications"`
	UniqueParticipants int64     `json:"unique_participants"`
	VerificationRate   float64   `json:"verification_rate"`
	MostVerifiedAction string    `json:"most_verified_action"`
	TopVerifier        string    `json:"top_verifier"`
	LastVerification   time.Time `json:"last_verification"`
	TodayVerifications int64     `json:"today_verifications"`
}

type verificationService struct {
	actionRepo      repositories.ActionRepository
	eventRepo       repositories.EventRepository
	userRepo        repositories.UserRepository
	participantRepo repositories.ParticipantRepository
	cfg             *config.Config
}

// NewVerificationService creates a new instance of VerificationService
func NewVerificationService(
	actionRepo repositories.ActionRepository,
	eventRepo repositories.EventRepository,
	userRepo repositories.UserRepository,
	participantRepo repositories.ParticipantRepository,
	cfg *config.Config,
) VerificationService {
	return &verificationService{
		actionRepo:      actionRepo,
		eventRepo:       eventRepo,
		userRepo:        userRepo,
		participantRepo: participantRepo,
		cfg:             cfg,
	}
}

// VerifyParticipantAction verifies a participant's action using QR code and action code
func (s *verificationService) VerifyParticipantAction(req VerifyRequest) (*VerificationResult, error) {
	// Step 1: Validate basic input
	if err := s.validateVerifyRequest(req); err != nil {
		return nil, err
	}

	// Step 2: Extract and validate participant from QR code
	participant, err := s.extractParticipantFromQR(req.QRCodeData)
	if err != nil {
		return nil, err
	}

	// Step 3: Get and validate the action
	action, err := s.getAndValidateAction(req.ActionCode)
	if err != nil {
		return nil, err
	}

	// Step 4: Get verifier information
	verifier, err := s.userRepo.GetUserByID(req.VerifierID)
	if err != nil {
		return nil, NewVerificationError("verifier not found", ErrVerifierNotFound, err)
	}

	// Step 5: Perform comprehensive verification checks
	if err := s.performVerificationChecks(participant, action); err != nil {
		return nil, err
	}

	// Step 6: Create verification record
	actionLog, err := s.createVerificationRecord(participant, action, verifier)
	if err != nil {
		return nil, err
	}

	// Step 7: Return successful result
	return &VerificationResult{
		Success:     true,
		Message:     fmt.Sprintf("Successfully verified %s for participant %s", action.Name, participant.Name),
		ActionLog:   actionLog,
		Participant: participant,
		EventAction: action,
		Timestamp:   time.Now(),
	}, nil
}

// GetParticipantVerificationHistory returns all verification records for a participant
func (s *verificationService) GetParticipantVerificationHistory(participantID string) ([]*models.ActionLog, error) {
	if participantID == "" {
		return nil, NewVerificationError("participant ID is required", ErrInvalidInput, nil)
	}

	// Validate participant exists
	if _, err := s.participantRepo.GetParticipantByID(participantID); err != nil {
		return nil, NewVerificationError("participant not found", ErrParticipantNotFound, err)
	}

	verifications, err := s.actionRepo.GetActionLogsByParticipant(participantID)
	if err != nil {
		return nil, NewVerificationError("failed to get verification history", ErrDatabaseError, err)
	}

	return verifications, nil
}

// GetEventVerifications returns paginated verification records for an event with filters
func (s *verificationService) GetEventVerifications(eventID string, filters *VerificationFilters) (*VerificationList, error) {
	if eventID == "" {
		return nil, NewVerificationError("event ID is required", ErrInvalidInput, nil)
	}

	// Validate event exists
	if _, err := s.eventRepo.GetEventByID(eventID); err != nil {
		return nil, NewVerificationError("event not found", ErrEventNotFound, err)
	}

	// Set default pagination
	if filters == nil {
		filters = &VerificationFilters{
			Page:     1,
			PageSize: 20,
		}
	}

	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.PageSize < 1 || filters.PageSize > 100 {
		filters.PageSize = 20
	}

	offset := (filters.Page - 1) * filters.PageSize

	// Get verifications with pagination
	verifications, total, err := s.actionRepo.GetActionLogsByEvent(eventID, offset, filters.PageSize)
	if err != nil {
		return nil, NewVerificationError("failed to get event verifications", ErrDatabaseError, err)
	}

	totalPages := (int(total) + filters.PageSize - 1) / filters.PageSize

	return &VerificationList{
		Verifications: verifications,
		TotalCount:    total,
		Page:          filters.Page,
		PageSize:      filters.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// GetVerificationStats returns verification statistics for an event
func (s *verificationService) GetVerificationStats(eventID string) (*VerificationStats, error) {
	if eventID == "" {
		return nil, NewVerificationError("event ID is required", ErrInvalidInput, nil)
	}

	event, err := s.eventRepo.GetEventByID(eventID)
	if err != nil {
		return nil, NewVerificationError("event not found", ErrEventNotFound, err)
	}

	// Get total participants count
	totalParticipants, err := s.participantRepo.GetParticipantCountByEventID(eventID)
	if err != nil {
		return nil, NewVerificationError("failed to get participant count", ErrDatabaseError, err)
	}

	// Get verification statistics (simplified - in real implementation, use complex queries)
	stats, err := s.calculateVerificationStatistics(eventID, totalParticipants)
	if err != nil {
		return nil, err
	}

	stats.EventID = eventID
	stats.EventTitle = event.Title

	return stats, nil
}

// CanVerifyParticipant checks if a participant can be verified for a specific action
func (s *verificationService) CanVerifyParticipant(participantID, actionID string) (bool, error) {
	if participantID == "" || actionID == "" {
		return false, NewVerificationError("participant ID and action ID are required", ErrInvalidInput, nil)
	}

	// Check if participant exists and has paid
	participant, err := s.participantRepo.GetParticipantByID(participantID)
	if err != nil {
		return false, NewVerificationError("participant not found", ErrParticipantNotFound, err)
	}

	// Check if action exists and is active
	action, err := s.eventRepo.GetEventActionByID(actionID)
	if err != nil {
		return false, NewVerificationError("action not found", ErrActionNotFound, err)
	}

	// Check payment status for paid events
	if s.isPaidEvent(participant.EventID.String()) && participant.PaymentStatus != "paid" {
		return false, NewVerificationError("participant has not paid", ErrPaymentRequired, nil)
	}

	// Check if already verified
	alreadyVerified, err := s.actionRepo.HasActionLog(participantID, actionID)
	if err != nil {
		return false, NewVerificationError("failed to check verification status", ErrDatabaseError, err)
	}

	if alreadyVerified {
		return false, NewVerificationError("already verified for this action", ErrAlreadyVerified, nil)
	}

	// Check if action belongs to the same event
	if action.EventID != participant.EventID {
		return false, NewVerificationError("action does not belong to participant's event", ErrEventMismatch, nil)
	}

	return true, nil
}

// RevertVerification allows admin to revert a verification (soft delete)
func (s *verificationService) RevertVerification(verificationID, adminID string) error {
	if verificationID == "" || adminID == "" {
		return NewVerificationError("verification ID and admin ID are required", ErrInvalidInput, nil)
	}

	// Verify admin user exists and has appropriate permissions
	admin, err := s.userRepo.GetUserByID(adminID)
	if err != nil {
		return NewVerificationError("admin user not found", ErrVerifierNotFound, err)
	}

	if admin.Role != "admin" {
		return NewVerificationError("only admin users can revert verifications", ErrPermissionDenied, nil)
	}

	// In a real implementation, you would:
	// 1. Find the verification record
	// 2. Create a revert log entry
	// 3. Soft delete or mark as reverted
	// 4. Update any related statistics

	return NewVerificationError("revert verification not yet implemented", ErrNotImplemented, nil)
}

// Private helper methods

func (s *verificationService) validateVerifyRequest(req VerifyRequest) error {
	if req.QRCodeData == "" {
		return NewVerificationError("QR code data is required", ErrInvalidInput, nil)
	}

	if req.ActionCode == "" {
		return NewVerificationError("action code is required", ErrInvalidInput, nil)
	}

	if req.VerifierID == "" {
		return NewVerificationError("verifier ID is required", ErrInvalidInput, nil)
	}

	return nil
}

func (s *verificationService) extractParticipantFromQR(qrData string) (*models.Participant, error) {
	// Try different methods to extract participant ID from QR data
	participantID, err := utils.ExtractUUIDFromQRPath(qrData)
	if err != nil {
		// If extraction fails, try direct UUID parsing
		if _, err := uuid.Parse(qrData); err == nil {
			participantID = qrData
		} else {
			return nil, NewVerificationError("invalid QR code format", ErrInvalidQRCode, err)
		}
	}

	participant, err := s.participantRepo.GetParticipantByID(participantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewVerificationError("participant not found", ErrParticipantNotFound, err)
		}
		return nil, NewVerificationError("failed to get participant", ErrDatabaseError, err)
	}

	return participant, nil
}

func (s *verificationService) getAndValidateAction(actionCode string) (*models.EventAction, error) {
	action, err := s.eventRepo.GetEventActionByCode(actionCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewVerificationError("action not found", ErrActionNotFound, err)
		}
		return nil, NewVerificationError("failed to get action", ErrDatabaseError, err)
	}

	if !action.IsActive {
		return nil, NewVerificationError("action is not active", ErrActionInactive, nil)
	}

	return action, nil
}

func (s *verificationService) performVerificationChecks(participant *models.Participant, action *models.EventAction) error {
	// Check payment status for paid events
	if s.isPaidEvent(participant.EventID.String()) && participant.PaymentStatus != "paid" {
		return NewVerificationError(
			fmt.Sprintf("participant payment status is '%s'", participant.PaymentStatus),
			ErrPaymentRequired,
			nil,
		)
	}

	// Check if already verified for this action
	alreadyVerified, err := s.actionRepo.HasActionLog(participant.ID.String(), action.ID.String())
	if err != nil {
		return NewVerificationError("failed to check existing verification", ErrDatabaseError, err)
	}

	if alreadyVerified {
		return NewVerificationError(
			fmt.Sprintf("already verified for action: %s", action.Name),
			ErrAlreadyVerified,
			nil,
		)
	}

	// Verify event consistency
	if action.EventID != participant.EventID {
		return NewVerificationError(
			"action does not belong to participant's event",
			ErrEventMismatch,
			nil,
		)
	}

	// Check event day validity (optional business rule)
	if err := s.checkEventDayValidity(action.EventDayID.String()); err != nil {
		return err
	}

	return nil
}

func (s *verificationService) isPaidEvent(eventID string) bool {
	event, err := s.eventRepo.GetEventByID(eventID)
	if err != nil {
		// If we can't get event info, assume it's free to avoid blocking verification
		return false
	}
	return event.TicketPrice > 0
}

func (s *verificationService) checkEventDayValidity(eventDayID string) error {
	eventDay, err := s.eventRepo.GetEventDayByID(eventDayID)
	if err != nil {
		// If we can't get event day, skip this check
		return nil
	}

	now := time.Now()
	eventDate := eventDay.Date

	// Only allow verification on or after the event day
	// Adjust this logic based on your business requirements
	if now.Before(eventDate.Truncate(24 * time.Hour)) {
		return NewVerificationError(
			fmt.Sprintf("verification not allowed before event day: %s", eventDate.Format("2006-01-02")),
			ErrEventNotStarted,
			nil,
		)
	}

	return nil
}

func (s *verificationService) createVerificationRecord(participant *models.Participant, action *models.EventAction, verifier *models.User) (*models.ActionLog, error) {
	actionLog := &models.ActionLog{
		ID:            uuid.New(),
		ParticipantID: participant.ID,
		ActionID:      action.ID,
		VerifiedBy:    verifier.ID,
		VerifiedAt:    time.Now(),
		CreatedAt:     time.Now(),
	}

	if err := s.actionRepo.CreateActionLog(actionLog); err != nil {
		return nil, NewVerificationError("failed to create verification record", ErrDatabaseError, err)
	}

	// Load relationships for the response
	actionLog.Participant = *participant
	actionLog.Action = *action
	actionLog.Verifier = *verifier

	return actionLog, nil
}

func (s *verificationService) calculateVerificationStatistics(eventID string, totalParticipants int64) (*VerificationStats, error) {
	// This is a simplified implementation
	// In production, you would use complex SQL queries to calculate these statistics

	// Get total verifications for the event
	verifications, _, err := s.actionRepo.GetActionLogsByEvent(eventID, 0, 1) // Just to get count
	if err != nil {
		return nil, NewVerificationError("failed to get verification data", ErrDatabaseError, err)
	}

	totalVerifications := int64(len(verifications))

	// Calculate verification rate
	verificationRate := 0.0
	if totalParticipants > 0 {
		verificationRate = float64(totalVerifications) / float64(totalParticipants)
	}

	return &VerificationStats{
		TotalVerifications: totalVerifications,
		UniqueParticipants: totalVerifications, // Simplified - in reality, count distinct participants
		VerificationRate:   verificationRate,
		MostVerifiedAction: "General Admission", // Simplified
		TopVerifier:        "System",            // Simplified
		LastVerification:   time.Now(),          // Simplified
		TodayVerifications: 0,                   // Simplified
	}, nil
}

// Error handling types and constants
type VerificationErrorType string

const (
	ErrInvalidInput        VerificationErrorType = "INVALID_INPUT"
	ErrInvalidQRCode       VerificationErrorType = "INVALID_QR_CODE"
	ErrParticipantNotFound VerificationErrorType = "PARTICIPANT_NOT_FOUND"
	ErrActionNotFound      VerificationErrorType = "ACTION_NOT_FOUND"
	ErrActionInactive      VerificationErrorType = "ACTION_INACTIVE"
	ErrVerifierNotFound    VerificationErrorType = "VERIFIER_NOT_FOUND"
	ErrPaymentRequired     VerificationErrorType = "PAYMENT_REQUIRED"
	ErrAlreadyVerified     VerificationErrorType = "ALREADY_VERIFIED"
	ErrEventNotFound       VerificationErrorType = "EVENT_NOT_FOUND"
	ErrEventMismatch       VerificationErrorType = "EVENT_MISMATCH"
	ErrEventNotStarted     VerificationErrorType = "EVENT_NOT_STARTED"
	ErrDatabaseError       VerificationErrorType = "DATABASE_ERROR"
	ErrPermissionDenied    VerificationErrorType = "PERMISSION_DENIED"
	ErrNotImplemented      VerificationErrorType = "NOT_IMPLEMENTED"
)

type VerificationError struct {
	Message string                `json:"message"`
	Code    VerificationErrorType `json:"code"`
	Details error                 `json:"details,omitempty"`
}

func (e *VerificationError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s [%s]: %v", e.Message, e.Code, e.Details)
	}
	return fmt.Sprintf("%s [%s]", e.Message, e.Code)
}

func NewVerificationError(message string, code VerificationErrorType, details error) *VerificationError {
	return &VerificationError{
		Message: message,
		Code:    code,
		Details: details,
	}
}

// Helper functions for error checking
func IsVerificationError(err error) bool {
	_, ok := err.(*VerificationError)
	return ok
}

func GetVerificationErrorCode(err error) VerificationErrorType {
	if verr, ok := err.(*VerificationError); ok {
		return verr.Code
	}
	return ""
}
