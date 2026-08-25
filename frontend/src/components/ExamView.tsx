import { useEffect, useMemo, useState } from 'react'
import type { Question, StartSessionResponse } from '../api/types'
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
  const [visited, setVisited] = useState<Record<string, boolean>>({})
  const [flagged, setFlagged] = useState<Record<string, boolean>>({})

  const question = questions[index]

  // Mark each question as visited the first time it is shown, so the
  // navigator and progress bar can reflect what the candidate has opened.
  useEffect(() => {
    setVisited((prev) => (prev[question.id] ? prev : { ...prev, [question.id]: true }))
  }, [question.id])

  const shown = hintsShown[question.id] ?? 0

  const visitedCount = Object.keys(visited).length
  const progressPct = useMemo(
    () => Math.round((visitedCount / questions.length) * 100),
    [visitedCount, questions.length],
  )

  function revealHint() {
    setHintsShown((prev) => ({
      ...prev,
      [question.id]: Math.min((prev[question.id] ?? 0) + 1, question.hints.length),
    }))
  }

  function statusOf(qid: string): 'done' | 'unanswered' {
    return visited[qid] ? 'done' : 'unanswered'
  }

  const isFlagged = Boolean(flagged[question.id])
  const flaggedCount = Object.values(flagged).filter(Boolean).length

  function toggleFlag() {
    setFlagged((prev) => ({ ...prev, [question.id]: !prev[question.id] }))
  }

  // killer.sh-style label for the dropdown: number, flag, done state.
  function optionLabel(q: Question, i: number): string {
    const parts = [`${i + 1}. ${q.title}`]
    if (visited[q.id]) parts.push('✓')
    if (flagged[q.id]) parts.push('⚑')
    return parts.join(' ')
  }

  return (
    <div className="exam">
      <div className="exam-topbar">
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
        <div className="exam-progress">
          <div className="progress-track">
            <div className="progress-fill" style={{ width: `${progressPct}%` }} />
          </div>
          <span className="muted">
            {visitedCount}/{questions.length} viewed
            {flaggedCount > 0 && ` · ⚑ ${flaggedCount} flagged`}
          </span>
        </div>
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
              {shown < question.hints.length && (
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
            the live cluster. Grading and reference solutions are revealed only after you
            press “Finish exam”.
          </p>
          <ExamTerminal />
          </aside>
        </div>
      </div>
    </div>
  )
}
