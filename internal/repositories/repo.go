package repositories

import (
	"event-management-backend/internal/models"
)

// ... [kode sebelumnya tetap] ...

// === NEW METHODS FOR VERIFICATION ===

// GetActionLogsByEventAndAction untuk laporan
func (r *Repository) GetActionLogsByEventAndAction(eventID, actionID string) ([]models.ActionLog, error) {
	var logs []models.ActionLog

	query := r.DB.
		Joins("LEFT JOIN participants ON action_logs.participant_id = participants.id").
		Joins("LEFT JOIN event_actions ON action_logs.action_id = event_actions.id").
		Where("participants.event_id = ?", eventID)

	if actionID != "" {
		query = query.Where("action_logs.action_id = ?", actionID)
	}

	if err := query.
		Preload("Participant").
		Preload("Action").
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

// GetEventDaysByEventID untuk mendapatkan hari-hari event
func (r *Repository) GetEventDaysByEventID(eventID string) ([]models.EventDay, error) {
	var days []models.EventDay
	if err := r.DB.Where("event_id = ?", eventID).Order("day_number ASC").Find(&days).Error; err != nil {
		return nil, err
	}
	return days, nil
}

// GetActionsByEventDayID untuk mendapatkan aksi per hari
func (r *Repository) GetActionsByEventDayID(dayID string) ([]models.EventAction, error) {
	var actions []models.EventAction
	if err := r.DB.Where("event_day_id = ?", dayID).Find(&actions).Error; err != nil {
		return nil, err
	}
	return actions, nil
}

// GetParticipantWithEvent untuk mendapatkan data peserta lengkap dengan event
func (r *Repository) GetParticipantWithEvent(participantID string) (*models.Participant, error) {
	var participant models.Participant
	if err := r.DB.
		Preload("Event").
		Where("id = ?", participantID).
		First(&participant).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

// GetActionWithEventDay untuk mendapatkan data aksi lengkap
func (r *Repository) GetActionWithEventDay(actionID string) (*models.EventAction, error) {
	var action models.EventAction
	if err := r.DB.
		Preload("EventDay").
		Where("id = ?", actionID).
		First(&action).Error; err != nil {
		return nil, err
	}
	return &action, nil
}
