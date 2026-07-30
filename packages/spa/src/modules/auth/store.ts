import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, onUnauthorized } from '@shared/api/client'

// Optional password auth. Mirrors the backend contract:
//   - enabled: the server has AIR_AUTH_PASSWORD set (from GET /auth/status).
//   - authenticated: the current session cookie is valid.
//   - ready: /auth/status has resolved at least once, so the guard can decide.
// When disabled, the app behaves exactly as before: nothing is ever gated.
export const useAuthStore = defineStore('auth', () => {
  const enabled = ref(false)
  const authenticated = ref(false)
  const ready = ref(false)

  // The session expired or is missing mid-session: any authenticated API call
  // came back 401. Only flip state here; the router hook handles navigation.
  onUnauthorized(() => {
    authenticated.value = false
  })

  // Resolve auth state from the server. Never gates on its own failure:
  // if /auth/status is unreachable we fail open (treat auth as disabled) so a
  // transient error can't brick the app — protected routes still return 401
  // server-side, which re-triggers the login flow.
  async function fetchStatus() {
    try {
      const status = await api.authStatus()
      enabled.value = status.authEnabled
      authenticated.value = status.authenticated
    } catch {
      enabled.value = false
      authenticated.value = false
    } finally {
      ready.value = true
    }
  }

  // Attempt a login. Throws ApiError on 401 (invalid password) / 429 (rate
  // limited) so the login view can show the right message.
  async function login(password: string) {
    const res = await api.login(password)
    authenticated.value = res.authenticated
    enabled.value = true
    return res
  }

  async function logout() {
    try {
      await api.logout()
    } finally {
      authenticated.value = false
    }
  }

  return { enabled, authenticated, ready, fetchStatus, login, logout }
})
