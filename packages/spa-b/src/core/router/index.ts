import { createRouter, createWebHistory } from 'vue-router'
import { routes, handleHotUpdate } from 'vue-router/auto-routes'
import { onUnauthorized } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'
import LoginPage from '@modules/auth/pages/LoginPage.vue'

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
  const auth = useAuthStore()

  if (!auth.ready) {
    await auth.fetchStatus()
  }

  const isLoginRoute = to.meta.public === true

  if (!auth.enabled) {
    return isLoginRoute ? '/' : true
  }

  if (!auth.authenticated) {
    return isLoginRoute ? true : { path: '/login', query: { redirect: to.fullPath } }
  }

  return isLoginRoute ? '/' : true
})

onUnauthorized(() => {
  const current = router.currentRoute.value.fullPath
  router.push({ path: '/login', query: { redirect: current } })
})

export default router

if (import.meta.hot) {
  handleHotUpdate(router)
}
