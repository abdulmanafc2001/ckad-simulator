// Package sqlite provides a file-backed implementation of the repository
// interface, so exams survive a backend restart: an in-progress session can
// be resumed and finished ones stay available for review.
//
// Only sessions and attempts are persisted. The question bank is compiled
// into the binary (see cmd/server/seed.go) and is held in memory exactly as
// the memory store holds it — persisting it would let a stale copy shadow
// the bank the running binary actually grades against.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/storeerr"

	// Pure-Go SQLite driver: no cgo, so `go build` keeps producing a
	// single self-contained binary.
	_ "modernc.org/sqlite"
)

// schema is applied on every open; each statement is idempotent.
const schema = `
CREATE TABLE IF NOT EXISTS sessions (
	id                TEXT PRIMARY KEY,
	question_ids      TEXT NOT NULL,
	started_at        TEXT NOT NULL,
	ended_at          TEXT,
	duration_limit_ns INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS attempts (
	id                 TEXT PRIMARY KEY,
	session_id         TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	question_id        TEXT NOT NULL,
	answer_text        TEXT NOT NULL DEFAULT '',
	time_spent_seconds INTEGER NOT NULL DEFAULT 0,
	check_results      TEXT NOT NULL DEFAULT '[]',
	is_correct         INTEGER NOT NULL DEFAULT 0,
	hint_count         INTEGER NOT NULL DEFAULT 0,
	score              INTEGER NOT NULL DEFAULT 0,
	started_at         TEXT NOT NULL,
	submitted_at       TEXT NOT NULL,
	seq                INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS attempts_by_session ON attempts(session_id);
`

// Store is a SQLite-backed repository. Questions are served from memory;
// sessions and attempts live in the database.
type Store struct {
	db *sql.DB

	questions map[string]*models.Question
	// order preserves the seed order so ListQuestions is deterministic.
	order []string
}

// New opens (creating it if needed) the database at path and returns a
// repository seeded with the given question bank.
func New(path string, questions []*models.Question) (*Store, error) {
	// busy_timeout keeps concurrent writers waiting rather than failing;
	// foreign_keys makes the attempts cascade actually fire; WAL keeps
	// reads from blocking behind the grading writes at the end of an exam.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	// One writer at a time. This is a single-candidate local tool, so the
	// simplicity is worth more than the concurrency.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}

	s := &Store{
		db:        db,
		questions: make(map[string]*models.Question, len(questions)),
		order:     make([]string, 0, len(questions)),
	}
	for _, q := range questions {
		if _, dup := s.questions[q.ID]; dup {
			continue
		}
		s.questions[q.ID] = q
		s.order = append(s.order, q.ID)
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// --- questions (in-memory, read-only) ---------------------------------

func (s *Store) ListQuestions() ([]*models.Question, error) {
	out := make([]*models.Question, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.questions[id])
	}
	return out, nil
}

func (s *Store) GetQuestion(id string) (*models.Question, error) {
	q, ok := s.questions[id]
	if !ok {
		return nil, storeerr.ErrNotFound
	}
	return q, nil
}

// --- time helpers ------------------------------------------------------

// timeLayout is RFC3339 with nanoseconds, stored as TEXT: it round-trips
// exactly, sorts lexicographically, and avoids every driver-specific
// timestamp conversion.
const timeLayout = time.RFC3339Nano

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

func parseTime(s string) time.Time {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

// --- sessions ----------------------------------------------------------

func (s *Store) CreateSession(sess *models.Session) error {
	ids, err := json.Marshal(sess.QuestionIDs)
	if err != nil {
		return err
	}
	var ended any
	if sess.EndedAt != nil {
		ended = formatTime(*sess.EndedAt)
	}

	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO sessions
		   (id, question_ids, started_at, ended_at, duration_limit_ns)
		 VALUES (?, ?, ?, ?, ?)`,
		sess.ID, string(ids), formatTime(sess.StartedAt), ended, int64(sess.DurationLimit))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storeerr.ErrAlreadyExists
	}
	return nil
}

func (s *Store) GetSession(id string) (*models.Session, error) {
	row := s.db.QueryRow(
		`SELECT id, question_ids, started_at, ended_at, duration_limit_ns
		   FROM sessions WHERE id = ?`, id)

	sess, err := scanSession(row)
	if err != nil {
		return nil, err
	}
	// Attempts are part of the session as the service sees it — scoring
	// reads sess.Attempts directly, so a session loaded without them would
	// silently score zero.
	attempts, err := s.ListAttempts(id)
	if err != nil {
		return nil, err
	}
	sess.Attempts = attempts
	return sess, nil
}

func (s *Store) UpdateSession(sess *models.Session) error {
	ids, err := json.Marshal(sess.QuestionIDs)
	if err != nil {
		return err
	}
	var ended any
	if sess.EndedAt != nil {
		ended = formatTime(*sess.EndedAt)
	}

	res, err := s.db.Exec(
		`UPDATE sessions
		    SET question_ids = ?, started_at = ?, ended_at = ?, duration_limit_ns = ?
		  WHERE id = ?`,
		string(ids), formatTime(sess.StartedAt), ended, int64(sess.DurationLimit), sess.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storeerr.ErrNotFound
	}
	return nil
}

func (s *Store) ListSessions() ([]*models.Session, error) {
	rows, err := s.db.Query(
		`SELECT id, question_ids, started_at, ended_at, duration_limit_ns
		   FROM sessions ORDER BY started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach attempts in one extra query rather than one per session.
	bySession, err := s.attemptsBySession()
	if err != nil {
		return nil, err
	}
	for _, sess := range out {
		sess.Attempts = bySession[sess.ID]
	}
	return out, nil
}

func (s *Store) DeleteSession(id string) error {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storeerr.ErrNotFound
	}
	return nil
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface{ Scan(dest ...any) error }

func scanSession(sc scanner) (*models.Session, error) {
	var (
		id, ids, startedAt string
		endedAt            sql.NullString
		durationNs         int64
	)
	if err := sc.Scan(&id, &ids, &startedAt, &endedAt, &durationNs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storeerr.ErrNotFound
		}
		return nil, err
	}

	sess := &models.Session{
		ID:            id,
		StartedAt:     parseTime(startedAt),
		DurationLimit: time.Duration(durationNs),
	}
	if err := json.Unmarshal([]byte(ids), &sess.QuestionIDs); err != nil {
		return nil, err
	}
	if endedAt.Valid {
		t := parseTime(endedAt.String)
		sess.EndedAt = &t
	}
	return sess, nil
}

// --- attempts ----------------------------------------------------------

func (s *Store) CreateAttempt(a *models.Attempt) error {
	checks, err := json.Marshal(a.CheckResults)
	if err != nil {
		return err
	}

	// seq preserves the order attempts were recorded in, which is the order
	// the results screen shows them in. rowid would do the same, but it is
	// reused after deletes — and re-submitting a question deletes.
	var next int64
	if err := s.db.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM attempts WHERE session_id = ?`,
		a.SessionID).Scan(&next); err != nil {
		return err
	}

	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO attempts
		   (id, session_id, question_id, answer_text, time_spent_seconds,
		    check_results, is_correct, hint_count, score, started_at, submitted_at, seq)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.SessionID, a.QuestionID, a.Answer.Text, a.Answer.TimeSpentSeconds,
		string(checks), a.IsCorrect, a.HintCount, a.Score,
		formatTime(a.StartedAt), formatTime(a.SubmittedAt), next)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storeerr.ErrAlreadyExists
	}
	return nil
}

func (s *Store) GetAttempt(id string) (*models.Attempt, error) {
	row := s.db.QueryRow(attemptColumns+` WHERE id = ?`, id)
	return scanAttempt(row)
}

func (s *Store) DeleteAttempt(id string) error {
	res, err := s.db.Exec(`DELETE FROM attempts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return storeerr.ErrNotFound
	}
	return nil
}

func (s *Store) ListAttempts(sessionID string) ([]*models.Attempt, error) {
	rows, err := s.db.Query(attemptColumns+` WHERE session_id = ? ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Attempt
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// attemptsBySession loads every attempt grouped by session, so listing
// sessions costs two queries instead of one per session.
func (s *Store) attemptsBySession() (map[string][]*models.Attempt, error) {
	rows, err := s.db.Query(attemptColumns + ` ORDER BY session_id, seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]*models.Attempt{}
	for rows.Next() {
		a, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		out[a.SessionID] = append(out[a.SessionID], a)
	}
	return out, rows.Err()
}

const attemptColumns = `SELECT id, session_id, question_id, answer_text, time_spent_seconds,
	check_results, is_correct, hint_count, score, started_at, submitted_at
	FROM attempts`

func scanAttempt(sc scanner) (*models.Attempt, error) {
	var (
		a                      models.Attempt
		checks                 string
		startedAt, submittedAt string
	)
	if err := sc.Scan(
		&a.ID, &a.SessionID, &a.QuestionID, &a.Answer.Text, &a.Answer.TimeSpentSeconds,
		&checks, &a.IsCorrect, &a.HintCount, &a.Score, &startedAt, &submittedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storeerr.ErrNotFound
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(checks), &a.CheckResults); err != nil {
		return nil, err
	}
	a.StartedAt = parseTime(startedAt)
	a.SubmittedAt = parseTime(submittedAt)
	return &a, nil
}
