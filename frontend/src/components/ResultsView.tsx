import { useState } from 'react'
import type { AttemptResult, EndSessionResponse } from '../api/types'
import { DifficultyBadge, DomainBadge } from './Badges'

interface ResultsViewProps {
  results: EndSessionResponse
  onRestart: () => void
}

export function ResultsView({ results, onRestart }: ResultsViewProps) {
  const pct = results.max > 0 ? Math.round((results.earned / results.max) * 100) : 0
  const correct = results.attempts.filter((a) => a.isCorrect).length

  return (
    <div className="results">
      <section className={`results-summary ${results.passed ? 'pass' : 'fail'}`}>
        <div className="results-score">
          <span className="results-pct">{pct}%</span>
          <span className="muted">
            {results.earned}/{results.max} points
          </span>
        </div>
        <div className="results-verdict">
          <h2>{results.passed ? 'Passed' : 'Not passed'}</h2>
          <p className="muted">
            Passing score is {results.passScore}%. You fully solved {correct} of{' '}
            {results.totalQuestions} questions ({results.attempts.length} attempted). Expand a task to see what failed
            and how to solve it.
          </p>
          <button className="btn btn-primary" onClick={onRestart}>
            Start a new session
          </button>
        </div>
      </section>

      <section className="results-breakdown">
        <h3>Review — what failed & how to solve it</h3>
        {results.attempts.length === 0 ? (
          <p className="muted">No questions were submitted in this session.</p>
        ) : (
          <ul className="results-review">
            {results.attempts.map((attempt) => (
              <AttemptReview key={attempt.attemptId} attempt={attempt} />
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

function AttemptReview({ attempt }: { attempt: AttemptResult }) {
  const [open, setOpen] = useState(!attempt.isCorrect)
  const failedChecks = attempt.checks.filter((c) => !c.passed)

  return (
    <li className={`review-item ${attempt.isCorrect ? 'ok' : 'bad'}`}>
      <button className="review-head" onClick={() => setOpen((o) => !o)}>
        <span className="review-mark">{attempt.isCorrect ? '✓' : '✗'}</span>
        <span className="review-title">
          {attempt.question}
          {failedChecks.length > 0 && (
            <span className="review-failed-note">
              {' '}
              — {failedChecks.length} check{failedChecks.length > 1 ? 's' : ''} failed
            </span>
          )}
        </span>
        <span className="review-badges">
          <DomainBadge domain={attempt.domain} />
          <DifficultyBadge difficulty={attempt.difficulty} />
        </span>
        <span className="review-score">
          {attempt.score}/{attempt.maxScore}
        </span>
        <span className="review-chevron">{open ? '▾' : '▸'}</span>
      </button>

      {open && (
        <div className="review-body">
          <h5>Task</h5>
          <p>{attempt.task}</p>

          <h5>Check results</h5>
          <ul className="check-list">
            {attempt.checks.map((c) => (
              <li key={c.checkId} className={c.passed ? 'check ok' : 'check bad'}>
                <span className="check-mark">{c.passed ? '✓' : '✗'}</span>
                <span className="check-desc">
                  {c.description}
                  {!c.passed && c.output && (
                    <code className="check-output">observed: {c.output}</code>
                  )}
                </span>
                <span className="check-pts">
                  {c.points}/{c.maxPoints}
                </span>
              </li>
            ))}
          </ul>

          <h5>How to do it correctly</h5>
          <pre>{attempt.solution}</pre>
        </div>
      )}
    </li>
  )
}
