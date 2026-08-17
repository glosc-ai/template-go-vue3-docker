import { apiBaseURL, request } from './client'

/** The signed-in user as returned by GET /api/v1/auth/session. */
export interface SessionUser {
  id: number
  subject: string
  name: string
  nickname: string
  email: string
  phone: string
  avatar: string
  created_at: string
  updated_at: string
  last_login_at: string
}

interface DataResponse<T> {
  data: T
}

/**
 * Reads the current session. Throws APIError with status 401 when no one is
 * signed in, which callers treat as a normal "anonymous" outcome.
 */
export async function fetchSession(): Promise<SessionUser> {
  const response = await request<DataResponse<SessionUser>>('/auth/session')
  return response.data
}

/** Clears the local session and reports the provider's end-session URL. */
export async function logout(): Promise<{ sso_logout_url: string }> {
  const response = await request<DataResponse<{ sso_logout_url: string }>>('/auth/logout', {
    method: 'POST',
  })
  return response.data
}

/**
 * Starts the SSO login. This is a full-page navigation on purpose: the
 * authorization code flow is a browser redirect chain, not a fetch() call.
 */
export function startLogin(redirectTo: string = window.location.pathname + window.location.search) {
  const target = new URL(`${apiBaseURL}/auth/sso/login`, window.location.origin)
  if (redirectTo && redirectTo !== '/login') {
    target.searchParams.set('redirect_to', redirectTo)
  }
  window.location.assign(target.toString())
}
