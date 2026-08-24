import { useEffect, useState } from 'react'

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

/** Counts down to the session deadline and renders HH:MM:SS. */
export function Timer({ startedAt, durationNs, onExpire }: TimerProps) {
  const deadline = new Date(startedAt).getTime() + durationNs / 1e6
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

  const low = remaining <= 300 // last 5 minutes
  return (
    <span className={`timer ${low ? 'timer-low' : ''}`} title="Time remaining">
      {format(remaining)}
    </span>
  )
}
