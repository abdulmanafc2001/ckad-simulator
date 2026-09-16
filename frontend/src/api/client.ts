// A thin, typed wrapper around fetch for the CKAD Simulator API. All calls go
// through /api/v1, which Vite proxies to the Go backend during development.

import type {
  ClusterStatus,
  EndSessionResponse,
  ExecRequest,
  ExecResponse,
  Question,
  QuestionSummary,
  SessionsResponse,
  StartSessionRequest,
  StartSessionResponse,
  SubmitAnswerRequest,
  SubmitAnswerResponse,
} from './types'

const BASE = '/api/v1'

/** Error thrown for non-2xx responses, carrying the HTTP status. */
export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const body = (await res.json()) as { error?: string }
      if (body.error) message = body.error
    } catch {
      // Response had no JSON body; fall back to the status text.
    }
    throw new ApiError(res.status, message)
  }

  // 204 No Content and empty bodies are treated as undefined.
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

export const api = {
  /**
   * Reports whether the exam cluster is reachable. The endpoint answers 503
   * (not an error condition for us) when it is down, so this reads the body
   * either way instead of going through `request`.
   */
  async clusterStatus(): Promise<ClusterStatus> {
    try {
      const res = await fetch(`${BASE}/cluster/status`, {
        headers: { 'Content-Type': 'application/json' },
      })
      const body = (await res.json()) as Partial<ClusterStatus>
      return {
        connected: body.connected === true,
        detail: body.detail ?? '',
      }
    } catch {
      // The API itself is unreachable — report it the same way.
      return { connected: false, detail: 'Cannot reach the simulator API on :8080' }
    }
  },

  listQuestions(): Promise<{ questions: QuestionSummary[] }> {
    return request('/questions')
  },

  getQuestion(id: string): Promise<Question> {
    return request(`/questions/${encodeURIComponent(id)}`)
  },

  startSession(body: StartSessionRequest = {}): Promise<StartSessionResponse> {
    return request('/sessions', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },

  /**
   * The exam to resume (if one is in progress) plus the finished exams
   * available for review. Asked for once on load.
   */
  sessions(): Promise<SessionsResponse> {
    return request('/sessions')
  },

  /** Stored results of a finished exam — nothing is re-graded. */
  sessionResults(sessionId: string): Promise<EndSessionResponse> {
    return request(`/sessions/${encodeURIComponent(sessionId)}/results`)
  },

  submitAnswer(
    sessionId: string,
    body: SubmitAnswerRequest,
  ): Promise<SubmitAnswerResponse> {
    return request(`/sessions/${encodeURIComponent(sessionId)}/answers`, {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },

  endSession(sessionId: string): Promise<EndSessionResponse> {
    return request(`/sessions/${encodeURIComponent(sessionId)}/end`, {
      method: 'POST',
    })
  },

  exec(body: ExecRequest): Promise<ExecResponse> {
    return request('/cluster/exec', {
      method: 'POST',
      body: JSON.stringify(body),
    })
  },

  /** Loads a file from the exam sandbox (built-in vi/nano editors). */
  readFile(path: string): Promise<{ path: string; content: string }> {
    return request(`/files?path=${encodeURIComponent(path)}`)
  },

  /** Saves a file edited with the built-in vi/nano editors. */
  writeFile(path: string, content: string): Promise<{ ok: boolean; path: string }> {
    return request('/files', {
      method: 'POST',
      body: JSON.stringify({ path, content }),
    })
  },

  /** Tab-completion candidates for a terminal line (commands & paths). */
  complete(line: string): Promise<{ matches: string[] }> {
    return request(`/files/complete?line=${encodeURIComponent(line)}`)
  },
}
