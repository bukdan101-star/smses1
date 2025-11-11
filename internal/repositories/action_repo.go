package repositories

import (
	"event-management-backend/internal/models"
	"gorm.io/gorm"
)

type actionRepo struct {
	db *gorm.DB
}

func NewActionRepository(db *gorm.DB) ActionRepository {
	return &actionRepo{db: db}
}

func (r *actionRepo) CreateActionLog(log *models.ActionLog) error {
	return r.db.Create(log).Error
}

func (r *actionRepo) HasActionLog(participantID, actionID string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.ActionLog{}).
		Where("participant_id = ? AND action_id = ?", participantID, actionID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *actionRepo) GetActionLogsByParticipant(participantID string) ([]*models.ActionLog, error) {
	var logs []*models.ActionLog
	if err := r.db.Preload("Action").Preload("Action.EventDay").
		Where("participant_id = ?", participantID).
		Order("verified_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *actionRepo) GetActionLogsByEvent(eventID string, offset, limit int) ([]*models.ActionLog, int64, error) {
	var logs []*models.ActionLog
	var total int64

	// Count total
	if err := r.db.Model(&models.ActionLog{}).
		Joins("JOIN participants ON action_logs.participant_id = participants.id").
		Where("participants.event_id = ?", eventID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get logs with pagination
	if err := r.db.Preload("Participant").Preload("Action").Preload("Verifier").
		Joins("JOIN participants ON action_logs.participant_id = participants.id").
		Where("participants.event_id = ?", eventID).
		Offset(offset).Limit(limit).
		Order("action_logs.verified_at DESC").
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
