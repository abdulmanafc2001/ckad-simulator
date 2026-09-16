import { useCallback, useEffect, useState } from 'react'
import './App.css'
import { api, ApiError } from './api/client'
import type { EndSessionResponse, Question, StartSessionResponse } from './api/types'
import { ExamView } from './components/ExamView'
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
      setView('results')
    } catch (err: unknown) {
      setStartError(err instanceof ApiError ? err.message : 'Could not finish the session')
    } finally {
      setFinishing(false)
    }
  }, [session])

  const handleRestart = useCallback(() => {
    setSession(null)
    setQuestions([])
    setResults(null)
    setView('home')
  }, [])

  // Leaving mid-exam loses the session, so the browser asks first. Only armed
  // while a session is actually in progress.
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
        {view === 'home' && (
          <Home onStart={handleStart} starting={starting} startError={startError} />
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
          <ResultsView results={results} onRestart={handleRestart} />
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
