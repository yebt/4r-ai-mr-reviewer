import { createRouter, createWebHistory } from 'vue-router'
import { routes, handleHotUpdate } from 'vue-router/auto-routes'
import { onUnauthorized } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'

// Routes may opt out of auth gating (e.g. the login page) via `meta.public`.
declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// Global auth guard. On the first navigation it resolves /auth/status once, then:
//   - auth disabled  -> never gate (login page redirects home, all else allowed).
//   - enabled & not authenticated & leaving login -> send to /login?redirect=<path>.
//   - authenticated (or disabled) & heading to /login -> send home.
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.ready) await auth.fetchStatus()

  if (!auth.enabled) {
    // Auth is off: the login page is meaningless, everything else is open.
    return to.path === '/login' ? { path: '/' } : true
  }

  if (!auth.authenticated && to.path !== '/login') {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (auth.authenticated && to.path === '/login') {
    return { path: '/' }
  }
  return true
})

// Session expired mid-session: an authenticated API call returned 401. The auth
// store already flipped `authenticated` to false; here we bounce to the login
// page (unless already there), preserving where the user was.
onUnauthorized(() => {
  const current = router.currentRoute.value
  if (current.path === '/login') return
  void router.push({ path: '/login', query: { redirect: current.fullPath } })
})

export default router

if (import.meta.hot) {
  handleHotUpdate(router)
}
