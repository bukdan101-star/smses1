package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	Role      string    `gorm:"type:varchar(20);not null;default:'staff'" json:"role"` // admin|organizer|staff
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Title       string    `gorm:"not null" json:"title"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Description string    `gorm:"type:text" json:"description"`
	StartsAt    time.Time `json:"starts_at"`
	EndsAt      time.Time `json:"ends_at"`
	LogoPath    string    `json:"logo_path"`
	TicketPrice float64   `gorm:"default:0" json:"ticket_price"`
	TicketQuota *int      `json:"ticket_quota"` // nil = unlimited
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relations
	EventDays    []EventDay    `gorm:"foreignKey:EventID" json:"event_days,omitempty"`
	Participants []Participant `gorm:"foreignKey:EventID" json:"participants,omitempty"`
}

type EventDay struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	EventID   uuid.UUID `gorm:"type:uuid;index;not null" json:"event_id"`
	DayNumber int       `gorm:"not null" json:"day_number"`
	Label     string    `gorm:"not null" json:"label"`
	Date      time.Time `gorm:"not null" json:"date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	EventActions []EventAction `gorm:"foreignKey:EventDayID" json:"event_actions,omitempty"`
}

type EventAction struct {
	ID         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	EventID    uuid.UUID `gorm:"type:uuid;index;not null" json:"event_id"`
	EventDayID uuid.UUID `gorm:"type:uuid;index;not null" json:"event_day_id"`
	Name       string    `gorm:"not null" json:"name"`
	Code       string    `gorm:"uniqueIndex;not null" json:"code"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Participant struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	EventID       uuid.UUID      `gorm:"type:uuid;index;not null" json:"event_id"`
	Name          string         `gorm:"not null" json:"name"`
	Email         string         `gorm:"not null" json:"email"`
	Phone         string         `json:"phone"`
	Division      string         `json:"division"`
	Address       string         `json:"address"`
	QRPath        string         `json:"qr_path"`
	PaymentStatus string         `gorm:"type:varchar(20);default:'unpaid'" json:"payment_status"` // unpaid|pending|paid
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Event      Event       `gorm:"foreignKey:EventID" json:"event,omitempty"`
	ActionLogs []ActionLog `gorm:"foreignKey:ParticipantID" json:"action_logs,omitempty"`
}

type ActionLog struct {
	ID            uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ParticipantID uuid.UUID `gorm:"type:uuid;index;not null" json:"participant_id"`
	ActionID      uuid.UUID `gorm:"type:uuid;index;not null" json:"action_id"`
	VerifiedBy    uuid.UUID `gorm:"type:uuid;index;not null" json:"verified_by"`
	VerifiedAt    time.Time `json:"verified_at"`
	CreatedAt     time.Time `json:"created_at"`

	// Relations
	Participant Participant `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	Action      EventAction `gorm:"foreignKey:ActionID" json:"action,omitempty"`
	Verifier    User        `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
}
