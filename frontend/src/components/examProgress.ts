// The backend restores a session itself — its tasks, its answers and its
// original deadline — after a reload or a restart. What it does not know is
// the candidate's own annotation of that exam: which questions they flagged,
// which they have opened, how many hints they burned and where they were.
// That lives in the browser, keyed by session id so a new exam starts clean.

const PROGRESS_PREFIX = 'ckad-progress-'

function progressKey(sessionId: string): string {
  return `${PROGRESS_PREFIX}${sessionId}`
}

/** Progress marks restored alongside a resumed exam. */
export interface ExamProgress {
  index: number
  hintsShown: Record<string, number>
  visited: Record<string, boolean>
  flagged: Record<string, boolean>
}

const EMPTY: ExamProgress = { index: 0, hintsShown: {}, visited: {}, flagged: {} }

export function loadExamProgress(sessionId: string): ExamProgress {
  try {
    const raw = localStorage.getItem(progressKey(sessionId))
    if (!raw) return EMPTY
    const parsed = JSON.parse(raw) as Partial<ExamProgress>
    return {
      index: typeof parsed.index === 'number' ? parsed.index : 0,
      hintsShown: parsed.hintsShown ?? {},
      visited: parsed.visited ?? {},
      flagged: parsed.flagged ?? {},
    }
  } catch {
    // Unreadable or corrupt — start the annotations over rather than fail.
    return EMPTY
  }
}

export function saveExamProgress(sessionId: string, progress: ExamProgress): void {
  try {
    localStorage.setItem(progressKey(sessionId), JSON.stringify(progress))
  } catch {
    // Storage full or blocked (private windows). The exam itself is on the
    // server, so losing the annotations is not worth an error.
  }
}

/**
 * Drops the progress marks of every session except the one still running.
 * Finished and abandoned exams would otherwise leave a key behind for good.
 */
export function pruneExamProgress(activeSessionId: string | null): void {
  try {
    const keep = activeSessionId ? progressKey(activeSessionId) : null
    for (const key of Object.keys(localStorage)) {
      if (key.startsWith(PROGRESS_PREFIX) && key !== keep) localStorage.removeItem(key)
    }
  } catch {
    // Storage unavailable — nothing to prune.
  }
}
