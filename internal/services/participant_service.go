package services

import (
	"errors"
	"fmt"

	"event-management-backend/internal/config"
	"event-management-backend/internal/models"
	"event-management-backend/internal/repositories"
	"event-management-backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ParticipantService struct {
	repo *repositories.Repository
	cfg  *config.Config
}

func NewParticipantService(repo *repositories.Repository, cfg *config.Config) *ParticipantService {
	return &ParticipantService{repo: repo, cfg: cfg}
}

type RegisterParticipantRequest struct {
	EventID  string
	Name     string
	Email    string
	Phone    string
	Division string
	Address  string
}

type RegisterParticipantResponse struct {
	Participant *models.Participant
	QRPath      string
}

func (s *ParticipantService) RegisterParticipant(req RegisterParticipantRequest) (*RegisterParticipantResponse, error) {
	var result *RegisterParticipantResponse

	err := s.repo.ParticipantRepo.Transaction(func(tx *gorm.DB) error {
		// Get event with lock for update to prevent race condition
		event, err := s.repo.EventRepo.GetEventByID(req.EventID)
		if err != nil {
			return errors.New("event not found")
		}

		// Check if email already registered for this event
		existing, _ := s.repo.ParticipantRepo.GetParticipantByEmailAndEvent(req.Email, req.EventID)
		if existing != nil {
			return errors.New("email already registered for this event")
		}

		// Check quota if applicable
		if event.TicketQuota != nil {
			currentCount, err := s.repo.ParticipantRepo.GetParticipantCountByEventID(req.EventID)
			if err != nil {
				return errors.New("failed to check quota")
			}
			if int(currentCount) >= *event.TicketQuota {
				return errors.New("ticket quota exceeded")
			}
		}

		// Create participant
		participant := &models.Participant{
			ID:       uuid.New(),
			EventID:  uuid.MustParse(req.EventID),
			Name:     req.Name,
			Email:    req.Email,
			Phone:    req.Phone,
			Division: req.Division,
			Address:  req.Address,
			PaymentStatus: func() string {
				if event.TicketPrice > 0 {
					return "pending"
				}
				return "paid"
			}(),
		}

		if err := s.repo.ParticipantRepo.CreateParticipant(participant); err != nil {
			return err
		}

		// Generate QR code
		filename, err := utils.GenerateQRCodeImage(participant.ID.String(), s.cfg.QRDir)
		if err != nil {
			return fmt.Errorf("failed to generate QR code: %w", err)
		}

		// Update participant with QR path
		participant.QRPath = fmt.Sprintf("/qrcodes/%s", filename)
		if err := s.repo.ParticipantRepo.UpdateParticipant(participant); err != nil {
			return err
		}

		result = &RegisterParticipantResponse{
			Participant: participant,
			QRPath:      participant.QRPath,
		}
		return nil
	})

	return result, err
}

func (s *ParticipantService) ImportParticipantsCSV(eventID string, rows [][]string) (int, int, []string, error) {
	success := 0
	fail := 0
	errors := make([]string, 0)

	for i, row := range rows {
		if len(row) < 5 {
			fail++
			errors = append(errors, fmt.Sprintf("Row %d: insufficient data", i+1))
			continue
		}

		req := RegisterParticipantRequest{
			EventID:  eventID,
			Name:     row[0],
			Email:    row[1],
			Phone:    row[2],
			Division: row[3],
			Address:  row[4],
		}

		_, err := s.RegisterParticipant(req)
		if err != nil {
			fail++
			errors = append(errors, fmt.Sprintf("Row %d: %s", i+1, err.Error()))
		} else {
			success++
		}
	}

	return success, fail, errors, nil
}

func (s *ParticipantService) ListParticipants(eventID string, page, pageSize int) ([]models.Participant, int64, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	participants, total, err := s.repo.ParticipantRepo.ListParticipantsByEvent(eventID, offset, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := (int(total) + pageSize - 1) / pageSize
	return participants, total, totalPages, nil
}

func (s *ParticipantService) UpdatePaymentStatus(participantID, status string) error {
	allowedStatus := map[string]bool{"unpaid": true, "pending": true, "paid": true}
	if !allowedStatus[status] {
		return errors.New("invalid payment status")
	}

	return s.repo.ParticipantRepo.UpdatePaymentStatus(participantID, status)
}
