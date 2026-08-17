import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { SessionUser } from '@/api/auth'
import { fetchSession, logout as requestLogout, startLogin } from '@/api/auth'
import { APIError } from '@/api/client'
import { notifyError, notifySuccess } from '@/lib/message'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<SessionUser | null>(null)
  const loading = ref(false)
  /** False until the first session probe finishes, so guards can await it. */
  const resolved = ref(false)
  /** Set when SSO itself is not configured on the server. */
  const unavailable = ref(false)

  const isAuthenticated = computed(() => user.value !== null)
  const displayName = computed(() =>
    user.value?.nickname || user.value?.name || user.value?.email || '已登录用户',
  )

  let inflight: Promise<void> | null = null

  /**
   * Loads the session once and shares the promise, so several guards or
   * components mounting together do not each hit the API.
   */
  async function ensureLoaded(force = false): Promise<void> {
    if (resolved.value && !force) {
      return
    }
    if (inflight && !force) {
      return inflight
    }

    loading.value = true
    inflight = (async () => {
      try {
        user.value = await fetchSession()
        unavailable.value = false
      }
      catch (caught) {
        user.value = null
        // 401 simply means "not signed in"; anything else is worth surfacing.
        if (caught instanceof APIError && caught.code === 'sso_disabled') {
          unavailable.value = true
        }
        else if (caught instanceof APIError && caught.status !== 401) {
          notifyError(`无法确认登录状态：${caught.message}`)
        }
      }
      finally {
        loading.value = false
        resolved.value = true
        inflight = null
      }
    })()

    return inflight
  }

  function login(redirectTo?: string) {
    if (unavailable.value) {
      notifyError('服务端尚未配置单点登录，请设置 SSO_CLIENT_ID 后重试。')
      return
    }
    startLogin(redirectTo)
  }

  /**
   * Clears the application session. With endSSOSession the browser then
   * continues to the provider's end-session endpoint, which also signs the
   * user out of every other app sharing that SSO session.
   */
  async function logout(endSSOSession = false) {
    try {
      const { sso_logout_url: ssoLogoutURL } = await requestLogout()
      user.value = null
      if (endSSOSession && ssoLogoutURL) {
        window.location.assign(ssoLogoutURL)
        return
      }
      notifySuccess('已退出登录')
    }
    catch (caught) {
      notifyError(caught instanceof Error ? caught.message : '退出登录失败')
    }
  }

  return {
    user,
    loading,
    resolved,
    unavailable,
    isAuthenticated,
    displayName,
    ensureLoaded,
    login,
    logout,
  }
})
