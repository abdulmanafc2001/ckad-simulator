package dto

// Package dto contains request/response payloads for the HTTP API.

import (
	"time"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// QuestionSummary is a question without hints/solution, used when listing.
type QuestionSummary struct {
	ID          string            `json:"id"`
	Domain      models.Domain     `json:"domain"`
	Difficulty  models.Difficulty `json:"difficulty"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Weight      int               `json:"weight"`
}

// NewQuestionSummary converts a Question to its public summary form.
func NewQuestionSummary(q *models.Question) QuestionSummary {
	return QuestionSummary{
		ID:          q.ID,
		Domain:      q.Domain,
		Difficulty:  q.Difficulty,
		Title:       q.Title,
		Description: q.Description,
		Weight:      q.Weight,
	}
}

// StartSessionRequest asks the API to create a new simulation session.
type StartSessionRequest struct {
	QuestionIDs []string `json:"questionIds"`
}

// StartSessionResponse returns the created session plus the cluster
// preparation log (killer.sh-style task environment provisioning).
type StartSessionResponse struct {
	ID            string        `json:"id"`
	QuestionIDs   []string      `json:"questionIds"`
	StartedAt     time.Time     `json:"startedAt"`
	DurationLimit time.Duration `json:"durationLimit"`
	PrepLog       []string      `json:"prepLog,omitempty"`
}

// SubmitAnswerRequest is a user submission for a question in a session.
type SubmitAnswerRequest struct {
	QuestionID       string `json:"questionId"`
	AnswerText       string `json:"answerText"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	HintCount        int    `json:"hintCount"`
}

// SubmitAnswerResponse acknowledges a submission WITHOUT revealing any
// grading information — marks are only computed when the exam ends
// (killer.sh behavior).
type SubmitAnswerResponse struct {
	AttemptID   string    `json:"attemptId"`
	QuestionID  string    `json:"questionId"`
	SubmittedAt time.Time `json:"submittedAt"`
}

// EndSessionResponse returns final results of a session.
type EndSessionResponse struct {
	ID             string          `json:"id"`
	StartedAt      time.Time       `json:"startedAt"`
	EndedAt        time.Time       `json:"endedAt"`
	Earned         int             `json:"earned"`
	Max            int             `json:"max"`
	TotalQuestions int             `json:"totalQuestions"`
	Passed         bool            `json:"passed"`
	PassScore      int             `json:"passScore"`
	Attempts       []AttemptResult `json:"attempts"`
}

// AttemptResult is the graded result of one attempt, including the
// reference solution so candidates can learn what they missed.
type AttemptResult struct {
	AttemptID  string               `json:"attemptId"`
	QuestionID string               `json:"questionId"`
	Question   string               `json:"question"`
	Task       string               `json:"task"`
	Domain     models.Domain        `json:"domain"`
	Difficulty models.Difficulty    `json:"difficulty"`
	IsCorrect  bool                 `json:"isCorrect"`
	Score      int                  `json:"score"`
	MaxScore   int                  `json:"maxScore"`
	Checks     []models.CheckResult `json:"checks"`
	Solution   string               `json:"solution"`
}
