// Types mirroring the backend API payloads (see backend/internal/store/dto
// and backend/internal/models). Keep these in sync with the Go structs.

export type Domain =
  | 'application-design'
  | 'application-deployment'
  | 'application-observability'
  | 'application-environment'
  | 'services-networking'

export type Difficulty = 'easy' | 'medium' | 'hard'

/** Human-readable labels for the CKAD curriculum domains. */
export const DOMAIN_LABELS: Record<Domain, string> = {
  'application-design': 'Application Design & Build',
  'application-deployment': 'Application Deployment',
  'application-observability': 'Observability & Maintenance',
  'application-environment': 'Environment, Config & Security',
  'services-networking': 'Services & Networking',
}

/** A question without hints/solution, returned when listing the bank. */
export interface QuestionSummary {
  id: string
  domain: Domain
  difficulty: Difficulty
  title: string
  description: string
  weight: number
}

/** A cluster preparation step run before the user starts the task. */
export interface SetupStep {
  name: string
  namespace?: string
  yaml?: string
  command?: string
}

/** One weighted verification executed against the live cluster. */
export interface Check {
  id: string
  description: string
  weight: number
  /** kubectl args executed verbatim as `kubectl <command>`. */
  command: string
}

/** Outcome of a single check during grading. */
export interface CheckResult {
  checkId: string
  description: string
  passed: boolean
  points: number
  maxPoints: number
  output?: string
}

/** A full question including preparation, task, hints, solution and checks. */
export interface Question extends QuestionSummary {
  prepare: SetupStep[]
  task: string
  hints: string[]
  solution: string
  checks: Check[]
  cleanup: string[]
}

export interface StartSessionRequest {
  questionIds?: string[]
}

export interface StartSessionResponse {
  id: string
  questionIds: string[]
  startedAt: string
  /** Go time.Duration serialized as nanoseconds. */
  durationLimit: number
}

export interface SubmitAnswerRequest {
  questionId: string
  answerText: string
  timeSpentSeconds: number
  hintCount: number
}

export interface SessionScore {
  earned: number
  max: number
}

/** Request to run one command in the exam terminal. */
export interface ExecRequest {
  command: string
}

/** Result of a terminal command executed on the backend. */
export interface ExecResponse {
  output: string
  exitCode: number
}

/** Acknowledgement of a submission — no grading info until the exam ends. */
export interface SubmitAnswerResponse {
  attemptId: string
  questionId: string
  submittedAt: string
}

export interface AttemptResult {
  attemptId: string
  questionId: string
  question: string
  task: string
  domain: Domain
  difficulty: Difficulty
  isCorrect: boolean
  score: number
  maxScore: number
  checks: CheckResult[] | null
  solution: string
}

export interface EndSessionResponse {
  id: string
  startedAt: string
  endedAt: string
  earned: number
  max: number
  totalQuestions: number
  passed: boolean
  passScore: number
  attempts: AttemptResult[]
}
