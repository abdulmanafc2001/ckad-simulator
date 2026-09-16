import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { Question, StartSessionResponse } from '../api/types'
import { DifficultyBadge, DomainBadge, WeightBadge } from './Badges'
import { CopyableText } from './Copyable'
import { loadExamProgress, saveExamProgress } from './examProgress'
import { Modal } from './Modal'
import { Timer } from './Timer'
import { ExamTerminal } from './Terminal'

interface ExamViewProps {
  session: StartSessionResponse
  questions: Question[]
  onFinish: () => void
  finishing: boolean
}

/** Persisted width of the question column, as a percentage of the split. */
const SPLIT_KEY = 'ckad-split'
const SPLIT_MIN = 22
const SPLIT_MAX = 70

function initialSplit(): number {
  const n = Number(localStorage.getItem(SPLIT_KEY))
  return Number.isFinite(n) && n >= SPLIT_MIN && n <= SPLIT_MAX ? n : 38
}

export function ExamView({ session, questions, onFinish, finishing }: ExamViewProps) {
  const [restored] = useState(() => loadExamProgress(session.id))
  // Clamp: the bank could have changed under a resumed session.
  const [index, setIndex] = useState(() =>
    Math.min(Math.max(restored.index, 0), Math.max(questions.length - 1, 0)),
  )
  const [hintsShown, setHintsShown] = useState<Record<string, number>>(restored.hintsShown)
  const [visited, setVisited] = useState<Record<string, boolean>>(restored.visited)
  const [flagged, setFlagged] = useState<Record<string, boolean>>(restored.flagged)
  const [confirmFinish, setConfirmFinish] = useState(false)
  const [showShortcuts, setShowShortcuts] = useState(false)
  const [termMax, setTermMax] = useState(false)
  const [split, setSplit] = useState(initialSplit)

  const question = questions[index]
  const panelRef = useRef<HTMLElement>(null)
  const columnsRef = useRef<HTMLDivElement>(null)

  // Mark each question as visited the first time it is shown, so the
  // navigator and progress bar can reflect what the candidate has opened.
  useEffect(() => {
    setVisited((prev) => (prev[question.id] ? prev : { ...prev, [question.id]: true }))
  }, [question.id])

  // Save the progress marks so a resumed exam comes back with the same
  // flags, hints and current question, not just the same tasks and timer.
  useEffect(() => {
    saveExamProgress(session.id, { index, hintsShown, visited, flagged })
  }, [session.id, index, hintsShown, visited, flagged])

  // Changing question must start the reader at the top of the new task, not
  // wherever the previous (possibly long) one was scrolled to.
  useEffect(() => {
    panelRef.current?.scrollTo({ top: 0 })
  }, [index])

  const shown = hintsShown[question.id] ?? 0

  const visitedCount = Object.keys(visited).length
  const progressPct = useMemo(
    () => Math.round((visitedCount / questions.length) * 100),
    [visitedCount, questions.length],
  )

  const revealHint = useCallback(() => {
    setHintsShown((prev) => ({
      ...prev,
      [question.id]: Math.min((prev[question.id] ?? 0) + 1, question.hints.length),
    }))
  }, [question.id, question.hints.length])

  function statusOf(qid: string): 'done' | 'unanswered' {
    return visited[qid] ? 'done' : 'unanswered'
  }

  const isFlagged = Boolean(flagged[question.id])
  const flaggedCount = Object.values(flagged).filter(Boolean).length

  const toggleFlag = useCallback(() => {
    setFlagged((prev) => ({ ...prev, [question.id]: !prev[question.id] }))
  }, [question.id])

  const go = useCallback(
    (delta: number) =>
      setIndex((i) => Math.max(0, Math.min(questions.length - 1, i + delta))),
    [questions.length],
  )

  // Keyboard shortcuts. All are Alt-based so they never collide with what the
  // candidate is typing in the terminal or in the emulated vi/nano editors.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!e.altKey || e.ctrlKey || e.metaKey) return
      switch (e.key) {
        case 'ArrowLeft':
          e.preventDefault()
          go(-1)
          break
        case 'ArrowRight':
          e.preventDefault()
          go(1)
          break
        case 'f':
        case 'F':
          e.preventDefault()
          toggleFlag()
          break
        case 'h':
        case 'H':
          e.preventDefault()
          revealHint()
          break
        case 'm':
        case 'M':
          e.preventDefault()
          setTermMax((m) => !m)
          break
        case '/':
        case '?':
          e.preventDefault()
          setShowShortcuts((s) => !s)
          break
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [go, toggleFlag, revealHint])

  // Drag-to-resize between the task and the terminal. Pointer capture keeps
  // the drag alive even when the cursor outruns the 10px handle.
  const onSplitterDown = (e: React.PointerEvent<HTMLDivElement>) => {
    e.currentTarget.setPointerCapture(e.pointerId)
    const move = (ev: PointerEvent) => {
      const box = columnsRef.current?.getBoundingClientRect()
      if (!box || box.width === 0) return
      const pct = ((ev.clientX - box.left) / box.width) * 100
      setSplit(Math.max(SPLIT_MIN, Math.min(SPLIT_MAX, pct)))
    }
    const up = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', up)
      document.body.classList.remove('is-resizing')
    }
    document.body.classList.add('is-resizing')
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', up)
  }

  useEffect(() => {
    localStorage.setItem(SPLIT_KEY, String(Math.round(split)))
  }, [split])

  // Keyboard-resizable splitter, per the WAI-ARIA separator pattern.
  const onSplitterKey = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowLeft') setSplit((s) => Math.max(SPLIT_MIN, s - 2))
    else if (e.key === 'ArrowRight') setSplit((s) => Math.min(SPLIT_MAX, s + 2))
    else return
    e.preventDefault()
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
          <div
            className="progress-track"
            role="progressbar"
            aria-valuenow={progressPct}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label="Questions viewed"
          >
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
          <button
            className="btn btn-ghost btn-icon"
            onClick={() => setShowShortcuts(true)}
            aria-label="Keyboard shortcuts"
            title="Keyboard shortcuts (Alt+/)"
          >
            {/* Inline SVG rather than ⌨ — the emoji keyboard glyph is missing
                from many default font stacks and renders as a tofu box. */}
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <rect
                x="2"
                y="6"
                width="20"
                height="12"
                rx="2"
                stroke="currentColor"
                strokeWidth="1.6"
              />
              <path
                d="M6 10h.01M9.5 10h.01M13 10h.01M16.5 10h.01M7.5 14h9"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
              />
            </svg>
          </button>
          <button
            className="btn btn-danger"
            onClick={() => setConfirmFinish(true)}
            disabled={finishing}
          >
            {finishing ? 'Grading…' : 'Finish exam'}
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
              aria-current={i === index ? 'true' : undefined}
              aria-label={`Question ${i + 1}: ${q.title}${visited[q.id] ? ', viewed' : ''}${
                flagged[q.id] ? ', flagged' : ''
              }`}
              title={optionLabel(q, i)}
            >
              {i + 1}
            </button>
          ))}
        </nav>

        <div
          className={`exam-columns ${termMax ? 'is-max' : ''}`}
          ref={columnsRef}
          style={
            termMax
              ? undefined
              : { gridTemplateColumns: `minmax(0, ${split}fr) 10px minmax(0, ${100 - split}fr)` }
          }
        >
          <article className="q-panel" ref={panelRef}>
            <header className="q-header">
              <div className="q-header-top">
                <div className="badges">
                  <DomainBadge domain={question.domain} />
                  <DifficultyBadge difficulty={question.difficulty} />
                  <WeightBadge weight={question.weight} />
                </div>
                <button
                  className={`btn btn-sm btn-flag ${isFlagged ? 'is-flagged' : ''}`}
                  onClick={toggleFlag}
                  aria-pressed={isFlagged}
                  title="Flag this question to review it later (Alt+F)"
                >
                  {isFlagged ? '⚑ Flagged for review' : '⚐ Flag for review'}
                </button>
              </div>
              <h3>
                Question {index + 1}. {question.title}
              </h3>
              <p className="muted">{question.description}</p>
            </header>

            <section className="q-task">
              <h4>Task</h4>
              <div className="q-task-text">
                <CopyableText text={question.task} />
              </div>
              {(question.prepare?.length ?? 0) > 0 && (
                <details className="q-setup">
                  <summary>Environment setup ({question.prepare.length})</summary>
                  <pre>
                    {question.prepare
                      .map((s) =>
                        s.file
                          ? `# file provided at ${s.file}\n${s.fileContent ?? ''}`
                          : s.command || s.yaml || s.name,
                      )
                      .join('\n')}
                  </pre>
                </details>
              )}
            </section>

            <section className="q-hints">
              <div className="q-hints-head">
                <h4>Hints</h4>
                {shown < question.hints.length && (
                  <button className="btn btn-ghost btn-sm" onClick={revealHint}>
                    Reveal hint ({shown}/{question.hints.length})
                  </button>
                )}
              </div>
              {shown === 0 ? (
                <p className="muted">
                  No hints revealed. Using hints does not reduce your score in this baseline.
                </p>
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
              <button className="btn" onClick={() => go(-1)} disabled={index === 0}>
                ← Previous
              </button>
              <span className="muted q-footer-count">
                {index + 1} of {questions.length}
              </span>
              <button
                className="btn"
                onClick={() => go(1)}
                disabled={index === questions.length - 1}
              >
                Next →
              </button>
            </footer>
          </article>

          <div
            className="splitter"
            role="separator"
            aria-orientation="vertical"
            aria-label="Resize the task and terminal panes"
            aria-valuenow={Math.round(split)}
            aria-valuemin={SPLIT_MIN}
            aria-valuemax={SPLIT_MAX}
            tabIndex={0}
            onPointerDown={onSplitterDown}
            onKeyDown={onSplitterKey}
            onDoubleClick={() => setSplit(38)}
          />

          <aside className="term-panel">
            <div className="term-head">
              <h4>Terminal</h4>
              <span className="term-hint muted">
                real <code>kubectl</code> · graded from live cluster state
              </span>
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setTermMax((m) => !m)}
                aria-pressed={termMax}
                title="Maximize the terminal (Alt+M)"
              >
                {termMax ? '⤡ Restore' : '⤢ Maximize'}
              </button>
            </div>
            <ExamTerminal />
          </aside>
        </div>
      </div>

      {confirmFinish && (
        <Modal
          title="Finish the exam?"
          onClose={() => setConfirmFinish(false)}
          actions={
            <>
              <button className="btn" onClick={() => setConfirmFinish(false)}>
                Keep working
              </button>
              <button
                className="btn btn-danger btn-solid"
                onClick={() => {
                  setConfirmFinish(false)
                  onFinish()
                }}
              >
                Finish &amp; grade
              </button>
            </>
          }
        >
          <p>
            All {questions.length} questions will be graded against the live cluster right
            now. You cannot return to the exam afterwards.
          </p>
          <ul className="confirm-facts">
            <li>
              <strong>{visitedCount}</strong> of {questions.length} questions opened
            </li>
            {flaggedCount > 0 && (
              <li>
                <strong>{flaggedCount}</strong> still flagged for review
              </li>
            )}
          </ul>
        </Modal>
      )}

      {showShortcuts && (
        <Modal
          title="Keyboard shortcuts"
          onClose={() => setShowShortcuts(false)}
          actions={
            <button className="btn" onClick={() => setShowShortcuts(false)}>
              Close
            </button>
          }
        >
          <dl className="shortcuts">
            <dt><kbd>Alt</kbd> + <kbd>←</kbd> / <kbd>→</kbd></dt>
            <dd>Previous / next question</dd>
            <dt><kbd>Alt</kbd> + <kbd>F</kbd></dt>
            <dd>Flag the current question for review</dd>
            <dt><kbd>Alt</kbd> + <kbd>H</kbd></dt>
            <dd>Reveal the next hint</dd>
            <dt><kbd>Alt</kbd> + <kbd>M</kbd></dt>
            <dd>Maximize / restore the terminal</dd>
            <dt><kbd>Alt</kbd> + <kbd>/</kbd></dt>
            <dd>Open this list</dd>
            <dt><kbd>Ctrl</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> / <kbd>V</kbd></dt>
            <dd>Copy / paste inside the terminal</dd>
          </dl>
          <p className="muted">
            Inside the terminal, <kbd>Tab</kbd> completes, <kbd>↑</kbd>/<kbd>↓</kbd> walk
            history, and <code>vi</code> / <code>nano</code> open the built-in editor.
          </p>
        </Modal>
      )}

      {finishing && (
        <div className="grading-overlay" role="status" aria-live="polite">
          <div className="grading-card">
            <span className="spinner" aria-hidden="true" />
            <h3>Grading your session</h3>
            <p className="muted">
              Running every check against the live cluster for all {questions.length}{' '}
              questions. This takes a minute or two — keep this tab open.
            </p>
          </div>
        </div>
      )}
    </div>
  )
}
