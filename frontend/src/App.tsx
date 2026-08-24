import { useCallback, useState } from 'react'
import './App.css'
import { api, ApiError } from './api/client'
import type { EndSessionResponse, Question, StartSessionResponse } from './api/types'
import { ExamView } from './components/ExamView'
import { Home } from './components/Home'
import { ResultsView } from './components/ResultsView'

type View = 'home' | 'exam' | 'results'

export default function App() {
  const [view, setView] = useState<View>('home')
  const [session, setSession] = useState<StartSessionResponse | null>(null)
  const [questions, setQuestions] = useState<Question[]>([])
  const [results, setResults] = useState<EndSessionResponse | null>(null)

  const [starting, setStarting] = useState(false)
  const [startError, setStartError] = useState<string | undefined>()
  const [finishing, setFinishing] = useState(false)

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

  return (
    <div className="app">
      <header className="app-header">
        <div className="brand" onClick={handleRestart} role="button" tabIndex={0}>
          <span className="logo">⎈</span>
          <span>CKAD Simulator</span>
        </div>
        <span className="tagline">Certified Kubernetes Application Developer practice</span>
      </header>

      <main className="app-main">
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

      <footer className="app-footer">
        <span className="muted">Baseline build · in-memory data · heuristic grading</span>
      </footer>
    </div>
  )
}
