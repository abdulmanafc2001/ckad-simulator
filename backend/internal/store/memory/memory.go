package memory

// Package memory provides a thread-safe in-memory implementation of the
// repository interface. It is intended for development and baseline use;
// swap it for a persistent backend (SQLite/Postgres) later.

import (
	"errors"
	"sync"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when creating an entity with a duplicate ID.
	ErrAlreadyExists = errors.New("already exists")
)

// Store is an in-memory repository guarded by a RWMutex.
type Store struct {
	mu        sync.RWMutex
	questions map[string]*models.Question
	sessions  map[string]*models.Session
	attempts  map[string]*models.Attempt
}

// New creates an empty in-memory store.
func New(questions []*models.Question) *Store {
	qm := make(map[string]*models.Question, len(questions))
	for _, q := range questions {
		qm[q.ID] = q
	}
	return &Store{
		questions: qm,
		sessions:  make(map[string]*models.Session),
		attempts:  make(map[string]*models.Attempt),
	}
}

func (s *Store) ListQuestions() ([]*models.Question, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Question, 0, len(s.questions))
	for _, q := range s.questions {
		out = append(out, q)
	}
	return out, nil
}

func (s *Store) GetQuestion(id string) (*models.Question, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q, ok := s.questions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return q, nil
}

func (s *Store) CreateSession(sess *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sess.ID]; ok {
		return ErrAlreadyExists
	}
	s.sessions[sess.ID] = sess
	return nil
}

func (s *Store) GetSession(id string) (*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return sess, nil
}

func (s *Store) UpdateSession(sess *models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sess.ID]; !ok {
		return ErrNotFound
	}
	s.sessions[sess.ID] = sess
	return nil
}

func (s *Store) CreateAttempt(a *models.Attempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.attempts[a.ID]; ok {
		return ErrAlreadyExists
	}
	s.attempts[a.ID] = a
	return nil
}

func (s *Store) GetAttempt(id string) (*models.Attempt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.attempts[id]
	if !ok {
		return nil, ErrNotFound
	}
	return a, nil
}

func (s *Store) DeleteAttempt(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.attempts[id]; !ok {
		return ErrNotFound
	}
	delete(s.attempts, id)
	return nil
}

func (s *Store) ListAttempts(sessionID string) ([]*models.Attempt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*models.Attempt
	for _, a := range s.attempts {
		if a.SessionID == sessionID {
			out = append(out, a)
		}
	}
	return out, nil
}
