import { useEffect, useRef, useState } from 'react'

interface TimerProps {
  /** RFC3339 session start time. */
  startedAt: string
  /** Total session duration in nanoseconds (Go time.Duration). */
  durationNs: number
  /** Called once when the countdown reaches zero. */
  onExpire?: () => void
}

function format(totalSeconds: number): string {
  const s = Math.max(0, totalSeconds)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = Math.floor(s % 60)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${pad(h)}:${pad(m)}:${pad(sec)}`
}

/** Minute marks at which remaining time is announced to screen readers. */
const ANNOUNCE_AT = [60, 30, 15, 5, 1]

/**
 * Counts down to the session deadline. Renders HH:MM:SS with a depletion bar
 * and escalating urgency (calm → amber under 15 min → red under 5 min), and
 * announces the milestones above to assistive technology.
 */
export function Timer({ startedAt, durationNs, onExpire }: TimerProps) {
  const deadline = new Date(startedAt).getTime() + durationNs / 1e6
  const totalSeconds = durationNs / 1e9
  const [remaining, setRemaining] = useState(() => (deadline - Date.now()) / 1000)

  useEffect(() => {
    const id = setInterval(() => {
      const next = (deadline - Date.now()) / 1000
      setRemaining(next)
      if (next <= 0) {
        clearInterval(id)
        onExpire?.()
      }
    }, 1000)
    return () => clearInterval(id)
  }, [deadline, onExpire])

  // Announce each milestone once, as it is crossed.
  const announcedRef = useRef<number | null>(null)
  const [announcement, setAnnouncement] = useState('')
  useEffect(() => {
    const minutes = Math.ceil(Math.max(0, remaining) / 60)
    if (!ANNOUNCE_AT.includes(minutes) || announcedRef.current === minutes) return
    announcedRef.current = minutes
    setAnnouncement(`${minutes} minute${minutes === 1 ? '' : 's'} remaining`)
  }, [remaining])

  const state = remaining <= 300 ? 'low' : remaining <= 900 ? 'warn' : 'ok'
  const leftPct = totalSeconds > 0 ? Math.max(0, Math.min(100, (remaining / totalSeconds) * 100)) : 0

  return (
    <>
      <div className={`timer-wrap timer-${state}`} title="Time remaining in this session">
        <span className={`timer ${state === 'low' ? 'timer-low' : ''}`}>
          {format(remaining)}
        </span>
        <span className="timer-bar" aria-hidden="true">
          <span className="timer-bar-fill" style={{ width: `${leftPct}%` }} />
        </span>
      </div>
      <span className="sr-only" role="status" aria-live="polite">
        {announcement}
      </span>
    </>
  )
}
