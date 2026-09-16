import { useCallback, useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type {
  ClusterStatus,
  Difficulty,
  Domain,
  QuestionSummary,
  SessionSummary,
} from '../api/types'
import { DOMAIN_LABELS } from '../api/types'
import { DifficultyBadge, WeightBadge } from './Badges'

interface HomeProps {
  onStart: () => void
  starting: boolean
  startError?: string
  /** Finished exams available for review, newest first. */
  history: SessionSummary[]
  onReview: (sessionId: string) => void
  reviewError?: string
}

const DOMAIN_ORDER: Domain[] = [
  'application-design',
  'application-deployment',
  'application-observability',
  'application-environment',
  'services-networking',
]

const DIFFICULTIES: Difficulty[] = ['easy', 'medium', 'hard']

/** Formats a finished exam's date as "16 Sep, 14:05". */
function formatExamDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString(undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function Home({
  onStart,
  starting,
  startError,
  history,
  onReview,
  reviewError,
}: HomeProps) {
  const [questions, setQuestions] = useState<QuestionSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Bank browsing controls — the bank is well over a hundred tasks, so it
  // needs to be searchable rather than only scrollable.
  const [query, setQuery] = useState('')
  const [difficulty, setDifficulty] = useState<Difficulty | 'all'>('all')

  // Exam tasks are provisioned on, solved against and graded from a live
  // cluster, so check it is actually up before letting anyone commit to a
  // two-hour session. Re-checked periodically so the page recovers on its
  // own once `minikube start` finishes.
  const [cluster, setCluster] = useState<ClusterStatus | null>(null)
  const [rechecking, setRechecking] = useState(false)

  const refreshCluster = useCallback(async () => {
    setRechecking(true)
    try {
      setCluster(await api.clusterStatus())
    } finally {
      setRechecking(false)
    }
  }, [])

  useEffect(() => {
    let active = true
    let timer: number | undefined
    const poll = async () => {
      const status = await api.clusterStatus()
      if (!active) return
      setCluster(status)
      // Poll faster while it is down so recovery is picked up promptly.
      timer = window.setTimeout(poll, status.connected ? 30_000 : 5_000)
    }
    void poll()
    return () => {
      active = false
      if (timer) window.clearTimeout(timer)
    }
  }, [])

  useEffect(() => {
    let active = true
    api
      .listQuestions()
      .then((res) => {
        if (active) setQuestions(res.questions)
      })
      .catch((err: unknown) => {
        if (active) {
          setError(err instanceof ApiError ? err.message : 'Failed to load questions')
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    return questions.filter(
      (item) =>
        (difficulty === 'all' || item.difficulty === difficulty) &&
        (q === '' ||
          item.title.toLowerCase().includes(q) ||
          item.description.toLowerCase().includes(q)),
    )
  }, [questions, query, difficulty])

  const grouped = useMemo(() => {
    const map = new Map<Domain, QuestionSummary[]>()
    for (const q of filtered) {
      const list = map.get(q.domain) ?? []
      list.push(q)
      map.set(q.domain, list)
    }
    return map
  }, [filtered])

  const totalWeight = useMemo(
    () => questions.reduce((sum, q) => sum + q.weight, 0),
    [questions],
  )

  const domainCount = useMemo(
    () => new Set(questions.map((q) => q.domain)).size,
    [questions],
  )

  const stats = [
    { label: 'Questions', value: questions.length, hint: 'in the bank' },
    { label: 'Domains', value: domainCount, hint: 'CKAD covered' },
    { label: 'Total points', value: totalWeight, hint: 'weighted' },
    { label: 'Exam length', value: '2h', hint: 'timed session' },
  ]

  const filtering = query.trim() !== '' || difficulty !== 'all'
  const clusterDown = cluster !== null && !cluster.connected

  return (
    <div className="home">
      <section className="hero">
        <div className="hero-eyebrow">Hands-on Kubernetes practice</div>
        <h2>Practice the Certified Kubernetes Application Developer exam</h2>
        <p>
          Work through realistic, hands-on tasks across all five CKAD domains. Start a
          timed session, solve each task with real <code>kubectl</code> commands in the
          embedded terminal, and get scored on live cluster state with reference solutions.
        </p>
        <div
          className={`cluster-status ${
            cluster === null ? 'checking' : cluster.connected ? 'up' : 'down'
          }`}
          role="status"
        >
          <span className="cluster-dot" aria-hidden="true" />
          {cluster === null ? (
            <span>Checking the Kubernetes cluster…</span>
          ) : cluster.connected ? (
            <span>
              Cluster ready — <code>{cluster.detail}</code>
            </span>
          ) : (
            <span>
              <strong>Cluster unreachable.</strong> The exam runs against a real cluster, so
              start one with <code>minikube start</code> before beginning.
              {cluster.detail && <em className="cluster-detail">{cluster.detail}</em>}
            </span>
          )}
          {clusterDown && (
            <button className="btn btn-ghost btn-sm" onClick={refreshCluster} disabled={rechecking}>
              {rechecking ? 'Checking…' : 'Re-check'}
            </button>
          )}
        </div>

        <button
          className="btn btn-primary btn-lg"
          onClick={onStart}
          disabled={starting || loading || cluster === null || !cluster.connected}
          title={clusterDown ? 'Start a Kubernetes cluster first (minikube start)' : undefined}
        >
          {starting && <span className="spinner" aria-hidden="true" />}
          {starting ? 'Preparing your exam…' : 'Start a 2-hour exam session'}
        </button>
        {starting && (
          <p className="muted start-note">
            Provisioning a namespace per task on the cluster — this can take a minute.
          </p>
        )}
        {startError && <p className="error">{startError}</p>}

        <ul className="hero-points">
          <li>17 tasks drawn fresh from the bank every session</li>
          <li>Partial credit per check, 66% to pass</li>
          <li>Solutions revealed only after you finish</li>
        </ul>
      </section>

      <section className="stats" aria-label="Overview">
        {stats.map((s) => (
          <div key={s.label} className="stat-card">
            <span className="stat-value">{s.value}</span>
            <span className="stat-label">{s.label}</span>
            <span className="stat-hint">{s.hint}</span>
          </div>
        ))}
      </section>

      {history.length > 0 && (
        <section className="history" aria-labelledby="history-head">
          <div className="history-head">
            <h3 id="history-head">Past exams</h3>
            <span className="muted">
              {history.length} kept · marks as they were scored at the time
            </span>
          </div>
          {reviewError && <p className="error">{reviewError}</p>}
          <ul className="history-list">
            {history.map((s) => {
              const pct = s.max > 0 ? Math.round((s.earned / s.max) * 100) : 0
              return (
                <li key={s.id}>
                  <button
                    className="history-row"
                    onClick={() => onReview(s.id)}
                    aria-label={`Review the exam finished on ${formatExamDate(s.endedAt)}, scored ${pct} percent`}
                  >
                    <span className={`history-verdict ${s.passed ? 'pass' : 'fail'}`}>
                      {s.passed ? 'Pass' : 'Fail'}
                    </span>
                    <span className="history-score">{pct}%</span>
                    <span className="history-meta">
                      <span className="history-date">{formatExamDate(s.endedAt)}</span>
                      <span className="muted">
                        {s.earned}/{s.max} points · {s.totalQuestions} tasks
                      </span>
                    </span>
                    <span className="history-go" aria-hidden="true">
                      Review →
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
        </section>
      )}

      <section className="bank">
        <div className="bank-head">
          <h3>Question bank</h3>
          <div className="bank-controls">
            <input
              className="bank-search"
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search tasks…"
              aria-label="Search the question bank"
              disabled={loading || Boolean(error)}
            />
            <div className="seg" role="group" aria-label="Filter by difficulty">
              <button
                className="seg-btn"
                aria-pressed={difficulty === 'all'}
                onClick={() => setDifficulty('all')}
                disabled={loading || Boolean(error)}
              >
                All
              </button>
              {DIFFICULTIES.map((d) => (
                <button
                  key={d}
                  className="seg-btn"
                  aria-pressed={difficulty === d}
                  onClick={() => setDifficulty(d)}
                  disabled={loading || Boolean(error)}
                >
                  {d[0].toUpperCase() + d.slice(1)}
                </button>
              ))}
            </div>
          </div>
        </div>

        {filtering && !loading && !error && (
          <p className="muted bank-count" role="status">
            {filtered.length} of {questions.length} tasks match
            {filtered.length > 0 && ' — '}
            {filtered.length > 0 && (
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => {
                  setQuery('')
                  setDifficulty('all')
                }}
              >
                Clear filters
              </button>
            )}
          </p>
        )}

        {loading && (
          <div className="bank-skeleton" aria-hidden="true">
            {Array.from({ length: 6 }, (_, i) => (
              <div key={i} className="skeleton skeleton-row" />
            ))}
          </div>
        )}
        {error && (
          <p className="error">
            {error}. Is the API running on <code>:8080</code>?
          </p>
        )}
        {!loading && !error && filtered.length === 0 && (
          <p className="muted empty-state">
            No tasks match “{query}”. Try a different term or clear the filters.
          </p>
        )}
        {!loading &&
          !error &&
          DOMAIN_ORDER.filter((d) => grouped.has(d)).map((domain) => (
            <div key={domain} className="domain-group">
              <h4>
                {DOMAIN_LABELS[domain]}
                <span className="domain-count">{grouped.get(domain)!.length}</span>
              </h4>
              <ul className="question-list">
                {grouped.get(domain)!.map((q) => (
                  <li key={q.id} className="question-item">
                    <div className="question-item-main">
                      <span className="question-title">{q.title}</span>
                      <p className="muted">{q.description}</p>
                    </div>
                    <div className="badges">
                      <DifficultyBadge difficulty={q.difficulty} />
                      <WeightBadge weight={q.weight} />
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ))}
      </section>
    </div>
  )
}
