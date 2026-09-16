package memory

// Package memory provides a thread-safe in-memory implementation of the
// repository interface. Everything it holds dies with the process, so it is
// the right choice for tests and for anyone who would rather not have exams
// written to disk; the sqlite package is the persistent alternative, and
// both are held to the same behaviour by the conformance suite in
// store/repository_test.go.

import (
	"sort"
	"sync"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/storeerr"
)

// Aliases of the shared sentinels, kept so existing callers can keep
// spelling them memory.ErrNotFound. They are the same values the sqlite
// store returns, so errors.Is matches either way.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = storeerr.ErrNotFound
	// ErrAlreadyExists is returned when creating an entity with a duplicate ID.
	ErrAlreadyExists = storeerr.ErrAlreadyExists
)

// Store is an in-memory repository guarded by a RWMutex.
type Store struct {
	mu        sync.RWMutex
	questions map[string]*models.Question
	sessions  map[string]*models.Session
	attempts  map[string]*models.Attempt
	// seq records the order attempts were created in, keyed by attempt ID.
	// Map iteration is randomised, so without it ListAttempts would return
	// a different order on every call; the sqlite store keeps the same
	// ordering in a column.
	seq     map[string]int64
	nextSeq int64
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
		seq:       make(map[string]int64),
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

func (s *Store) ListSessions() ([]*models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*models.Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		out = append(out, sess)
	}
	return out, nil
}

// DeleteSession removes a session and cascades to its attempts, so a reset
// cannot leave orphaned attempts behind. The sqlite store relies on a
// foreign key for the same guarantee.
func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[id]; !ok {
		return ErrNotFound
	}
	delete(s.sessions, id)
	for aid, a := range s.attempts {
		if a.SessionID == id {
			delete(s.attempts, aid)
			delete(s.seq, aid)
		}
	}
	return nil
}

func (s *Store) CreateAttempt(a *models.Attempt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.attempts[a.ID]; ok {
		return ErrAlreadyExists
	}
	s.attempts[a.ID] = a
	s.nextSeq++
	s.seq[a.ID] = s.nextSeq
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
	delete(s.seq, id)
	return nil
}

// ListAttempts returns a session's attempts in the order they were created.
func (s *Store) ListAttempts(sessionID string) ([]*models.Attempt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []*models.Attempt
	for _, a := range s.attempts {
		if a.SessionID == sessionID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return s.seq[out[i].ID] < s.seq[out[j].ID] })
	return out, nil
}
