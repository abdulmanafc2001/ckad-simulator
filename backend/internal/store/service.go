package store

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/checker"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/dto"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/storeerr"
	"github.com/google/uuid"
)

// Exam constants matching the real CKAD exam rules.
const (
	// SessionDuration is the total time allowed for a full exam session.
	SessionDuration = 2 * time.Hour
	// PassScore is the minimum percentage required to pass the CKAD exam.
	PassScore = 66
)

// ErrClusterUnavailable is returned when an exam cannot start because the
// Kubernetes cluster is unreachable. Callers map it to 503.
var ErrClusterUnavailable = errors.New(
	"kubernetes cluster is not reachable — start it with `minikube start` and try again")

// Service contains the application business logic.
type Service struct {
	repo    Repository
	checker *checker.Checker
}

// NewService builds a Service backed by the given Repository and a kubectl
// checker that talks to the underlying (minikube) cluster.
func NewService(repo Repository, chk *checker.Checker) *Service {
	if chk == nil {
		chk = checker.New()
	}
	return &Service{repo: repo, checker: chk}
}

// ClusterStatus reports connectivity to the underlying cluster.
func (s *Service) ClusterStatus(ctx context.Context) (bool, string) {
	return s.checker.ClusterStatus(ctx)
}

// Exec runs a command typed in the exam terminal against the cluster.
func (s *Service) Exec(ctx context.Context, command string) checker.ExecResult {
	return s.checker.Exec(ctx, command)
}

// ReadFile loads a file from the exam sandbox for the built-in editors.
func (s *Service) ReadFile(path string) (string, error) {
	return s.checker.ReadFile(path)
}

// WriteFile stores an edited file inside the exam sandbox.
func (s *Service) WriteFile(path, content string) error {
	return s.checker.WriteFile(path, content)
}

// CompleteLine provides Tab completion candidates for the exam terminal.
func (s *Service) CompleteLine(line string) []string {
	return s.checker.CompleteLine(line)
}

// PrepareSession provisions the cluster environment for all questions in
// the session (namespaces, workloads) and returns a step log.
func (s *Service) PrepareSession(ctx context.Context, sessionID string) ([]string, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	qs := make([]*models.Question, 0, len(sess.QuestionIDs))
	for _, id := range sess.QuestionIDs {
		q, err := s.repo.GetQuestion(id)
		if err != nil {
			continue
		}
		qs = append(qs, q)
	}
	return s.checker.Prepare(ctx, qs), nil
}

// CleanupSession resets the cluster state created for/by a session.
func (s *Service) CleanupSession(ctx context.Context, sessionID string) ([]string, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	var cmds []string
	for _, id := range sess.QuestionIDs {
		q, err := s.repo.GetQuestion(id)
		if err != nil {
			continue
		}
		cmds = append(cmds, q.Cleanup...)
	}
	return s.checker.Cleanup(ctx, cmds), nil
}

// ListQuestions returns all questions in the bank.
func (s *Service) ListQuestions() ([]dto.QuestionSummary, error) {
	qs, err := s.repo.ListQuestions()
	if err != nil {
		return nil, err
	}
	out := make([]dto.QuestionSummary, 0, len(qs))
	for _, q := range qs {
		out = append(out, dto.NewQuestionSummary(q))
	}
	return out, nil
}

// GetQuestion returns the full question (including hints & solution) by ID.
func (s *Service) GetQuestion(id string) (*models.Question, error) {
	return s.repo.GetQuestion(id)
}

// StartSession creates a new session and provisions its cluster
// environment. If no question IDs are provided, a balanced set is picked
// automatically across all CKAD domains.
func (s *Service) StartSession(ctx context.Context, req dto.StartSessionRequest) (*dto.StartSessionResponse, error) {
	// Refuse before touching anything. Without a cluster every prepare step
	// and every check fails, which would hand the candidate a silently
	// broken 2-hour exam that can only score zero — and resetting first
	// would discard the previous session's state for nothing.
	if connected, detail := s.checker.ClusterStatus(ctx); !connected {
		if detail = strings.TrimSpace(detail); detail != "" {
			return nil, fmt.Errorf("%w (%s)", ErrClusterUnavailable, firstLine(detail))
		}
		return nil, ErrClusterUnavailable
	}

	// Wipe any leftover state from previous exams (cluster namespaces and
	// stored sessions/attempts) so every new exam starts completely fresh.
	s.resetPriorState(ctx)

	ids := req.QuestionIDs
	if len(ids) == 0 {
		picked, err := s.pickQuestionSet()
		if err != nil {
			return nil, err
		}
		ids = picked
	}
	if len(ids) == 0 {
		return nil, errors.New("no questions available")
	}

	now := time.Now().UTC()
	sess := &models.Session{
		ID:            uuid.NewString(),
		QuestionIDs:   ids,
		StartedAt:     now,
		DurationLimit: SessionDuration,
	}
	if err := s.repo.CreateSession(sess); err != nil {
		return nil, err
	}

	// resetPriorState deletes namespaces without waiting, so this exam's own
	// namespaces may still be terminating. Creating one in that state fails,
	// which used to leave tasks unprepared — and stale resources behind that
	// scored points nobody earned.
	s.checker.WaitNamespacesGone(ctx, s.namespacesFor(ids))

	prepLog := s.checker.Prepare(ctx, s.questionsFor(ids))

	return &dto.StartSessionResponse{
		ID:            sess.ID,
		QuestionIDs:   ids,
		StartedAt:     sess.StartedAt,
		DurationLimit: sess.DurationLimit,
		PrepLog:       prepLog,
	}, nil
}

// nsCreateRe and nsDeleteRe extract the namespace a question provisions or
// tears down, so a reset can collect exactly the namespaces the question
// bank owns.
var (
	nsCreateRe = regexp.MustCompile(`^create\s+(?:namespace|ns)\s+(\S+)`)
	nsDeleteRe = regexp.MustCompile(`^delete\s+(?:namespace|ns)\s+`)
)

// resetPriorState wipes any state left behind by previous exam sessions so
// the new exam begins from a clean cluster and an empty store. It is
// best-effort: failures are ignored and never prevent a new session from
// starting.
func (s *Service) resetPriorState(ctx context.Context) {
	// 1. Run the prior sessions' cleanup commands that delete cluster-scoped
	//    resources (PersistentVolumes and friends), which no namespace
	//    deletion would remove. Namespace deletes are skipped here and
	//    handled in bulk by step 2: running them one at a time, once per
	//    prior session, is what made starting an exam take minutes.
	//    Commands are de-duplicated — sessions share questions.
	if sessions, err := s.repo.ListSessions(); err == nil {
		seen := map[string]bool{}
		var cmds []string
		for _, sess := range sessions {
			for _, id := range sess.QuestionIDs {
				q, e := s.repo.GetQuestion(id)
				if e != nil {
					continue
				}
				for _, cm := range q.Cleanup {
					cm = strings.TrimSpace(cm)
					if cm == "" || seen[cm] || nsDeleteRe.MatchString(cm) {
						continue
					}
					seen[cm] = true
					cmds = append(cmds, cm)
				}
			}
		}
		if len(cmds) > 0 {
			s.checker.Cleanup(ctx, cmds)
		}
	}

	// 2. Delete every namespace the question bank owns that is still
	//    lingering, in one bulk call — this also covers namespaces from a
	//    crashed or untracked session, which the store knows nothing about.
	s.checker.ResetCluster(ctx, s.examNamespaces())

	// 3. Drop unfinished sessions so their answers never leak into the new
	//    exam, but keep finished ones: they are the candidate's history,
	//    and with a persistent store they are expected to outlive a
	//    restart. Deleting a session cascades to its attempts.
	if sessions, err := s.repo.ListSessions(); err == nil {
		var finished []*models.Session
		for _, sess := range sessions {
			if sess.EndedAt == nil {
				_ = s.repo.DeleteSession(sess.ID)
				continue
			}
			finished = append(finished, sess)
		}
		s.pruneHistory(finished)
	}
}

// HistoryLimit is how many finished exams are kept for review. Without a
// cap the store would grow for as long as the tool is used, and nobody
// revisits their fortieth-from-last attempt.
const HistoryLimit = 20

// pruneHistory deletes the oldest finished sessions beyond HistoryLimit.
func (s *Service) pruneHistory(finished []*models.Session) {
	if len(finished) <= HistoryLimit {
		return
	}
	// Oldest first, so the excess is at the front.
	sort.Slice(finished, func(i, j int) bool {
		return finished[i].EndedAt.Before(*finished[j].EndedAt)
	})
	for _, sess := range finished[:len(finished)-HistoryLimit] {
		_ = s.repo.DeleteSession(sess.ID)
	}
}

// ActiveSession returns the exam still in progress, if there is one, so the
// app can resume it after a reload or a backend restart. A session whose
// time limit has run out is not resumable — the candidate is sent to the
// results instead.
func (s *Service) ActiveSession() (*dto.StartSessionResponse, error) {
	sessions, err := s.repo.ListSessions()
	if err != nil {
		return nil, err
	}

	var active *models.Session
	now := time.Now().UTC()
	for _, sess := range sessions {
		if sess.EndedAt != nil || now.After(sess.StartedAt.Add(sess.DurationLimit)) {
			continue
		}
		// Newest wins; there should only ever be one.
		if active == nil || sess.StartedAt.After(active.StartedAt) {
			active = sess
		}
	}
	if active == nil {
		return nil, storeerr.ErrNotFound
	}

	return &dto.StartSessionResponse{
		ID:            active.ID,
		QuestionIDs:   active.QuestionIDs,
		StartedAt:     active.StartedAt,
		DurationLimit: active.DurationLimit,
	}, nil
}

// ListSessionSummaries returns the finished exams, newest first.
func (s *Service) ListSessionSummaries() ([]dto.SessionSummary, error) {
	sessions, err := s.repo.ListSessions()
	if err != nil {
		return nil, err
	}

	out := make([]dto.SessionSummary, 0, len(sessions))
	for _, sess := range sessions {
		if sess.EndedAt == nil {
			continue
		}
		earned, max := s.sessionScore(sess)
		out = append(out, dto.SessionSummary{
			ID:             sess.ID,
			StartedAt:      sess.StartedAt,
			EndedAt:        *sess.EndedAt,
			Earned:         earned,
			Max:            max,
			TotalQuestions: len(sess.QuestionIDs),
			Passed:         max > 0 && earned*100/max >= PassScore,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EndedAt.After(out[j].EndedAt) })
	return out, nil
}

// SessionResults rebuilds the results of a finished exam from the stored
// attempts. Nothing is re-graded: the cluster has moved on since (the next
// exam resets it), so the marks recorded at the time are the only honest
// answer.
func (s *Service) SessionResults(sessionID string) (*dto.EndSessionResponse, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.EndedAt == nil {
		return nil, errors.New("session has not ended yet")
	}
	return s.buildResults(sess), nil
}

// examNamespaces returns every namespace the whole question bank
// provisions. Only these are ever deleted by a reset, so a cluster shared
// with other work keeps its own namespaces.
func (s *Service) examNamespaces() []string {
	qs, err := s.repo.ListQuestions()
	if err != nil {
		return nil
	}
	return namespacesOf(qs)
}

// namespacesFor returns the namespaces used by the given question IDs.
func (s *Service) namespacesFor(ids []string) []string {
	return namespacesOf(s.questionsFor(ids))
}

// namespacesOf collects the namespaces a set of questions creates, from
// both their prepare steps and their cleanup commands.
func namespacesOf(qs []*models.Question) []string {
	set := map[string]struct{}{}
	for _, q := range qs {
		for _, step := range q.Prepare {
			if step.Namespace != "" {
				set[step.Namespace] = struct{}{}
			}
			if m := nsCreateRe.FindStringSubmatch(strings.TrimSpace(step.CommandArgs)); m != nil {
				set[m[1]] = struct{}{}
			}
		}
		for _, cm := range q.Cleanup {
			fields := strings.Fields(cm)
			if len(fields) >= 3 && fields[0] == "delete" &&
				(fields[1] == "namespace" || fields[1] == "ns") {
				set[fields[2]] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// questionsFor loads the questions for the given IDs (skipping unknowns).
func (s *Service) questionsFor(ids []string) []*models.Question {
	qs := make([]*models.Question, 0, len(ids))
	for _, id := range ids {
		if q, err := s.repo.GetQuestion(id); err == nil {
			qs = append(qs, q)
		}
	}
	return qs
}

// GetSession returns a session by ID.
func (s *Service) GetSession(id string) (*models.Session, error) {
	return s.repo.GetSession(id)
}

// examSize is the number of questions in a default exam session.
// The real CKAD exam has 17 tasks in 2 hours.
const examSize = 17

// domainWeights mirror the official CKAD curriculum weighting so generated
// exams feel like the real thing.
var domainWeights = map[models.Domain]int{
	models.DomainApplicationDesign:        20,
	models.DomainApplicationDeployment:    20,
	models.DomainApplicationObservability: 15,
	models.DomainApplicationEnvironment:   25,
	models.DomainServicesNetworking:       20,
}

// pickQuestionSet builds a dynamic exam: questions are shuffled within each
// domain, the per-domain counts follow the CKAD curriculum weights, and the
// final order is randomized — every session gets a different set.
func (s *Service) pickQuestionSet() ([]string, error) {
	qs, err := s.repo.ListQuestions()
	if err != nil {
		return nil, err
	}

	byDomain := map[models.Domain][]*models.Question{}
	for _, q := range qs {
		byDomain[q.Domain] = append(byDomain[q.Domain], q)
	}

	// Shuffle within each domain and compute total available weight.
	totalWeight := 0
	for d, list := range byDomain {
		rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
		if w, ok := domainWeights[d]; ok {
			totalWeight += w
		}
	}

	// Allocate quota per domain proportionally to its weight.
	type quota struct {
		domain models.Domain
		list   []*models.Question
		want   int
	}
	var quotas []quota
	allocated := 0
	for d, list := range byDomain {
		w := domainWeights[d]
		want := 0
		if totalWeight > 0 {
			want = examSize * w / totalWeight
		}
		if want > len(list) {
			want = len(list)
		}
		quotas = append(quotas, quota{domain: d, list: list, want: want})
		allocated += want
	}

	// Distribute any remaining slots to domains that still have questions,
	// in random order.
	remaining := examSize - allocated
	for remaining > 0 {
		rand.Shuffle(len(quotas), func(i, j int) { quotas[i], quotas[j] = quotas[j], quotas[i] })
		progressed := false
		for i := range quotas {
			if remaining == 0 {
				break
			}
			q := &quotas[i]
			if q.want < len(q.list) {
				q.want++
				remaining--
				progressed = true
			}
		}
		if !progressed {
			break // question bank exhausted
		}
	}

	var picked []string
	for _, q := range quotas {
		for _, item := range q.list[:q.want] {
			picked = append(picked, item.ID)
		}
	}

	// Randomize the order candidates see them in.
	rand.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	return picked, nil
}

// SubmitAnswer verifies the live cluster state for the question's checks
// (killer.sh style), records the attempt with partial credit, and returns
// the result.
func (s *Service) SubmitAnswer(ctx context.Context, sessionID string, req dto.SubmitAnswerRequest) (*dto.SubmitAnswerResponse, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.EndedAt != nil {
		return nil, errors.New("session already ended")
	}
	// The 2-hour limit is enforced here as well as in the browser: a timer
	// the client owns alone is no limit at all.
	if time.Now().UTC().After(sess.StartedAt.Add(sess.DurationLimit)) {
		return nil, errors.New("session time limit has expired")
	}

	question, err := s.repo.GetQuestion(req.QuestionID)
	if err != nil {
		return nil, err
	}
	if !contains(sess.QuestionIDs, question.ID) {
		return nil, errors.New("question is not part of this session")
	}

	// Grade by inspecting the cluster — partial credit per satisfied check.
	// Results stay hidden until the session ends.
	checkResults := s.checker.Grade(ctx, question)

	earned := 0
	allPassed := true
	for _, cr := range checkResults {
		earned += cr.Points
		if !cr.Passed {
			allPassed = false
		}
	}

	attempt := &models.Attempt{
		ID:           uuid.NewString(),
		SessionID:    sessionID,
		QuestionID:   question.ID,
		Answer:       models.Answer{Text: req.AnswerText, TimeSpentSeconds: req.TimeSpentSeconds},
		CheckResults: checkResults,
		IsCorrect:    allPassed,
		HintCount:    req.HintCount,
		Score:        earned,
		StartedAt:    sess.StartedAt,
		SubmittedAt:  time.Now().UTC(),
	}

	// Re-submitting a question replaces the earlier attempt so candidates
	// can keep refining their work until the exam ends.
	replaced := false
	for i, existing := range sess.Attempts {
		if existing.QuestionID == question.ID {
			_ = s.repo.DeleteAttempt(existing.ID)
			attempt.ID = existing.ID // keep the original attempt slot stable
			sess.Attempts[i] = attempt
			replaced = true
			break
		}
	}
	if !replaced {
		sess.Attempts = append(sess.Attempts, attempt)
	}

	if err := s.repo.CreateAttempt(attempt); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSession(sess); err != nil {
		return nil, err
	}

	return &dto.SubmitAnswerResponse{
		AttemptID:   attempt.ID,
		QuestionID:  question.ID,
		SubmittedAt: attempt.SubmittedAt,
	}, nil
}

// EndSession finalizes a session. It grades every question in the session
// against the current cluster state, stores the attempts, and returns the
// results. Grading happens here (rather than per-question during the exam)
// so candidates can work freely in the terminal until they finish.
func (s *Service) EndSession(ctx context.Context, sessionID string) (*dto.EndSessionResponse, error) {
	sess, err := s.repo.GetSession(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.EndedAt == nil {
		now := time.Now().UTC()
		sess.EndedAt = &now
		if err := s.repo.UpdateSession(sess); err != nil {
			return nil, err
		}
	}

	// Grade every question in the session against the live cluster.
	// This is the killer.sh model: candidates solve tasks via the terminal
	// (kubectl) and grading inspects cluster state at the end. We must
	// grade ALL questions, not just those with a prior SubmitAnswer call —
	// the exam UI never calls SubmitAnswer, it only uses the terminal.
	// If a prior attempt exists (e.g. via direct API use), preserve its
	// metadata (Answer, HintCount, timestamps) but re-grade the checks.
	prevByQID := make(map[string]*models.Attempt, len(sess.Attempts))
	for _, a := range sess.Attempts {
		prevByQID[a.QuestionID] = a
	}

	attempts := make([]*models.Attempt, 0, len(sess.QuestionIDs))
	for _, qid := range sess.QuestionIDs {
		q, err := s.repo.GetQuestion(qid)
		if err != nil {
			continue
		}
		checkResults := s.checker.Grade(ctx, q)
		earned := 0
		allPassed := true
		for _, cr := range checkResults {
			earned += cr.Points
			if !cr.Passed {
				allPassed = false
			}
		}
		prev := prevByQID[qid]
		attemptID := uuid.NewString()
		answer := models.Answer{Text: "", TimeSpentSeconds: 0}
		hintCount := 0
		startedAt := sess.StartedAt
		submittedAt := time.Now().UTC()
		if prev != nil {
			attemptID = prev.ID
			answer = prev.Answer
			hintCount = prev.HintCount
			startedAt = prev.StartedAt
			submittedAt = prev.SubmittedAt
		}
		attempts = append(attempts, &models.Attempt{
			ID:           attemptID,
			SessionID:    sess.ID,
			QuestionID:   q.ID,
			Answer:       answer,
			CheckResults: checkResults,
			IsCorrect:    allPassed,
			HintCount:    hintCount,
			Score:        earned,
			StartedAt:    startedAt,
			SubmittedAt:  submittedAt,
		})
	}

	// Replace any previously stored attempts with the freshly graded set.
	for _, a := range sess.Attempts {
		_ = s.repo.DeleteAttempt(a.ID)
	}
	sess.Attempts = attempts
	for _, a := range attempts {
		if err := s.repo.CreateAttempt(a); err != nil {
			return nil, err
		}
	}
	if err := s.repo.UpdateSession(sess); err != nil {
		return nil, err
	}

	return s.buildResults(sess), nil
}

// buildResults renders a finished session's stored attempts as the results
// payload. It reads only what is in the store, so reviewing an exam later
// shows exactly the marks it was given when it ended.
func (s *Service) buildResults(sess *models.Session) *dto.EndSessionResponse {
	earned, max := s.sessionScore(sess)
	passed := false
	if max > 0 {
		passed = earned*100/max >= PassScore
	}

	results := make([]dto.AttemptResult, 0, len(sess.Attempts))
	for _, a := range sess.Attempts {
		q, err := s.repo.GetQuestion(a.QuestionID)
		if err != nil {
			continue
		}
		results = append(results, dto.AttemptResult{
			AttemptID:  a.ID,
			QuestionID: a.QuestionID,
			Question:   q.Title,
			Task:       q.Task,
			Domain:     q.Domain,
			Difficulty: q.Difficulty,
			IsCorrect:  a.IsCorrect,
			Score:      a.Score,
			MaxScore:   q.Weight,
			Checks:     a.CheckResults,
			Solution:   q.Solution,
		})
	}

	endedAt := time.Time{}
	if sess.EndedAt != nil {
		endedAt = *sess.EndedAt
	}

	return &dto.EndSessionResponse{
		ID:             sess.ID,
		StartedAt:      sess.StartedAt,
		EndedAt:        endedAt,
		Earned:         earned,
		Max:            max,
		TotalQuestions: len(sess.QuestionIDs),
		Passed:         passed,
		PassScore:      PassScore,
		Attempts:       results,
	}
}

// sessionScore sums earned and max points for a session.
// Max is computed from ALL questions in the session, not just attempted ones,
// so unattempted questions count as zero.
func (s *Service) sessionScore(sess *models.Session) (earned, max int) {
	// Sum max from every question in the session.
	for _, qid := range sess.QuestionIDs {
		if q, err := s.repo.GetQuestion(qid); err == nil {
			max += q.Weight
		}
	}
	// Sum earned from submitted attempts only.
	for _, a := range sess.Attempts {
		earned += a.Score
	}
	return earned, max
}

// firstLine trims a multi-line kubectl error down to its first line, which
// carries the useful part ("connection refused", "no such host").
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// contains reports whether id is present in ids.
func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
