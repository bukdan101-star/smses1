package repositories

import (
	"errors"
	"fmt"
	"time"

	"event-management-backend/internal/models"

	"gorm.io/gorm"
)

type EventRepository interface {
	CreateEvent(event *models.Event) error
	GetEventByID(id string) (*models.Event, error)
	GetEventBySlug(slug string) (*models.Event, error)
	ListEvents(offset, limit int, filters *EventFilters) ([]models.Event, int64, error)
	UpdateEvent(event *models.Event) error
	SoftDeleteEvent(id string) error
	GetEventWithDays(id string) (*models.Event, error)

	// Event Days
	CreateEventDay(day *models.EventDay) error
	GetEventDayByID(id string) (*models.EventDay, error)
	GetEventDaysByEventID(eventID string) ([]models.EventDay, error)
	UpdateEventDay(day *models.EventDay) error
	DeleteEventDay(id string) error

	// Event Actions
	CreateEventAction(action *models.EventAction) error
	GetEventActionByID(id string) (*models.EventAction, error)
	GetEventActionByCode(code string) (*models.EventAction, error)
	GetEventActionsByDayID(dayID string) ([]models.EventAction, error)
	GetEventActionsByEventID(eventID string) ([]models.EventAction, error)
	UpdateEventAction(action *models.EventAction) error
	DeleteEventAction(id string) error
}

type EventFilters struct {
	IsActive    *bool
	StartsAfter *time.Time
	EndsBefore  *time.Time
	Search      string
}

type eventRepo struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepo{db: db}
}

// CreateEvent creates a new event
func (r *eventRepo) CreateEvent(event *models.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	// Check if slug already exists
	var existingEvent models.Event
	if err := r.db.Where("slug = ?", event.Slug).First(&existingEvent).Error; err == nil {
		return fmt.Errorf("event with slug '%s' already exists", event.Slug)
	}

	return r.db.Create(event).Error
}

// GetEventByID retrieves an event by its ID
func (r *eventRepo) GetEventByID(id string) (*models.Event, error) {
	if id == "" {
		return nil, errors.New("event ID cannot be empty")
	}

	var event models.Event
	if err := r.db.Where("id = ?", id).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// GetEventBySlug retrieves an event by its slug
func (r *eventRepo) GetEventBySlug(slug string) (*models.Event, error) {
	if slug == "" {
		return nil, errors.New("event slug cannot be empty")
	}

	var event models.Event
	if err := r.db.Where("slug = ?", slug).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event not found with slug: %s", slug)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

// GetEventWithDays retrieves an event with its associated days and actions
func (r *eventRepo) GetEventWithDays(id string) (*models.Event, error) {
	if id == "" {
		return nil, errors.New("event ID cannot be empty")
	}

	var event models.Event
	if err := r.db.
		Preload("EventDays", func(db *gorm.DB) *gorm.DB {
			return db.Order("event_days.day_number ASC")
		}).
		Preload("EventDays.EventActions", func(db *gorm.DB) *gorm.DB {
			return db.Order("event_actions.name ASC")
		}).
		Where("id = ?", id).
		First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get event with days: %w", err)
	}

	return &event, nil
}

// ListEvents retrieves a paginated list of events with optional filters
func (r *eventRepo) ListEvents(offset, limit int, filters *EventFilters) ([]models.Event, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var events []models.Event
	var total int64

	query := r.db.Model(&models.Event{})

	// Apply filters
	if filters != nil {
		if filters.IsActive != nil {
			query = query.Where("is_active = ?", *filters.IsActive)
		}
		if filters.StartsAfter != nil {
			query = query.Where("starts_at >= ?", *filters.StartsAfter)
		}
		if filters.EndsBefore != nil {
			query = query.Where("ends_at <= ?", *filters.EndsBefore)
		}
		if filters.Search != "" {
			searchTerm := "%" + filters.Search + "%"
			query = query.Where("title ILIKE ? OR description ILIKE ?", searchTerm, searchTerm)
		}
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Fetch paginated results
	if err := query.
		Preload("EventDays").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	return events, total, nil
}

// UpdateEvent updates an existing event
func (r *eventRepo) UpdateEvent(event *models.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	// Check if event exists
	var existingEvent models.Event
	if err := r.db.Where("id = ?", event.ID).First(&existingEvent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("event not found with ID: %s", event.ID)
		}
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	// Check if slug is being changed and if it conflicts with another event
	if event.Slug != existingEvent.Slug {
		var slugConflict models.Event
		if err := r.db.Where("slug = ? AND id != ?", event.Slug, event.ID).First(&slugConflict).Error; err == nil {
			return fmt.Errorf("event with slug '%s' already exists", event.Slug)
		}
	}

	return r.db.Save(event).Error
}

// SoftDeleteEvent soft deletes an event by setting is_active to false
func (r *eventRepo) SoftDeleteEvent(id string) error {
	if id == "" {
		return errors.New("event ID cannot be empty")
	}

	result := r.db.Model(&models.Event{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to soft delete event: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("event not found with ID: %s", id)
	}

	return nil
}

// CreateEventDay creates a new event day
func (r *eventRepo) CreateEventDay(day *models.EventDay) error {
	if day == nil {
		return errors.New("event day cannot be nil")
	}

	// Check if event exists
	var event models.Event
	if err := r.db.Where("id = ?", day.EventID).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("event not found with ID: %s", day.EventID)
		}
		return fmt.Errorf("failed to check event existence: %w", err)
	}

	// Check for duplicate day number for the same event
	var existingDay models.EventDay
	if err := r.db.Where("event_id = ? AND day_number = ?", day.EventID, day.DayNumber).First(&existingDay).Error; err == nil {
		return fmt.Errorf("day number %d already exists for this event", day.DayNumber)
	}

	return r.db.Create(day).Error
}

// GetEventDayByID retrieves an event day by its ID
func (r *eventRepo) GetEventDayByID(id string) (*models.EventDay, error) {
	if id == "" {
		return nil, errors.New("event day ID cannot be empty")
	}

	var day models.EventDay
	if err := r.db.
		Preload("EventActions").
		Where("id = ?", id).
		First(&day).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event day not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get event day: %w", err)
	}

	return &day, nil
}

// GetEventDaysByEventID retrieves all event days for a specific event
func (r *eventRepo) GetEventDaysByEventID(eventID string) ([]models.EventDay, error) {
	if eventID == "" {
		return nil, errors.New("event ID cannot be empty")
	}

	var days []models.EventDay
	if err := r.db.
		Preload("EventActions").
		Where("event_id = ?", eventID).
		Order("day_number ASC").
		Find(&days).Error; err != nil {
		return nil, fmt.Errorf("failed to get event days: %w", err)
	}

	return days, nil
}

// UpdateEventDay updates an existing event day
func (r *eventRepo) UpdateEventDay(day *models.EventDay) error {
	if day == nil {
		return errors.New("event day cannot be nil")
	}

	// Check if event day exists
	var existingDay models.EventDay
	if err := r.db.Where("id = ?", day.ID).First(&existingDay).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("event day not found with ID: %s", day.ID)
		}
		return fmt.Errorf("failed to check event day existence: %w", err)
	}

	// Check for duplicate day number if it's being changed
	if day.DayNumber != existingDay.DayNumber {
		var duplicateDay models.EventDay
		if err := r.db.Where("event_id = ? AND day_number = ? AND id != ?",
			day.EventID, day.DayNumber, day.ID).First(&duplicateDay).Error; err == nil {
			return fmt.Errorf("day number %d already exists for this event", day.DayNumber)
		}
	}

	return r.db.Save(day).Error
}

// DeleteEventDay deletes an event day
func (r *eventRepo) DeleteEventDay(id string) error {
	if id == "" {
		return errors.New("event day ID cannot be empty")
	}

	// Check if there are any actions associated with this day
	var actionCount int64
	if err := r.db.Model(&models.EventAction{}).Where("event_day_id = ?", id).Count(&actionCount).Error; err != nil {
		return fmt.Errorf("failed to check event actions: %w", err)
	}

	if actionCount > 0 {
		return errors.New("cannot delete event day with associated actions")
	}

	result := r.db.Where("id = ?", id).Delete(&models.EventDay{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete event day: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("event day not found with ID: %s", id)
	}

	return nil
}

// CreateEventAction creates a new event action
func (r *eventRepo) CreateEventAction(action *models.EventAction) error {
	if action == nil {
		return errors.New("event action cannot be nil")
	}

	// Check if event day exists
	var eventDay models.EventDay
	if err := r.db.Where("id = ?", action.EventDayID).First(&eventDay).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("event day not found with ID: %s", action.EventDayID)
		}
		return fmt.Errorf("failed to check event day existence: %w", err)
	}

	// Check if code already exists
	var existingAction models.EventAction
	if err := r.db.Where("code = ?", action.Code).First(&existingAction).Error; err == nil {
		return fmt.Errorf("event action with code '%s' already exists", action.Code)
	}

	return r.db.Create(action).Error
}

// GetEventActionByID retrieves an event action by its ID
func (r *eventRepo) GetEventActionByID(id string) (*models.EventAction, error) {
	if id == "" {
		return nil, errors.New("event action ID cannot be empty")
	}

	var action models.EventAction
	if err := r.db.
		Preload("EventDay").
		Where("id = ?", id).
		First(&action).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event action not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get event action: %w", err)
	}

	return &action, nil
}

// GetEventActionByCode retrieves an event action by its code
func (r *eventRepo) GetEventActionByCode(code string) (*models.EventAction, error) {
	if code == "" {
		return nil, errors.New("event action code cannot be empty")
	}

	var action models.EventAction
	if err := r.db.
		Preload("EventDay").
		Where("code = ? AND is_active = ?", code, true).
		First(&action).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("event action not found with code: %s", code)
		}
		return nil, fmt.Errorf("failed to get event action: %w", err)
	}

	return &action, nil
}

// GetEventActionsByDayID retrieves all event actions for a specific event day
func (r *eventRepo) GetEventActionsByDayID(dayID string) ([]models.EventAction, error) {
	if dayID == "" {
		return nil, errors.New("event day ID cannot be empty")
	}

	var actions []models.EventAction
	if err := r.db.
		Where("event_day_id = ? AND is_active = ?", dayID, true).
		Order("name ASC").
		Find(&actions).Error; err != nil {
		return nil, fmt.Errorf("failed to get event actions: %w", err)
	}

	return actions, nil
}

// GetEventActionsByEventID retrieves all event actions for a specific event
func (r *eventRepo) GetEventActionsByEventID(eventID string) ([]models.EventAction, error) {
	if eventID == "" {
		return nil, errors.New("event ID cannot be empty")
	}

	var actions []models.EventAction
	if err := r.db.
		Joins("JOIN event_days ON event_actions.event_day_id = event_days.id").
		Where("event_days.event_id = ? AND event_actions.is_active = ?", eventID, true).
		Order("event_days.day_number ASC, event_actions.name ASC").
		Find(&actions).Error; err != nil {
		return nil, fmt.Errorf("failed to get event actions: %w", err)
	}

	return actions, nil
}

// UpdateEventAction updates an existing event action
func (r *eventRepo) UpdateEventAction(action *models.EventAction) error {
	if action == nil {
		return errors.New("event action cannot be nil")
	}

	// Check if event action exists
	var existingAction models.EventAction
	if err := r.db.Where("id = ?", action.ID).First(&existingAction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("event action not found with ID: %s", action.ID)
		}
		return fmt.Errorf("failed to check event action existence: %w", err)
	}

	// Check if code is being changed and if it conflicts with another action
	if action.Code != existingAction.Code {
		var codeConflict models.EventAction
		if err := r.db.Where("code = ? AND id != ?", action.Code, action.ID).First(&codeConflict).Error; err == nil {
			return fmt.Errorf("event action with code '%s' already exists", action.Code)
		}
	}

	return r.db.Save(action).Error
}

// DeleteEventAction soft deletes an event action by setting is_active to false
func (r *eventRepo) DeleteEventAction(id string) error {
	if id == "" {
		return errors.New("event action ID cannot be empty")
	}

	result := r.db.Model(&models.EventAction{}).
		Where("id = ?", id).
		Update("is_active", false)

	if result.Error != nil {
		return fmt.Errorf("failed to delete event action: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("event action not found with ID: %s", id)
	}

	return nil
}
