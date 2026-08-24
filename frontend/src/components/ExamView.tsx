import { useEffect, useMemo, useRef, useState } from 'react'
import { api, ApiError } from '../api/client'
import type {
  Question,
  StartSessionResponse,
  SubmitAnswerResponse,
} from '../api/types'
import { DifficultyBadge, DomainBadge, WeightBadge } from './Badges'
import { CopyableText } from './Copyable'
import { Timer } from './Timer'
import { ExamTerminal } from './Terminal'

interface ExamViewProps {
  session: StartSessionResponse
  questions: Question[]
  onFinish: () => void
  finishing: boolean
}

export function ExamView({ session, questions, onFinish, finishing }: ExamViewProps) {
  const [index, setIndex] = useState(0)
  const [hintsShown, setHintsShown] = useState<Record<string, number>>({})
  const [submissions, setSubmissions] = useState<Record<string, SubmitAnswerResponse>>({})
  const [flagged, setFlagged] = useState<Record<string, boolean>>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // First time each question is displayed, so we can report time spent.
  const startTimes = useRef<Record<string, number>>({})

  const question = questions[index]

  // Record when the current question is first shown (in an effect so we don't
  // mutate a ref or call Date.now() during render).
  useEffect(() => {
    if (!startTimes.current[question.id]) {
      startTimes.current[question.id] = Date.now()
    }
  }, [question.id])

  const submission = submissions[question.id]
  const answered = Boolean(submission)
  const shown = hintsShown[question.id] ?? 0

  const answeredCount = Object.keys(submissions).length
  const progressPct = useMemo(
    () => Math.round((answeredCount / questions.length) * 100),
    [answeredCount, questions.length],
  )

  function revealHint() {
    setHintsShown((prev) => ({
      ...prev,
      [question.id]: Math.min((prev[question.id] ?? 0) + 1, question.hints.length),
    }))
  }

  async function submit() {
    setSubmitting(true)
    setError(null)
    try {
      const timeSpentSeconds = Math.round(
        (Date.now() - (startTimes.current[question.id] ?? Date.now())) / 1000,
      )
      const res = await api.submitAnswer(session.id, {
        questionId: question.id,
        answerText: '',
        timeSpentSeconds,
        hintCount: shown,
      })
      setSubmissions((prev) => ({ ...prev, [question.id]: res }))
    } catch (err: unknown) {
      setError(err instanceof ApiError ? err.message : 'Failed to submit answer')
    } finally {
      setSubmitting(false)
    }
  }

  function statusOf(qid: string): 'done' | 'unanswered' {
    return submissions[qid] ? 'done' : 'unanswered'
  }

  const isFlagged = Boolean(flagged[question.id])
  const flaggedCount = Object.values(flagged).filter(Boolean).length

  function toggleFlag() {
    setFlagged((prev) => ({ ...prev, [question.id]: !prev[question.id] }))
  }

  // killer.sh-style label for the dropdown: number, flag, done state.
  function optionLabel(q: Question, i: number): string {
    const parts = [`${i + 1}. ${q.title}`]
    if (submissions[q.id]) parts.push('✓')
    if (flagged[q.id]) parts.push('⚑')
    return parts.join(' ')
  }

  return (
    <div className="exam">
      <div className="exam-topbar">
        <div className="exam-progress">
          <div className="progress-track">
            <div className="progress-fill" style={{ width: `${progressPct}%` }} />
          </div>
          <span className="muted">
            {answeredCount}/{questions.length} answered
            {flaggedCount > 0 && ` · ⚑ ${flaggedCount} flagged`}
          </span>
        </div>
        <select
          className="q-select"
          value={index}
          onChange={(e) => setIndex(Number(e.target.value))}
          aria-label="Jump to question"
        >
          {questions.map((q, i) => (
            <option key={q.id} value={i}>
              {optionLabel(q, i)}
            </option>
          ))}
        </select>
        <div className="exam-meta">
          <Timer
            startedAt={session.startedAt}
            durationNs={session.durationLimit}
            onExpire={onFinish}
          />
          <button className="btn btn-danger" onClick={onFinish} disabled={finishing}>
            {finishing ? 'Finishing…' : 'Finish exam'}
          </button>
        </div>
      </div>

      <div className="exam-body">
        <nav className="q-nav" aria-label="Question navigator">
          {questions.map((q, i) => (
            <button
              key={q.id}
              className={`q-dot ${i === index ? 'active' : ''} ${statusOf(q.id)} ${
                flagged[q.id] ? 'flagged' : ''
              }`}
              onClick={() => setIndex(i)}
              title={optionLabel(q, i)}
            >
              {i + 1}
            </button>
          ))}
        </nav>

        <div className="exam-columns">
          <article className="q-panel">
          <header className="q-header">
            <div className="badges">
              <DomainBadge domain={question.domain} />
              <DifficultyBadge difficulty={question.difficulty} />
              <WeightBadge weight={question.weight} />
            </div>
            <h3>
              Question {index + 1}. {question.title}
            </h3>
            <p className="muted">{question.description}</p>
            <button
              className={`btn btn-flag ${isFlagged ? 'is-flagged' : ''}`}
              onClick={toggleFlag}
              title="Flag this question to review it later"
            >
              {isFlagged ? '⚑ Flagged for review' : '⚐ Flag for review'}
            </button>
          </header>

          <section className="q-task">
            <h4>Task</h4>
            <p>
              <CopyableText text={question.task} />
            </p>
            {(question.prepare?.length ?? 0) > 0 && (
              <details className="q-setup">
                <summary>Environment setup ({question.prepare.length})</summary>
                <pre>{question.prepare.map((s) => s.command || s.yaml || s.name).join('\n')}</pre>
              </details>
            )}
          </section>

          <section className="q-hints">
            <div className="q-hints-head">
              <h4>Hints</h4>
              {shown < question.hints.length && !answered && (
                <button className="btn btn-ghost" onClick={revealHint}>
                  Reveal hint ({shown}/{question.hints.length})
                </button>
              )}
            </div>
            {shown === 0 ? (
              <p className="muted">No hints revealed. Using hints does not reduce your score in this baseline.</p>
            ) : (
              <ol className="hint-list">
                {question.hints.slice(0, shown).map((h, i) => (
                  <li key={i}>
                    <CopyableText text={h} />
                  </li>
                ))}
              </ol>
            )}
          </section>

          <footer className="q-footer">
            <button
              className="btn"
              onClick={() => setIndex((i) => Math.max(0, i - 1))}
              disabled={index === 0}
            >
              ← Previous
            </button>
            <button
              className="btn"
              onClick={() => setIndex((i) => Math.min(questions.length - 1, i + 1))}
              disabled={index === questions.length - 1}
            >
              Next →
            </button>
          </footer>
          </article>

          <aside className="term-panel">
          <h4>Terminal</h4>
          <p className="muted">
            Solve the task directly here with <code>kubectl</code>. Your commands run against
            the live cluster — press “Mark as done” when finished. Marks and solutions are
            revealed only after you finish the exam.
          </p>
          <ExamTerminal />
          {error && <p className="error">{error}</p>}
          {!answered ? (
            <button className="btn btn-primary" onClick={submit} disabled={submitting}>
              {submitting ? 'Saving…' : 'Mark as done'}
            </button>
          ) : (
            <div className="result done">
              <div className="result-head">
                <strong>✓ Marked as done</strong>
              </div>
              <p className="muted">
                You can keep working on this task and mark it again — your latest cluster
                state is what gets graded when the exam ends.
              </p>
              <button className="btn" onClick={submit} disabled={submitting}>
                {submitting ? 'Saving…' : 'Mark as done again'}
              </button>
            </div>
          )}
          </aside>
        </div>
      </div>
    </div>
  )
}
