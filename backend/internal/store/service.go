package store

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/checker"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/dto"
	"github.com/google/uuid"
)

// Exam constants matching the real CKAD exam rules.
const (
	// SessionDuration is the total time allowed for a full exam session.
	SessionDuration = 2 * time.Hour
	// PassScore is the minimum percentage required to pass the CKAD exam.
	PassScore = 66
)

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

	prepLog := s.checker.Prepare(ctx, s.questionsFor(ids))

	return &dto.StartSessionResponse{
		ID:            sess.ID,
		QuestionIDs:   ids,
		StartedAt:     sess.StartedAt,
		DurationLimit: sess.DurationLimit,
		PrepLog:       prepLog,
	}, nil
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

// EndSession finalizes a session and returns the results.
func (s *Service) EndSession(sessionID string) (*dto.EndSessionResponse, error) {
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

	return &dto.EndSessionResponse{
		ID:             sess.ID,
		StartedAt:      sess.StartedAt,
		EndedAt:        *sess.EndedAt,
		Earned:         earned,
		Max:            max,
		TotalQuestions: len(sess.QuestionIDs),
		Passed:         passed,
		PassScore:      PassScore,
		Attempts:       results,
	}, nil
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

// contains reports whether id is present in ids.
func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
