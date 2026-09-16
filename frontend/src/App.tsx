import { useCallback, useEffect, useState } from 'react'
import './App.css'
import { api, ApiError } from './api/client'
import type {
  EndSessionResponse,
  Question,
  SessionSummary,
  StartSessionResponse,
} from './api/types'
import { ExamView } from './components/ExamView'
import { pruneExamProgress } from './components/examProgress'
import { Home } from './components/Home'
import { ResultsView } from './components/ResultsView'

type View = 'home' | 'exam' | 'results'
type Theme = 'light' | 'dark'

function getInitialTheme(): Theme {
  const stored = localStorage.getItem('ckad-theme')
  return stored === 'light' || stored === 'dark' ? stored : 'dark'
}

export default function App() {
  const [view, setView] = useState<View>('home')
  const [session, setSession] = useState<StartSessionResponse | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [results, setResults] = useState<EndSessionResponse | null>(null)

  const [starting, setStarting] = useState(false)
  const [startError, setStartError] = useState<string | undefined>()
  const [finishing, setFinishing] = useState(false)

  // Exams are persisted, so the app asks the server what is going on before
  // it renders anything: there may be an exam to resume, and there is a
  // history to list. `booting` keeps the home screen from flashing up in
  // front of an exam that is about to be restored.
  const [booting, setBooting] = useState(true)
  const [history, setHistory] = useState<SessionSummary[]>([])
  const [reviewing, setReviewing] = useState(false)
  const [reviewError, setReviewError] = useState<string | undefined>()

  const refreshHistory = useCallback(async () => {
    try {
      setHistory((await api.sessions()).history)
    } catch {
      // History is a convenience; failing to load it must not break the app.
    }
  }, [])

  useEffect(() => {
    let active = true
    const boot = async () => {
      try {
        const { active: live, history: past } = await api.sessions()
        if (!active) return
        setHistory(past)
        // Anything left over from an exam that is no longer running is dead
        // weight in localStorage.
        pruneExamProgress(live?.id ?? null)
        if (live) {
          // An exam was still running when the page (or the backend) went
          // away. Restore it with its original start time so the timer
          // picks up where it left off rather than granting a fresh 2 hours.
          const full = await Promise.all(live.questionIds.map((id) => api.getQuestion(id)))
          if (!active) return
          setSession(live)
          setQuestions(full)
          setView('exam')
        }
      } catch {
        // No server, or nothing to restore — start at the home screen.
      } finally {
        if (active) setBooting(false)
      }
    }
    void boot()
    return () => {
      active = false
    }
  }, [])

  const [theme, setTheme] = useState<Theme>(getInitialTheme)
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('ckad-theme', theme)
  }, [theme])
  const toggleTheme = useCallback(
    () => setTheme((t) => (t === 'dark' ? 'light' : 'dark')),
    [],
  )

  const handleStart = useCallback(async () => {
    setStarting(true)
    setStartError(undefined)
    try {
      const sess = await api.startSession()
      const full = await Promise.all(sess.questionIds.map((id) => api.getQuestion(id)))
      setSession(sess)
      setQuestions(full)
      setResults(null)
      setView('exam')
    } catch (err: unknown) {
      setStartError(err instanceof ApiError ? err.message : 'Could not start a session')
    } finally {
      setStarting(false)
    }
  }, [])

  const handleFinish = useCallback(async () => {
    if (!session) return
    setFinishing(true)
    try {
      const res = await api.endSession(session.id)
      setResults(res)
      setReviewing(false)
      setView('results')
      pruneExamProgress(null)
      void refreshHistory()
    } catch (err: unknown) {
      setStartError(err instanceof ApiError ? err.message : 'Could not finish the session')
    } finally {
      setFinishing(false)
    }
  }, [session, refreshHistory])

  const handleRestart = useCallback(() => {
    setSession(null)
    setQuestions([])
    setResults(null)
    setReviewing(false)
    setView('home')
  }, [])

  // Opening a finished exam from the history. The marks are replayed from
  // the store — nothing is re-graded, because the cluster those answers
  // were scored against is long gone.
  const handleReview = useCallback(async (sessionId: string) => {
    setReviewError(undefined)
    setReviewing(true)
    try {
      const res = await api.sessionResults(sessionId)
      setResults(res)
      setView('results')
    } catch (err: unknown) {
      setReviewing(false)
      setReviewError(
        err instanceof ApiError ? err.message : 'Could not load that exam',
      )
    }
  }, [])

  // The session itself survives a reload now — the server holds it and the
  // app resumes it on load — but the terminal scrollback does not, and the
  // clock keeps running while the page comes back. Still worth a prompt.
  // Only armed while a session is actually in progress.
  useEffect(() => {
    if (view !== 'exam') return
    const onBeforeUnload = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [view])

  const inExam = view === 'exam'

  return (
    <div className={`app ${inExam ? 'in-exam' : ''}`}>
      <a className="skip-link" href="#main">
        Skip to content
      </a>

      <header className="app-header">
        <button className="brand" onClick={handleRestart} aria-label="CKAD Simulator — home">
          <span className="logo" aria-hidden="true">
            ⎈
          </span>
          <span>CKAD Simulator</span>
        </button>
        <span className="tagline">Certified Kubernetes Application Developer practice</span>
        {inExam && <span className="exam-chip">Exam in progress</span>}
        <button
          className="btn btn-ghost btn-icon theme-toggle"
          onClick={toggleTheme}
          aria-label="Toggle color theme"
          title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
        >
          {theme === 'dark' ? '☀' : '☾'}
        </button>
      </header>

      <main className="app-main" id="main">
        {view === 'home' && booting && (
          <div className="booting" role="status">
            <span className="spinner" aria-hidden="true" />
            <span>Checking for an exam in progress…</span>
          </div>
        )}
        {view === 'home' && !booting && (
          <Home
            onStart={handleStart}
            starting={starting}
            startError={startError}
            history={history}
            onReview={handleReview}
            reviewError={reviewError}
          />
        )}
        {view === 'exam' && session && (
          <ExamView
            session={session}
            questions={questions}
            onFinish={handleFinish}
            finishing={finishing}
          />
        )}
        {view === 'results' && results && (
          <ResultsView results={results} onRestart={handleRestart} reviewing={reviewing} />
        )}
      </main>

      {!inExam && (
        <footer className="app-footer">
          <span className="muted">
            Open-source CKAD practice · graded against live cluster state
          </span>
        </footer>
      )}
    </div>
  )
}
