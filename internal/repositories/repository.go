package repositories

import (
	"event-management-backend/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	DB              *gorm.DB
	EventRepo       EventRepository
	UserRepo        UserRepository
	ParticipantRepo ParticipantRepository
	ActionRepo      ActionRepository
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		DB:              db,
		EventRepo:       NewEventRepository(db),
		UserRepo:        NewUserRepository(db),
		ParticipantRepo: NewParticipantRepository(db),
		ActionRepo:      NewActionRepository(db),
	}
}

func AutoMigrate(db *gorm.DB) error {
	// Enable UUID extension
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error; err != nil {
		return err
	}

	// Migrate models
	return db.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.EventDay{},
		&models.EventAction{},
		&models.Participant{},
		&models.ActionLog{},
	)
}

// Interface definitions
type UserRepository interface {
	GetUserByEmail(email string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	CreateUser(user *models.User) error
	UpdateUser(user *models.User) error
}

type ParticipantRepository interface {
	CreateParticipant(participant *models.Participant) error
	GetParticipantByID(id string) (*models.Participant, error)
	GetParticipantByEmailAndEvent(email, eventID string) (*models.Participant, error)
	FindParticipantByQRPath(qrPath string) (*models.Participant, error)
	GetParticipantCountByEventID(eventID string) (int64, error)
	ListParticipantsByEvent(eventID string, offset, limit int) ([]models.Participant, int64, error)
	UpdateParticipant(participant *models.Participant) error
	UpdatePaymentStatus(participantID, status string) error
	Transaction(txFunc func(*gorm.DB) error) error
}

type ActionRepository interface {
	CreateActionLog(log *models.ActionLog) error
	HasActionLog(participantID, actionID string) (bool, error)
	GetActionLogsByParticipant(participantID string) ([]*models.ActionLog, error)
	GetActionLogsByEvent(eventID string, offset, limit int) ([]*models.ActionLog, int64, error)
}
