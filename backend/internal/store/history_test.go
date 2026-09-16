package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/storeerr"
)

// endedSession builds a finished session with one graded attempt.
func endedSession(id string, endedAt time.Time, score int) *models.Session {
	ended := endedAt
	attempt := &models.Attempt{
		ID:          id + "-a1",
		SessionID:   id,
		QuestionID:  "q-test",
		Score:       score,
		IsCorrect:   score == 2,
		StartedAt:   endedAt.Add(-time.Hour),
		SubmittedAt: endedAt,
		CheckResults: []models.CheckResult{
			{CheckID: "c1", Description: "exists", Passed: score == 2, Points: score, MaxPoints: 2},
		},
	}
	return &models.Session{
		ID:            id,
		QuestionIDs:   []string{"q-test"},
		Attempts:      []*models.Attempt{attempt},
		StartedAt:     endedAt.Add(-time.Hour),
		EndedAt:       &ended,
		DurationLimit: SessionDuration,
	}
}

// storeSession writes a session and its attempts through the repository.
func storeSession(t *testing.T, repo Repository, sess *models.Session) {
	t.Helper()
	if err := repo.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession(%s): %v", sess.ID, err)
	}
	for _, a := range sess.Attempts {
		if err := repo.CreateAttempt(a); err != nil {
			t.Fatalf("CreateAttempt(%s): %v", a.ID, err)
		}
	}
}

func newTestService(t *testing.T) (*Service, Repository) {
	t.Helper()
	repo := NewRepository(memory.New([]*models.Question{testQuestion()}))
	return NewService(repo, downChecker(t)), repo
}

// An exam interrupted by a reload or a backend restart must be resumable —
// that is the whole point of persisting it.
func TestActiveSessionResumesAnUnfinishedExam(t *testing.T) {
	svc, repo := newTestService(t)
	started := time.Now().UTC().Add(-30 * time.Minute)
	storeSession(t, repo, &models.Session{
		ID:            "live",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     started,
		DurationLimit: SessionDuration,
	})

	active, err := svc.ActiveSession()
	if err != nil {
		t.Fatalf("ActiveSession: %v", err)
	}
	if active.ID != "live" {
		t.Errorf("resumed %q, want live", active.ID)
	}
	if !active.StartedAt.Equal(started) {
		t.Errorf("StartedAt = %v, want %v — the timer must keep counting from the original start",
			active.StartedAt, started)
	}
	if active.DurationLimit != SessionDuration {
		t.Errorf("DurationLimit = %v, want %v", active.DurationLimit, SessionDuration)
	}
}

// A session past its deadline is over, whatever the store says: resuming it
// would hand the candidate an exam they can no longer submit to.
func TestActiveSessionIgnoresExpiredAndFinishedExams(t *testing.T) {
	svc, repo := newTestService(t)
	storeSession(t, repo, &models.Session{
		ID:            "expired",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     time.Now().UTC().Add(-3 * time.Hour),
		DurationLimit: SessionDuration,
	})
	storeSession(t, repo, endedSession("finished", time.Now().UTC().Add(-time.Hour), 2))

	if _, err := svc.ActiveSession(); !errors.Is(err, storeerr.ErrNotFound) {
		t.Fatalf("ActiveSession error = %v, want ErrNotFound", err)
	}
}

// Starting a new exam must clear whatever was in progress (its answers
// belong to an exam the candidate abandoned) but keep finished exams, which
// are the history the persistent store exists to preserve.
func TestResetPriorStateKeepsFinishedExams(t *testing.T) {
	svc, repo := newTestService(t)
	storeSession(t, repo, endedSession("done", time.Now().UTC().Add(-time.Hour), 2))
	storeSession(t, repo, &models.Session{
		ID:            "abandoned",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     time.Now().UTC().Add(-10 * time.Minute),
		DurationLimit: SessionDuration,
	})

	svc.resetPriorState(context.Background())

	if _, err := repo.GetSession("done"); err != nil {
		t.Errorf("finished exam was wiped: %v", err)
	}
	if _, err := repo.GetSession("abandoned"); !errors.Is(err, storeerr.ErrNotFound) {
		t.Errorf("unfinished exam survived the reset: %v", err)
	}
	// The abandoned session's attempts must go with it, or its answers
	// would linger in the store forever.
	if got, _ := repo.ListAttempts("abandoned"); len(got) != 0 {
		t.Errorf("abandoned exam left %d attempts behind", len(got))
	}
}

// History is capped, or the store grows for as long as the tool is used.
func TestResetPriorStatePrunesOldestHistory(t *testing.T) {
	svc, repo := newTestService(t)
	base := time.Now().UTC().Add(-100 * time.Hour)
	for i := range HistoryLimit + 5 {
		storeSession(t, repo, endedSession(
			string(rune('a'+i))+"-sess", base.Add(time.Duration(i)*time.Hour), 2))
	}

	svc.resetPriorState(context.Background())

	sessions, err := repo.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != HistoryLimit {
		t.Fatalf("kept %d sessions, want %d", len(sessions), HistoryLimit)
	}
	// The five oldest must be the ones dropped.
	for i := range 5 {
		id := string(rune('a'+i)) + "-sess"
		if _, err := repo.GetSession(id); !errors.Is(err, storeerr.ErrNotFound) {
			t.Errorf("expected oldest session %s to be pruned", id)
		}
	}
}

func TestListSessionSummariesNewestFirst(t *testing.T) {
	svc, repo := newTestService(t)
	now := time.Now().UTC()
	storeSession(t, repo, endedSession("older", now.Add(-3*time.Hour), 0))
	storeSession(t, repo, endedSession("newer", now.Add(-time.Hour), 2))
	// An exam still running is not history and must not be listed.
	storeSession(t, repo, &models.Session{
		ID:            "live",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     now,
		DurationLimit: SessionDuration,
	})

	got, err := svc.ListSessionSummaries()
	if err != nil {
		t.Fatalf("ListSessionSummaries: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 finished exams, got %d", len(got))
	}
	if got[0].ID != "newer" || got[1].ID != "older" {
		t.Errorf("order = %s,%s, want newer,older", got[0].ID, got[1].ID)
	}
	if got[0].Earned != 2 || got[0].Max != 2 || !got[0].Passed {
		t.Errorf("newer = %d/%d passed=%v, want 2/2 passed", got[0].Earned, got[0].Max, got[0].Passed)
	}
	if got[1].Earned != 0 || got[1].Passed {
		t.Errorf("older = %d earned passed=%v, want 0 and not passed", got[1].Earned, got[1].Passed)
	}
}

// Reviewing a past exam must replay the marks it was given, not re-grade
// against a cluster that has since been reset for the next exam.
func TestSessionResultsReplaysStoredMarks(t *testing.T) {
	svc, repo := newTestService(t)
	storeSession(t, repo, endedSession("done", time.Now().UTC().Add(-time.Hour), 2))

	res, err := svc.SessionResults("done")
	if err != nil {
		t.Fatalf("SessionResults: %v", err)
	}
	if res.Earned != 2 || res.Max != 2 || !res.Passed {
		t.Errorf("score = %d/%d passed=%v, want 2/2 passed", res.Earned, res.Max, res.Passed)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("want 1 attempt, got %d", len(res.Attempts))
	}
	a := res.Attempts[0]
	if a.QuestionID != "q-test" || !a.IsCorrect || a.Score != 2 {
		t.Errorf("attempt = %+v, want the stored passing mark", a)
	}
	// The review screen shows the reference solution and the per-check
	// detail, both pulled from the bank at render time.
	if len(a.Checks) != 1 || a.Checks[0].CheckID != "c1" {
		t.Errorf("checks = %+v, want the stored check result", a.Checks)
	}
}

func TestSessionResultsRejectsUnfinishedAndMissing(t *testing.T) {
	svc, repo := newTestService(t)
	storeSession(t, repo, &models.Session{
		ID:            "live",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     time.Now().UTC(),
		DurationLimit: SessionDuration,
	})

	if _, err := svc.SessionResults("live"); err == nil {
		t.Error("expected results of an unfinished exam to be refused")
	}
	if _, err := svc.SessionResults("nope"); !errors.Is(err, storeerr.ErrNotFound) {
		t.Errorf("SessionResults(nope) = %v, want ErrNotFound", err)
	}
}
