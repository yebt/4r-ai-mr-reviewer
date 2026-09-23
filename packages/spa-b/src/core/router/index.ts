import NProgress from 'nprogress'
import { createRouter, createWebHistory } from 'vue-router'
import { routes, handleHotUpdate } from 'vue-router/auto-routes'
import { onUnauthorized } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'
import LoginPage from '@modules/auth/pages/LoginPage.vue'
import { formatDocumentTitle } from './documentTitle'

NProgress.configure({ showSpinner: false })

declare module 'vue-router' {
  interface RouteMeta {
    /** Routes reachable without an authenticated session (e.g. /login). */
    public?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// Registered manually (not via src/pages) — the real login UI is a separate
// task and is expected to replace/extend LoginPage.vue in place, not this
// route registration.
router.addRoute({
  path: '/login',
  name: 'login',
  component: LoginPage,
  meta: { public: true },
})

router.beforeEach(async (to) => {
  NProgress.start()

  // /design is the dev-only token/component preview (src/pages/design.vue) —
  // never a live production route.
  if (to.path === '/design' && !import.meta.env.DEV) return '/'

  const auth = useAuthStore()

  if (!auth.ready) {
    await auth.fetchStatus()
  }

  // `isLoginRoute` identifies /login specifically (for the "already signed
  // in, bounce away from /login" cases below); `isPublicRoute` is the
  // broader "skip the auth redirect" check, which also covers other public
  // routes like the 404 catch-all (src/pages/[...path].vue) that must stay
  // reachable without auth but must NOT bounce an authenticated visitor
  // back to '/' the way /login does.
  // Compared by path, not name: /login is registered imperatively (below), so
  // its name isn't in the typed-router's generated RouteNamedMap union.
  const isLoginRoute = to.path === '/login'
  const isPublicRoute = to.meta.public === true

  if (!auth.enabled) {
    return isLoginRoute ? '/' : true
  }

  if (!auth.authenticated) {
    return isPublicRoute ? true : { path: '/login', query: { redirect: to.fullPath } }
  }

  return isLoginRoute ? '/' : true
})

router.afterEach(() => {
  NProgress.done()
})

router.afterEach((to) => {
  document.title = formatDocumentTitle(to.meta.title)
})

router.onError(() => {
  NProgress.done()
})

onUnauthorized(() => {
  const current = router.currentRoute.value.fullPath
  router.push({ path: '/login', query: { redirect: current } })
})

export default router

if (import.meta.hot) {
  handleHotUpdate(router)
}
