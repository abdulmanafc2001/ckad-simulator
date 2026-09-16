package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/sqlite"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/storeerr"
)

// Every Repository implementation must behave identically — the service
// layer only ever sees the interface, so a behavioural difference between
// the memory and sqlite stores would show up as an exam bug, not a storage
// bug. These tests run the same suite against both.

func repoQuestions() []*models.Question {
	return []*models.Question{
		{ID: "q1", Title: "First", Weight: 10, Domain: models.DomainApplicationDesign},
		{ID: "q2", Title: "Second", Weight: 5, Domain: models.DomainServicesNetworking},
	}
}

// repoFactory builds an empty repository seeded with repoQuestions().
type repoFactory struct {
	name string
	open func(t *testing.T) Repository
}

func repoFactories() []repoFactory {
	return []repoFactory{
		{
			name: "memory",
			open: func(*testing.T) Repository { return memory.New(repoQuestions()) },
		},
		{
			name: "sqlite",
			open: func(t *testing.T) Repository {
				t.Helper()
				db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"), repoQuestions())
				if err != nil {
					t.Fatalf("open sqlite store: %v", err)
				}
				t.Cleanup(func() { db.Close() })
				return db
			},
		},
	}
}

// forEachRepo runs fn as a subtest against every implementation.
func forEachRepo(t *testing.T, fn func(t *testing.T, repo Repository)) {
	t.Helper()
	for _, f := range repoFactories() {
		t.Run(f.name, func(t *testing.T) { fn(t, f.open(t)) })
	}
}

func testSession(id string) *models.Session {
	return &models.Session{
		ID:            id,
		QuestionIDs:   []string{"q1", "q2"},
		StartedAt:     time.Date(2026, 9, 16, 10, 30, 0, 0, time.UTC),
		DurationLimit: 2 * time.Hour,
	}
}

func testAttempt(id, sessionID, questionID string) *models.Attempt {
	return &models.Attempt{
		ID:         id,
		SessionID:  sessionID,
		QuestionID: questionID,
		Answer:     models.Answer{Text: "kubectl run nginx --image=nginx", TimeSpentSeconds: 42},
		CheckResults: []models.CheckResult{
			{CheckID: "c1", Description: "pod exists", Passed: true, Points: 5, MaxPoints: 5},
			{CheckID: "c2", Description: "image matches", Passed: false, Points: 0, MaxPoints: 5, Output: "not found"},
		},
		IsCorrect:   false,
		HintCount:   2,
		Score:       5,
		StartedAt:   time.Date(2026, 9, 16, 10, 30, 0, 0, time.UTC),
		SubmittedAt: time.Date(2026, 9, 16, 11, 15, 30, 0, time.UTC),
	}
}

func TestRepositoryQuestions(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		qs, err := repo.ListQuestions()
		if err != nil {
			t.Fatalf("ListQuestions: %v", err)
		}
		if len(qs) != 2 {
			t.Fatalf("want 2 questions, got %d", len(qs))
		}

		q, err := repo.GetQuestion("q2")
		if err != nil {
			t.Fatalf("GetQuestion: %v", err)
		}
		if q.Title != "Second" {
			t.Errorf("GetQuestion(q2).Title = %q, want %q", q.Title, "Second")
		}

		if _, err := repo.GetQuestion("nope"); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("GetQuestion(nope) error = %v, want ErrNotFound", err)
		}
	})
}

func TestRepositorySessionRoundTrip(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		want := testSession("s1")
		if err := repo.CreateSession(want); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		got, err := repo.GetSession("s1")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.ID != want.ID {
			t.Errorf("ID = %q, want %q", got.ID, want.ID)
		}
		if len(got.QuestionIDs) != 2 || got.QuestionIDs[0] != "q1" || got.QuestionIDs[1] != "q2" {
			t.Errorf("QuestionIDs = %v, want [q1 q2]", got.QuestionIDs)
		}
		if !got.StartedAt.Equal(want.StartedAt) {
			t.Errorf("StartedAt = %v, want %v", got.StartedAt, want.StartedAt)
		}
		if got.DurationLimit != want.DurationLimit {
			t.Errorf("DurationLimit = %v, want %v", got.DurationLimit, want.DurationLimit)
		}
		if got.EndedAt != nil {
			t.Errorf("EndedAt = %v, want nil for an in-progress session", got.EndedAt)
		}
	})
}

func TestRepositorySessionDuplicateAndMissing(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		if err := repo.CreateSession(testSession("s1")); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if err := repo.CreateSession(testSession("s1")); !errors.Is(err, storeerr.ErrAlreadyExists) {
			t.Errorf("duplicate CreateSession error = %v, want ErrAlreadyExists", err)
		}
		if _, err := repo.GetSession("missing"); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("GetSession(missing) error = %v, want ErrNotFound", err)
		}
		if err := repo.UpdateSession(testSession("missing")); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("UpdateSession(missing) error = %v, want ErrNotFound", err)
		}
		if err := repo.DeleteSession("missing"); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("DeleteSession(missing) error = %v, want ErrNotFound", err)
		}
	})
}

// EndSession stamps EndedAt and saves — the field is a pointer, so a store
// that dropped it would silently leave every finished exam looking active.
func TestRepositoryUpdateSessionPersistsEndedAt(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		sess := testSession("s1")
		if err := repo.CreateSession(sess); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		ended := sess.StartedAt.Add(90 * time.Minute)
		sess.EndedAt = &ended
		if err := repo.UpdateSession(sess); err != nil {
			t.Fatalf("UpdateSession: %v", err)
		}

		got, err := repo.GetSession("s1")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.EndedAt == nil {
			t.Fatal("EndedAt = nil, want the stored end time")
		}
		if !got.EndedAt.Equal(ended) {
			t.Errorf("EndedAt = %v, want %v", *got.EndedAt, ended)
		}
	})
}

func TestRepositoryListSessions(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		for _, id := range []string{"s1", "s2", "s3"} {
			if err := repo.CreateSession(testSession(id)); err != nil {
				t.Fatalf("CreateSession(%s): %v", id, err)
			}
		}
		sessions, err := repo.ListSessions()
		if err != nil {
			t.Fatalf("ListSessions: %v", err)
		}
		if len(sessions) != 3 {
			t.Fatalf("want 3 sessions, got %d", len(sessions))
		}

		if err := repo.DeleteSession("s2"); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}
		sessions, err = repo.ListSessions()
		if err != nil {
			t.Fatalf("ListSessions after delete: %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("want 2 sessions after delete, got %d", len(sessions))
		}
	})
}

func TestRepositoryAttemptRoundTrip(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		if err := repo.CreateSession(testSession("s1")); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		want := testAttempt("a1", "s1", "q1")
		if err := repo.CreateAttempt(want); err != nil {
			t.Fatalf("CreateAttempt: %v", err)
		}

		got, err := repo.GetAttempt("a1")
		if err != nil {
			t.Fatalf("GetAttempt: %v", err)
		}
		if got.SessionID != "s1" || got.QuestionID != "q1" {
			t.Errorf("session/question = %q/%q, want s1/q1", got.SessionID, got.QuestionID)
		}
		if got.Answer.Text != want.Answer.Text || got.Answer.TimeSpentSeconds != 42 {
			t.Errorf("Answer = %+v, want %+v", got.Answer, want.Answer)
		}
		if got.Score != 5 || got.HintCount != 2 || got.IsCorrect {
			t.Errorf("score/hints/correct = %d/%d/%v, want 5/2/false", got.Score, got.HintCount, got.IsCorrect)
		}
		if !got.SubmittedAt.Equal(want.SubmittedAt) {
			t.Errorf("SubmittedAt = %v, want %v", got.SubmittedAt, want.SubmittedAt)
		}

		// Check results are what the review screen renders; they must
		// survive storage intact, output text included.
		if len(got.CheckResults) != 2 {
			t.Fatalf("want 2 check results, got %d", len(got.CheckResults))
		}
		if !got.CheckResults[0].Passed || got.CheckResults[0].Points != 5 {
			t.Errorf("check 0 = %+v, want passed with 5 points", got.CheckResults[0])
		}
		if got.CheckResults[1].Output != "not found" {
			t.Errorf("check 1 output = %q, want %q", got.CheckResults[1].Output, "not found")
		}

		if _, err := repo.GetAttempt("nope"); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("GetAttempt(nope) error = %v, want ErrNotFound", err)
		}
		if err := repo.CreateAttempt(want); !errors.Is(err, storeerr.ErrAlreadyExists) {
			t.Errorf("duplicate CreateAttempt error = %v, want ErrAlreadyExists", err)
		}
	})
}

// The results screen lists attempts in the order they were recorded, which
// is the session's question order.
func TestRepositoryListAttemptsIsScopedAndOrdered(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		for _, id := range []string{"s1", "s2"} {
			if err := repo.CreateSession(testSession(id)); err != nil {
				t.Fatalf("CreateSession(%s): %v", id, err)
			}
		}
		for _, a := range []*models.Attempt{
			testAttempt("a1", "s1", "q1"),
			testAttempt("a2", "s1", "q2"),
			testAttempt("b1", "s2", "q1"),
		} {
			if err := repo.CreateAttempt(a); err != nil {
				t.Fatalf("CreateAttempt(%s): %v", a.ID, err)
			}
		}

		got, err := repo.ListAttempts("s1")
		if err != nil {
			t.Fatalf("ListAttempts: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 attempts for s1, got %d", len(got))
		}
		if got[0].ID != "a1" || got[1].ID != "a2" {
			t.Errorf("attempt order = %s,%s, want a1,a2", got[0].ID, got[1].ID)
		}

		if err := repo.DeleteAttempt("a1"); err != nil {
			t.Fatalf("DeleteAttempt: %v", err)
		}
		if got, _ = repo.ListAttempts("s1"); len(got) != 1 {
			t.Fatalf("want 1 attempt after delete, got %d", len(got))
		}
		if err := repo.DeleteAttempt("a1"); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("DeleteAttempt(a1) twice = %v, want ErrNotFound", err)
		}
	})
}

// Re-submitting a question deletes the old attempt and writes a new one
// under the same ID (see Service.SubmitAnswer) — a store that recycled
// ordering keys would reshuffle the results screen.
func TestRepositoryReplaceAttemptKeepsOrder(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		if err := repo.CreateSession(testSession("s1")); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		for _, a := range []*models.Attempt{
			testAttempt("a1", "s1", "q1"),
			testAttempt("a2", "s1", "q2"),
		} {
			if err := repo.CreateAttempt(a); err != nil {
				t.Fatalf("CreateAttempt(%s): %v", a.ID, err)
			}
		}

		if err := repo.DeleteAttempt("a1"); err != nil {
			t.Fatalf("DeleteAttempt: %v", err)
		}
		improved := testAttempt("a1", "s1", "q1")
		improved.Score = 10
		improved.IsCorrect = true
		if err := repo.CreateAttempt(improved); err != nil {
			t.Fatalf("re-CreateAttempt: %v", err)
		}

		got, err := repo.ListAttempts("s1")
		if err != nil {
			t.Fatalf("ListAttempts: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 attempts, got %d", len(got))
		}
		var replaced *models.Attempt
		for _, a := range got {
			if a.ID == "a1" {
				replaced = a
			}
		}
		if replaced == nil {
			t.Fatal("replaced attempt a1 is missing")
		}
		if replaced.Score != 10 || !replaced.IsCorrect {
			t.Errorf("replaced attempt = %d/%v, want 10/true", replaced.Score, replaced.IsCorrect)
		}
	})
}

// A session loaded from the store must carry its attempts: sessionScore
// sums sess.Attempts, so a session without them scores zero.
func TestRepositoryGetSessionCarriesAttempts(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		sess := testSession("s1")
		if err := repo.CreateSession(sess); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if err := repo.CreateAttempt(testAttempt("a1", "s1", "q1")); err != nil {
			t.Fatalf("CreateAttempt: %v", err)
		}
		// The memory store hands back the live object, so mirror what the
		// service does: keep the session's own slice in step.
		sess.Attempts = []*models.Attempt{testAttempt("a1", "s1", "q1")}
		if err := repo.UpdateSession(sess); err != nil {
			t.Fatalf("UpdateSession: %v", err)
		}

		got, err := repo.GetSession("s1")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if len(got.Attempts) != 1 {
			t.Fatalf("want 1 attempt on the loaded session, got %d", len(got.Attempts))
		}
		if got.Attempts[0].Score != 5 {
			t.Errorf("attempt score = %d, want 5", got.Attempts[0].Score)
		}
	})
}

// Deleting a session must take its attempts with it, or a reset would leave
// orphaned rows that grow forever.
func TestRepositoryDeleteSessionRemovesAttempts(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo Repository) {
		if err := repo.CreateSession(testSession("s1")); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		if err := repo.CreateAttempt(testAttempt("a1", "s1", "q1")); err != nil {
			t.Fatalf("CreateAttempt: %v", err)
		}
		if err := repo.DeleteSession("s1"); err != nil {
			t.Fatalf("DeleteSession: %v", err)
		}

		got, err := repo.ListAttempts("s1")
		if err != nil {
			t.Fatalf("ListAttempts: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("want no attempts after the session is deleted, got %d", len(got))
		}
	})
}
