package sqlite_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/sqlite"
)

func bank() []*models.Question {
	return []*models.Question{{ID: "q1", Title: "First", Weight: 10}}
}

// The whole point of the sqlite store: what it holds must still be there
// after the process that wrote it is gone. Everything else it does is
// covered by the conformance suite in store/repository_test.go; this is the
// one property that suite cannot check, because it never reopens the file.
func TestStateSurvivesAReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	startedAt := time.Date(2026, 9, 16, 9, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(95 * time.Minute)

	// First run: one exam in progress, one finished with a graded attempt.
	first, err := sqlite.New(path, bank())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := first.CreateSession(&models.Session{
		ID: "live", QuestionIDs: []string{"q1"},
		StartedAt: startedAt, DurationLimit: 2 * time.Hour,
	}); err != nil {
		t.Fatalf("CreateSession(live): %v", err)
	}
	if err := first.CreateSession(&models.Session{
		ID: "done", QuestionIDs: []string{"q1"},
		StartedAt: startedAt.Add(-3 * time.Hour), EndedAt: &endedAt,
		DurationLimit: 2 * time.Hour,
	}); err != nil {
		t.Fatalf("CreateSession(done): %v", err)
	}
	if err := first.CreateAttempt(&models.Attempt{
		ID: "a1", SessionID: "done", QuestionID: "q1",
		Answer:      models.Answer{Text: "kubectl run web --image=nginx", TimeSpentSeconds: 300},
		Score:       7,
		IsCorrect:   false,
		HintCount:   1,
		StartedAt:   startedAt,
		SubmittedAt: endedAt,
		CheckResults: []models.CheckResult{
			{CheckID: "c1", Description: "pod runs", Passed: true, Points: 7, MaxPoints: 10},
		},
	}); err != nil {
		t.Fatalf("CreateAttempt: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Second run: a brand-new process opening the same file.
	second, err := sqlite.New(path, bank())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { second.Close() })

	live, err := second.GetSession("live")
	if err != nil {
		t.Fatalf("GetSession(live) after reopen: %v", err)
	}
	if !live.StartedAt.Equal(startedAt) {
		t.Errorf("StartedAt = %v, want %v — a resumed exam must keep its original deadline",
			live.StartedAt, startedAt)
	}
	if live.EndedAt != nil {
		t.Errorf("EndedAt = %v, want nil", live.EndedAt)
	}

	done, err := second.GetSession("done")
	if err != nil {
		t.Fatalf("GetSession(done) after reopen: %v", err)
	}
	if done.EndedAt == nil || !done.EndedAt.Equal(endedAt) {
		t.Errorf("EndedAt = %v, want %v", done.EndedAt, endedAt)
	}
	if len(done.Attempts) != 1 {
		t.Fatalf("want 1 attempt after reopen, got %d", len(done.Attempts))
	}

	a := done.Attempts[0]
	if a.Score != 7 || a.HintCount != 1 || a.Answer.TimeSpentSeconds != 300 {
		t.Errorf("attempt = score %d, hints %d, %ds; want 7, 1, 300s",
			a.Score, a.HintCount, a.Answer.TimeSpentSeconds)
	}
	if len(a.CheckResults) != 1 || !a.CheckResults[0].Passed || a.CheckResults[0].Points != 7 {
		t.Errorf("check results = %+v, want the stored passing check", a.CheckResults)
	}
}

// Reopening must not lose or duplicate history.
func TestReopenListsEverythingOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")

	first, err := sqlite.New(path, bank())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, id := range []string{"s1", "s2", "s3"} {
		if err := first.CreateSession(&models.Session{
			ID: id, QuestionIDs: []string{"q1"},
			StartedAt: time.Now().UTC(), DurationLimit: 2 * time.Hour,
		}); err != nil {
			t.Fatalf("CreateSession(%s): %v", id, err)
		}
	}
	first.Close()

	second, err := sqlite.New(path, bank())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { second.Close() })

	sessions, err := second.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("want 3 sessions after reopen, got %d", len(sessions))
	}
}
