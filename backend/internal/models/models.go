package models

import "time"

// Domain describes a Kubernetes subject area covered by the CKAD exam.
type Domain string

// CKAD exam domains (weights per the official curriculum).
const (
	DomainApplicationDesign        Domain = "application-design"
	DomainApplicationDeployment    Domain = "application-deployment"
	DomainApplicationObservability Domain = "application-observability"
	DomainApplicationEnvironment   Domain = "application-environment"
	DomainServicesNetworking       Domain = "services-networking"
)

// Difficulty of a question.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// SetupStep provisions cluster state required by a task before the user
// starts solving it (killer.sh-style task environments).
type SetupStep struct {
	Name string `json:"name"`
	// Namespace is the target namespace for YAML application.
	Namespace string `json:"namespace,omitempty"`
	// YAML is applied via `kubectl apply -n <ns> -f -` when set.
	YAML string `json:"yaml,omitempty"`
	// CommandArgs are executed verbatim as `kubectl <CommandArgs>` when set.
	CommandArgs string `json:"command,omitempty"`
}

// Check is one weighted verification executed against the live cluster.
// Scoring follows killer.sh semantics: every satisfied check earns its
// points, so a partially-correct solution receives partial credit.
type Check struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
	// CommandArgs are executed verbatim as `kubectl <CommandArgs>`
	// (current kubeconfig context, i.e. minikube, is used).
	CommandArgs string `json:"command"`
	// ExpectSubstring must appear in stdout (case-insensitive).
	ExpectSubstring string `json:"expectSubstring,omitempty"`
	// ExpectRegex must match stdout (alternative to substring).
	ExpectRegex string `json:"expectRegex,omitempty"`
	// Invert flips the expectation (e.g. resource must NOT exist).
	Invert bool `json:"invert,omitempty"`
}

// Question is a single CKAD simulation exercise.
type Question struct {
	ID          string     `json:"id"`
	Domain      Domain     `json:"domain"`
	Difficulty  Difficulty `json:"difficulty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	// Prepare provisions the environment the user will work in.
	Prepare []SetupStep `json:"prepare"`
	// Task is the objective the user must accomplish.
	Task string `json:"task"`
	// Hints are progressively revealed during the attempt.
	Hints []string `json:"hints"`
	// Solution holds reference manifests/commands used for self-evaluation.
	Solution string `json:"solution"`
	// Checks verify the final cluster state and award weighted points.
	Checks []Check `json:"checks"`
	// Cleanup resets the cluster state created by this task.
	Cleanup []string `json:"cleanup"`
	// Weight is the total number of points (sum of check weights).
	Weight int `json:"weight"`
}

// Answer is a user submission for a single question.
type Answer struct {
	// Text is the user's response (commands, manifests, notes).
	Text string `json:"text"`
	// TimeSpentSeconds is how long the user spent on the question.
	TimeSpentSeconds int `json:"timeSpentSeconds"`
}

// Attempt is a single question attempt made during a session.
type Attempt struct {
	ID           string        `json:"id"`
	SessionID    string        `json:"sessionId"`
	QuestionID   string        `json:"questionId"`
	Question     *Question     `json:"question,omitempty"`
	Answer       Answer        `json:"answer"`
	CheckResults []CheckResult `json:"checkResults"`
	IsCorrect    bool          `json:"isCorrect"`
	HintCount    int           `json:"hintCount"`
	Score        int           `json:"score"`
	StartedAt    time.Time     `json:"startedAt"`
	SubmittedAt  time.Time     `json:"submittedAt"`
}

// CheckResult captures the outcome of one Check during grading.
type CheckResult struct {
	CheckID     string `json:"checkId"`
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Points      int    `json:"points"`
	MaxPoints   int    `json:"maxPoints"`
	// Output is the truncated kubectl output for transparency.
	Output string `json:"output,omitempty"`
}

// Session is a timed CKAD simulation made of multiple attempts.
type Session struct {
	ID            string        `json:"id"`
	QuestionIDs   []string      `json:"questionIds"`
	Attempts      []*Attempt    `json:"attempts"`
	StartedAt     time.Time     `json:"startedAt"`
	EndedAt       *time.Time    `json:"endedAt,omitempty"`
	DurationLimit time.Duration `json:"durationLimit"`
}
