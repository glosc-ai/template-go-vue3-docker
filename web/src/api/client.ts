const apiBaseURL = (import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')

interface ErrorPayload {
  error?: {
    code?: string
    message?: string
  }
}

export class APIError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (init.body) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(`${apiBaseURL}${path}`, { ...init, headers })
  if (response.status === 204) {
    return undefined as T
  }

  const payload = await response.json().catch(() => null) as ErrorPayload | T | null
  if (!response.ok) {
    const error = payload as ErrorPayload | null
    throw new APIError(
      response.status,
      error?.error?.code || 'request_failed',
      error?.error?.message || `Request failed with status ${response.status}`,
    )
  }
  return payload as T
}
