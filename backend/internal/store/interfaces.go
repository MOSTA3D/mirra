package store

import (
	"context"
	"time"
)

// User represents a registered user.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Persona represents a distilled persona entity.
type Persona struct {
	ID         string
	OwnerID    string
	Name       string
	Slug       string
	Visibility string // private | public
	Status     string // draft | processing | ready
	Disclaimer string
	Confidence map[string]float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Source represents a data source attached to a persona.
type Source struct {
	ID        string
	PersonaID string
	Type      string // url | pdf | text
	Content   string
	Status    string // pending | processed | failed
	CreatedAt time.Time
}

// Job represents an async processing job.
type Job struct {
	ID          string
	PersonaID   string
	OwnerID     string
	Status      string // queued | running | done | failed
	CurrentStep string // ingest | extract | distill | score
	ErrorLog    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UserStore defines user persistence operations.
type UserStore interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// PersonaStore defines persona persistence operations.
type PersonaStore interface {
	Create(ctx context.Context, persona *Persona) error
	GetByID(ctx context.Context, id string) (*Persona, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*Persona, error)
	Update(ctx context.Context, persona *Persona) error
	Delete(ctx context.Context, id string) error
	AddSource(ctx context.Context, source *Source) error
	ListSources(ctx context.Context, personaID string) ([]*Source, error)
}

// JobStore defines job persistence operations.
type JobStore interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
	ListByOwner(ctx context.Context, ownerID string) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
}

// Stores groups all store interfaces — passed around as a single dependency.
type Stores struct {
	Users    UserStore
	Personas PersonaStore
	Jobs     JobStore
}

// Sentinel errors
var (
	ErrNotFound      = &StoreError{Code: "NOT_FOUND", Message: "record not found"}
	ErrAlreadyExists = &StoreError{Code: "ALREADY_EXISTS", Message: "record already exists"}
)

type StoreError struct {
	Code    string
	Message string
}

func (e *StoreError) Error() string { return e.Message }
