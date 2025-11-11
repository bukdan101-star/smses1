package services

import (
	"errors"
	"time"

	"event-management-backend/internal/config"
	"event-management-backend/internal/models"
	"event-management-backend/internal/repositories"

	"github.com/google/uuid"
)

type EventService struct {
	repo *repositories.Repository
	cfg  *config.Config
}

func NewEventService(repo *repositories.Repository, cfg *config.Config) *EventService {
	return &EventService{repo: repo, cfg: cfg}
}

type CreateEventRequest struct {
	Title       string
	Slug        string
	Description string
	StartsAt    time.Time
	EndsAt      time.Time
	LogoPath    string
	TicketPrice float64
	TicketQuota *int
}

func (s *EventService) CreateEvent(req CreateEventRequest) (*models.Event, error) {
	// Validate dates
	if req.EndsAt.Before(req.StartsAt) {
		return nil, errors.New("end date must be after start date")
	}

	event := &models.Event{
		ID:          uuid.New(),
		Title:       req.Title,
		Slug:        req.Slug,
		Description: req.Description,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		LogoPath:    req.LogoPath,
		TicketPrice: req.TicketPrice,
		TicketQuota: req.TicketQuota,
		IsActive:    true,
	}

	if err := s.repo.EventRepo.CreateEvent(event); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventService) AddEventDay(eventID string, dayNumber int, label string, date time.Time) (*models.EventDay, error) {
	// Verify event exists
	event, err := s.repo.EventRepo.GetEventByID(eventID)
	if err != nil {
		return nil, errors.New("event not found")
	}

	day := &models.EventDay{
		ID:        uuid.New(),
		EventID:   event.ID,
		DayNumber: dayNumber,
		Label:     label,
		Date:      date,
	}

	if err := s.repo.EventRepo.CreateEventDay(day); err != nil {
		return nil, err
	}

	return day, nil
}

func (s *EventService) AddEventAction(eventID, dayID, name, code string) (*models.EventAction, error) {
	// Verify event and day exist
	event, err := s.repo.EventRepo.GetEventByID(eventID)
	if err != nil {
		return nil, errors.New("event not found")
	}

	action := &models.EventAction{
		ID:         uuid.New(),
		EventID:    event.ID,
		EventDayID: uuid.MustParse(dayID),
		Name:       name,
		Code:       code,
		IsActive:   true,
	}

	if err := s.repo.EventRepo.CreateEventAction(action); err != nil {
		return nil, err
	}

	return action, nil
}

func (s *EventService) ListEvents(page, pageSize int) ([]models.Event, int64, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	events, total, err := s.repo.EventRepo.ListEvents(offset, pageSize, nil)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := (int(total) + pageSize - 1) / pageSize
	return events, total, totalPages, nil
}

func (s *EventService) GetEvent(id string) (*models.Event, error) {
	return s.repo.EventRepo.GetEventByID(id)
}

func (s *EventService) GetEventBySlug(slug string) (*models.Event, error) {
	return s.repo.EventRepo.GetEventBySlug(slug)
}
