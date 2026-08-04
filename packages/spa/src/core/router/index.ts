import { createRouter, createWebHistory } from 'vue-router'
import { routes, handleHotUpdate } from 'vue-router/auto-routes'
import NProgress from 'nprogress'
import { onUnauthorized } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'

// Slim top-of-page progress bar for navigations. The spinner is off (the bar
// alone reads as "loading") and the bar colour is themed to the accent token in
// main.css.
NProgress.configure({ showSpinner: false })

// Routes may opt out of auth gating (e.g. the login page) via `meta.public`.
declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    // Page name used to build the browser tab title ("<title> - AI Review").
    title?: string
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
  NProgress.start()
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

// Keep the browser tab title in sync with the active page's meta.title.
router.afterEach((to) => {
  NProgress.done()
  const t = to.meta.title as string | undefined
  document.title = t ? `${t} - AI Review` : 'AI Review'
})

// A failed/aborted navigation never reaches afterEach, so clear the bar here so
// it can never get stuck at the top of the page.
router.onError(() => NProgress.done())

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
