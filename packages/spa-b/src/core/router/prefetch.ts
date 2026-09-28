/**
 * Warms the lazy chunks of the app's main top-level pages once the app has
 * mounted and its first navigation has settled — so a *later* click on
 * /runs, /reviews, or /settings doesn't pay the dynamic-import cost cold
 * (measured ~300ms on first visit to /runs in a prod build). Every page
 * route from `vue-router/auto-routes` is registered as `component: () =>
 * import(...)`, so `router.getRoutes()` exposes it as `components.default`,
 * a function returning the import() promise — this module finds those
 * functions for a fixed allowlist of paths and calls them.
 *
 * Deliberately NOT eager: runs one loader at a time via
 * `requestIdleCallback` (falling back to a 1.5s `setTimeout` where it's
 * unavailable, e.g. Safari or jsdom in tests), so it never competes with
 * real interaction work, and swallows any load error — a failed prefetch
 * just means the normal lazy import runs later, on real navigation.
 */
import type { Router, RouteRecordNormalized } from 'vue-router'

type LazyComponentLoader = () => Promise<unknown>

/**
 * Resolved `path` (not the file-based route name) of every top-level page
 * worth warming ahead of time — the pages a signed-in user is overwhelmingly
 * likely to visit next after the app boots.
 */
const PREFETCH_PATHS = ['/runs', '/runs/:id', '/reviews', '/reviews/:id', '/settings']

function isLazyComponentLoader(component: unknown): component is LazyComponentLoader {
  return typeof component === 'function'
}

/**
 * Pure selection logic (unit-tested): picks the lazy `component: () =>
 * import(...)` loaders to warm, in `PREFETCH_PATHS` order, skipping any path
 * that isn't registered or whose `default` view isn't lazy (e.g. already
 * eagerly imported, or missing in a given build).
 */
export function selectPrefetchLoaders(routes: RouteRecordNormalized[]): LazyComponentLoader[] {
  const routeByPath = new Map(routes.map((route) => [route.path, route]))
  const loaders: LazyComponentLoader[] = []
  for (const path of PREFETCH_PATHS) {
    const loader = routeByPath.get(path)?.components?.default
    if (isLazyComponentLoader(loader)) loaders.push(loader)
  }
  return loaders
}

function scheduleIdle(run: () => void): void {
  if (typeof requestIdleCallback === 'function') {
    requestIdleCallback(run)
  } else {
    setTimeout(run, 1500)
  }
}

/** Loads `loaders` one at a time, each scheduled on its own idle slice. */
function prefetchSequentially(loaders: LazyComponentLoader[], index = 0): void {
  const loader = loaders[index]
  if (!loader) return
  scheduleIdle(() => {
    loader()
      .catch(() => {
        // no-op — the route just lazy-loads normally on real navigation
      })
      .finally(() => {
        prefetchSequentially(loaders, index + 1)
      })
  })
}

/** Entry point — call once, after `router.isReady()`. See module doc above. */
export function prefetchRouteComponents(router: Router): void {
  prefetchSequentially(selectPrefetchLoaders(router.getRoutes()))
}
