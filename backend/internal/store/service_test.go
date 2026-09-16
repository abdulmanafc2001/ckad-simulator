package store

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/checker"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/dto"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
)

// TestNamespacesOfCollectsEveryExamNamespace checks that the namespaces a
// reset is allowed to delete are discovered from both prepare steps and
// cleanup commands — a prefix rule alone misses almost all of them.
func TestNamespacesOfCollectsEveryExamNamespace(t *testing.T) {
	qs := []*models.Question{
		{
			ID: "a",
			Prepare: []models.SetupStep{
				{Name: "ns", CommandArgs: "create namespace q34"},
				{Name: "workload", Namespace: "q34", YAML: "kind: Pod\n"},
			},
			Cleanup: []string{"delete namespace q34 --ignore-not-found"},
		},
		{
			ID:      "b",
			Prepare: []models.SetupStep{{Name: "ns", CommandArgs: "create ns ingress-namespace"}},
			Cleanup: []string{"delete ns ingress-namespace --ignore-not-found"},
		},
		{
			// Namespace only ever named in cleanup.
			ID:      "c",
			Cleanup: []string{"delete namespace storage-namespace --ignore-not-found", "delete pv my-pv --ignore-not-found"},
		},
	}

	got := namespacesOf(qs)
	want := []string{"ingress-namespace", "q34", "storage-namespace"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// TestNamespacesOfIgnoresNonNamespaceCleanup makes sure cluster-scoped
// deletes are never mistaken for namespaces.
func TestNamespacesOfIgnoresNonNamespaceCleanup(t *testing.T) {
	qs := []*models.Question{{
		ID: "a",
		Cleanup: []string{
			"delete pv task-pv-volume --ignore-not-found",
			"delete pvc mypvc --ignore-not-found",
			"delete pod nginx --ignore-not-found",
			"delete clusterrole reader --ignore-not-found",
		},
	}}
	if got := namespacesOf(qs); len(got) != 0 {
		t.Fatalf("expected no namespaces, got %v", got)
	}
}

// downChecker returns a Checker pointed at a binary that does not exist, so
// every cluster call fails the way it would with minikube stopped.
func downChecker(t *testing.T) *checker.Checker {
	t.Helper()
	c := checker.New()
	c.Binary = filepath.Join(t.TempDir(), "kubectl-that-does-not-exist")
	c.Timeout = 5 * time.Second
	return c
}

func testQuestion() *models.Question {
	return &models.Question{
		ID:         "q-test",
		Domain:     models.DomainApplicationDesign,
		Difficulty: models.DifficultyEasy,
		Title:      "test",
		Weight:     2,
		Prepare:    []models.SetupStep{{Name: "ns", CommandArgs: "create namespace q-test"}},
		Checks:     []models.Check{{ID: "c1", Description: "exists", Weight: 2, CommandArgs: "get ns q-test"}},
		Cleanup:    []string{"delete namespace q-test --ignore-not-found"},
	}
}

// TestStartSessionRequiresACluster is the guard itself: an exam must not
// start when the cluster is unreachable, or the candidate gets a silently
// broken two-hour session that can only ever score zero.
func TestStartSessionRequiresACluster(t *testing.T) {
	repo := NewRepository(memory.New([]*models.Question{testQuestion()}))
	svc := NewService(repo, downChecker(t))

	_, err := svc.StartSession(context.Background(), dto.StartSessionRequest{})
	if err == nil {
		t.Fatal("expected StartSession to refuse while the cluster is down")
	}
	if !errors.Is(err, ErrClusterUnavailable) {
		t.Fatalf("expected ErrClusterUnavailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "minikube start") {
		t.Errorf("error should tell the user how to fix it, got %q", err)
	}
}

// TestStartSessionFailureLeavesPriorStateIntact checks the guard runs before
// the reset: a failed start must not discard a session already in progress.
func TestStartSessionFailureLeavesPriorStateIntact(t *testing.T) {
	repo := NewRepository(memory.New([]*models.Question{testQuestion()}))
	existing := &models.Session{
		ID:            "keep-me",
		QuestionIDs:   []string{"q-test"},
		StartedAt:     time.Now().UTC(),
		DurationLimit: SessionDuration,
	}
	if err := repo.CreateSession(existing); err != nil {
		t.Fatal(err)
	}

	svc := NewService(repo, downChecker(t))
	if _, err := svc.StartSession(context.Background(), dto.StartSessionRequest{}); err == nil {
		t.Fatal("expected StartSession to fail")
	}

	if _, err := repo.GetSession("keep-me"); err != nil {
		t.Fatalf("a failed start wiped the previous session: %v", err)
	}
	sessions, err := repo.ListSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected the store to be untouched, got %d sessions", len(sessions))
	}
}

// TestNsDeleteReMatchesOnlyNamespaceDeletes guards the filter that decides
// which cleanup commands resetPriorState still runs one by one.
func TestNsDeleteReMatchesOnlyNamespaceDeletes(t *testing.T) {
	shouldMatch := []string{
		"delete namespace q34 --ignore-not-found",
		"delete ns q34 --ignore-not-found",
		"delete  namespace  q34",
	}
	shouldNotMatch := []string{
		"delete pv task-pv-volume --ignore-not-found",
		"delete pvc mypvc --ignore-not-found",
		"delete pod nginx --ignore-not-found",
	}
	for _, cm := range shouldMatch {
		if !nsDeleteRe.MatchString(cm) {
			t.Errorf("expected %q to be treated as a namespace delete", cm)
		}
	}
	for _, cm := range shouldNotMatch {
		if nsDeleteRe.MatchString(cm) {
			t.Errorf("expected %q NOT to be treated as a namespace delete", cm)
		}
	}
}
