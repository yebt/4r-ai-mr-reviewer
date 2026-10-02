import { ref } from 'vue'
import { defineStore } from 'pinia'
import { onUnauthorized } from '@shared/api/client'
import { authStatus, login as loginRequest, logout as logoutRequest } from './api'

export const useAuthStore = defineStore('auth', () => {
  const enabled = ref(false)
  const authenticated = ref(false)
  const ready = ref(false)

  // Any 401 on a non-/auth/ path means the session died server-side.
  onUnauthorized(() => {
    authenticated.value = false
  })

  async function fetchStatus(): Promise<void> {
    try {
      const status = await authStatus()
      enabled.value = status.authEnabled
      authenticated.value = status.authenticated
    } catch {
      // Network/backend failure: fail OPEN so a broken auth backend never
      // locks the whole app out.
      enabled.value = false
    } finally {
      ready.value = true
    }
  }

  async function login(password: string): Promise<void> {
    const result = await loginRequest(password)
    authenticated.value = result.authenticated
    enabled.value = true
  }

  async function logout(): Promise<void> {
    try {
      await logoutRequest()
    } finally {
      authenticated.value = false
    }
  }

  return { enabled, authenticated, ready, fetchStatus, login, logout }
})
