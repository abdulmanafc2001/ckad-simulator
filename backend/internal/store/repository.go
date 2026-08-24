// Package store defines the persistence contract (Repository) and the
// application service (Service) used by the HTTP handlers.
package store

import "github.com/manaf/ckad-simulator/backend/internal/models"

// Repository abstracts persistence so the in-memory store can later be
// swapped for a real database without touching the service layer.
type Repository interface {
	ListQuestions() ([]*models.Question, error)
	GetQuestion(id string) (*models.Question, error)

	CreateSession(s *models.Session) error
	GetSession(id string) (*models.Session, error)
	UpdateSession(s *models.Session) error

	CreateAttempt(a *models.Attempt) error
	GetAttempt(id string) (*models.Attempt, error)
	DeleteAttempt(id string) error
	ListAttempts(sessionID string) ([]*models.Attempt, error)
}

// NewRepository wraps any Repository implementation so callers only depend
// on the interface.
func NewRepository(repo Repository) Repository { return repo }
