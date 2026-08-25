import { useEffect, useMemo, useState } from 'react'
import { api, ApiError } from '../api/client'
import type { Domain, QuestionSummary } from '../api/types'
import { DOMAIN_LABELS } from '../api/types'
import { DifficultyBadge, WeightBadge } from './Badges'

interface HomeProps {
  onStart: () => void
  starting: boolean
  startError?: string
}

const DOMAIN_ORDER: Domain[] = [
  'application-design',
  'application-deployment',
  'application-observability',
  'application-environment',
  'services-networking',
]

export function Home({ onStart, starting, startError }: HomeProps) {
  const [questions, setQuestions] = useState<QuestionSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

  const grouped = useMemo(() => {
    const map = new Map<Domain, QuestionSummary[]>()
    for (const q of questions) {
      const list = map.get(q.domain) ?? []
      list.push(q)
      map.set(q.domain, list)
    }
    return map
  }, [questions])

  const totalWeight = useMemo(
    () => questions.reduce((sum, q) => sum + q.weight, 0),
    [questions],
  )

  const domainCount = grouped.size

  const stats = [
    { label: 'Questions', value: questions.length, hint: 'in the bank' },
    { label: 'Domains', value: domainCount, hint: 'CKAD covered' },
    { label: 'Total points', value: totalWeight, hint: 'weighted' },
    { label: 'Exam length', value: '2h', hint: 'timed session' },
  ]

  return (
    <div className="home">
      <section className="hero">
        <div className="hero-eyebrow">Hands-on Kubernetes practice</div>
        <h2>Practice the Certified Kubernetes Application Developer exam</h2>
        <p>
          Work through realistic, hands-on tasks across all five CKAD domains. Start a
          timed session, submit your <code>kubectl</code> commands or YAML manifests, and
          get instant scoring with reference solutions.
        </p>
        <button className="btn btn-primary btn-lg" onClick={onStart} disabled={starting || loading}>
          {starting ? 'Starting…' : 'Start a 2-hour exam session'}
        </button>
        {startError && <p className="error">{startError}</p>}
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

      <section className="bank">
        <h3>Question bank</h3>
        {loading && <p className="muted">Loading questions…</p>}
        {error && (
          <p className="error">
            {error}. Is the API running on <code>:8080</code>?
          </p>
        )}
        {!loading &&
          !error &&
          DOMAIN_ORDER.filter((d) => grouped.has(d)).map((domain) => (
            <div key={domain} className="domain-group">
              <h4>{DOMAIN_LABELS[domain]}</h4>
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
