import { useMemo, useState } from 'react'
import type { AttemptResult, Domain, EndSessionResponse } from '../api/types'
import { DOMAIN_LABELS } from '../api/types'
import { DifficultyBadge, DomainBadge } from './Badges'
import { CopyableText, CopyChip } from './Copyable'

interface ResultsViewProps {
  results: EndSessionResponse
  onRestart: () => void
}

type Filter = 'all' | 'failed' | 'passed'

/** Formats a session span as "1h 24m". */
function formatSpan(startedAt: string, endedAt: string): string {
  const ms = new Date(endedAt).getTime() - new Date(startedAt).getTime()
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  const mins = Math.round(ms / 60000)
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return h > 0 ? `${h}h ${m}m` : `${m}m`
}

export function ResultsView({ results, onRestart }: ResultsViewProps) {
  const pct = results.max > 0 ? Math.round((results.earned / results.max) * 100) : 0
  const attempted = results.attempts.filter((a) => a.attemptId !== '')
  const correct = attempted.filter((a) => a.isCorrect).length

  const [filter, setFilter] = useState<Filter>('all')

  const counts = {
    all: results.attempts.length,
    failed: results.attempts.filter((a) => !a.isCorrect).length,
    passed: results.attempts.filter((a) => a.isCorrect).length,
  }

  const shown = useMemo(
    () =>
      results.attempts.filter((a) =>
        filter === 'all' ? true : filter === 'passed' ? a.isCorrect : !a.isCorrect,
      ),
    [results.attempts, filter],
  )

  // Per-domain score, so the candidate can see which curriculum area to
  // revise rather than only a single overall number.
  const byDomain = useMemo(() => {
    const map = new Map<Domain, { earned: number; max: number; count: number }>()
    for (const a of results.attempts) {
      const cur = map.get(a.domain) ?? { earned: 0, max: 0, count: 0 }
      cur.earned += a.score
      cur.max += a.maxScore
      cur.count += 1
      map.set(a.domain, cur)
    }
    return [...map.entries()].sort((x, y) => {
      const px = x[1].max > 0 ? x[1].earned / x[1].max : 0
      const py = y[1].max > 0 ? y[1].earned / y[1].max : 0
      return px - py // weakest domain first — that is what needs the work
    })
  }, [results.attempts])

  return (
    <div className="results">
      <section className={`results-summary ${results.passed ? 'pass' : 'fail'}`}>
        <div className="results-score">
          <div
            className="score-ring"
            style={{ '--pct': pct } as React.CSSProperties}
            role="img"
            aria-label={`Score ${pct} percent`}
          >
            <span className="results-pct">{pct}%</span>
          </div>
          <span className="muted">
            {results.earned}/{results.max} points
          </span>
        </div>
        <div className="results-verdict">
          <h2>{results.passed ? 'Passed' : 'Not passed'}</h2>
          <p className="muted">
            Passing score is {results.passScore}%. You fully solved {correct} of{' '}
            {results.totalQuestions} questions ({attempted.length} attempted) in{' '}
            {formatSpan(results.startedAt, results.endedAt)}. Expand a task to see what
            failed and how to solve it.
          </p>
          <button className="btn btn-primary" onClick={onRestart}>
            Start a new session
          </button>
        </div>
      </section>

      <section className="domain-scores" aria-label="Score by domain">
        <h3>Score by domain</h3>
        <ul className="domain-score-list">
          {byDomain.map(([domain, s]) => {
            const dp = s.max > 0 ? Math.round((s.earned / s.max) * 100) : 0
            return (
              <li key={domain} className="domain-score">
                <span className="domain-score-name">{DOMAIN_LABELS[domain]}</span>
                <span className="domain-score-track">
                  <span
                    className={`domain-score-fill ${
                      dp >= results.passScore ? 'ok' : dp > 0 ? 'partial' : 'none'
                    }`}
                    style={{ width: `${dp}%` }}
                  />
                </span>
                <span className="domain-score-val">
                  {dp}% <span className="muted">({s.earned}/{s.max})</span>
                </span>
              </li>
            )
          })}
        </ul>
      </section>

      <section className="results-breakdown">
        <div className="breakdown-head">
          <h3>Review — what failed &amp; how to solve it</h3>
          <div className="seg" role="group" aria-label="Filter reviewed questions">
            {(['all', 'failed', 'passed'] as Filter[]).map((f) => (
              <button
                key={f}
                className="seg-btn"
                aria-pressed={filter === f}
                onClick={() => setFilter(f)}
              >
                {f === 'all' ? 'All' : f === 'failed' ? 'Needs work' : 'Solved'} ({counts[f]})
              </button>
            ))}
          </div>
        </div>
        {shown.length === 0 ? (
          <p className="muted empty-state">
            {filter === 'failed'
              ? 'Nothing failed — every task was fully solved.'
              : 'No fully-solved tasks in this session yet.'}
          </p>
        ) : (
          <ul className="results-review">
            {shown.map((attempt) => (
              <AttemptReview key={attempt.questionId} attempt={attempt} />
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}

function AttemptReview({ attempt }: { attempt: AttemptResult }) {
  const [open, setOpen] = useState(!attempt.isCorrect)
  const checks = attempt.checks ?? []
  const failedChecks = checks.filter((c) => !c.passed)
  const attempted = attempt.attemptId !== ''
  const pct = attempt.maxScore > 0 ? Math.round((attempt.score / attempt.maxScore) * 100) : 0

  return (
    <li className={`review-item ${attempt.isCorrect ? 'ok' : 'bad'}`}>
      <button className="review-head" onClick={() => setOpen((o) => !o)} aria-expanded={open}>
        <span className="review-mark">{attempt.isCorrect ? '✓' : '✗'}</span>
        <span className="review-title">
          {attempt.question}
          {!attempted && <span className="review-failed-note"> — not attempted</span>}
          {attempted && failedChecks.length > 0 && (
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
        <span className="review-score" title={`${pct}% of this task's points`}>
          {attempt.score}/{attempt.maxScore}
        </span>
        <span className="review-chevron" aria-hidden="true">
          {open ? '▾' : '▸'}
        </span>
      </button>

      {open && (
        <div className="review-body">
          <h5>Task</h5>
          <div className="q-task-text">
            <CopyableText text={attempt.task} />
          </div>

          {checks.length > 0 ? (
            <>
              <h5>Check results</h5>
              <ul className="check-list">
                {checks.map((c) => (
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
            </>
          ) : (
            <p className="muted">Not attempted — no checks were run.</p>
          )}

          <div className="solution-head">
            <h5>How to do it correctly</h5>
            <CopyChip value={attempt.solution} display="Copy solution" />
          </div>
          <pre>{attempt.solution}</pre>
        </div>
      )}
    </li>
  )
}
