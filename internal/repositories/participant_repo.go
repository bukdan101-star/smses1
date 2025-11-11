package repositories

import (
	"event-management-backend/internal/models"
	"gorm.io/gorm"
)

type participantRepo struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepo{db: db}
}

func (r *participantRepo) CreateParticipant(participant *models.Participant) error {
	return r.db.Create(participant).Error
}

func (r *participantRepo) GetParticipantByID(id string) (*models.Participant, error) {
	var participant models.Participant
	if err := r.db.Where("id = ?", id).First(&participant).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *participantRepo) GetParticipantByEmailAndEvent(email, eventID string) (*models.Participant, error) {
	var participant models.Participant
	if err := r.db.Where("email = ? AND event_id = ?", email, eventID).First(&participant).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *participantRepo) FindParticipantByQRPath(qrPath string) (*models.Participant, error) {
	var participant models.Participant
	if err := r.db.Where("qr_path = ?", qrPath).First(&participant).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *participantRepo) GetParticipantCountByEventID(eventID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Participant{}).Where("event_id = ?", eventID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *participantRepo) ListParticipantsByEvent(eventID string, offset, limit int) ([]models.Participant, int64, error) {
	var participants []models.Participant
	var total int64

	// Count total
	if err := r.db.Model(&models.Participant{}).Where("event_id = ?", eventID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get participants with pagination
	if err := r.db.Where("event_id = ?", eventID).
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&participants).Error; err != nil {
		return nil, 0, err
	}

	return participants, total, nil
}

func (r *participantRepo) UpdateParticipant(participant *models.Participant) error {
	return r.db.Save(participant).Error
}

func (r *participantRepo) UpdatePaymentStatus(participantID, status string) error {
	return r.db.Model(&models.Participant{}).
		Where("id = ?", participantID).
		Update("payment_status", status).Error
}

func (r *participantRepo) Transaction(txFunc func(*gorm.DB) error) error {
	return r.db.Transaction(txFunc)
}