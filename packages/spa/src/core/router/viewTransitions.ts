import type { Router } from 'vue-router'

// Minimal typing for the View Transitions API: it is still absent from the DOM
// lib in some TS targets and entirely absent at runtime in Firefox, older
// browsers, and the jsdom test environment. We never call it unguarded.
type ViewTransition = { finished?: Promise<unknown> }
type StartViewTransition = (callback: () => void | Promise<void>) => ViewTransition

function supportsViewTransitions(): boolean {
  return (
    typeof document !== 'undefined' &&
    typeof (document as Document & { startViewTransition?: StartViewTransition })
      .startViewTransition === 'function'
  )
}

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

// Wrap route-driven DOM updates in document.startViewTransition so navigations
// cross-fade (and named shared elements morph). The reactive DOM update happens
// AFTER the navigation resolves, so a naive beforeEach can't wrap it. We follow
// the documented vue-router pattern: beforeResolve opens the transition and
// pauses navigation until the old snapshot is captured, then afterEach/onError
// releases it so the browser captures the new snapshot and animates.
//
// Feature-detection and reduced-motion are re-checked per navigation: when the
// API is missing (jsdom, Firefox) or the user prefers reduced motion, the guard
// returns without touching navigation, so routing proceeds normally with no
// errors and every existing guard/test behaves exactly as before.
export function installViewTransitions(router: Router): void {
  // Releases the pending transition callback so the browser captures the new
  // snapshot and animates. Held between beforeResolve and afterEach/onError.
  let releaseTransition: (() => void) | undefined

  router.beforeResolve(() => {
    if (!supportsViewTransitions() || prefersReducedMotion()) return

    // Kept pending so the new snapshot is taken only after Vue flushes the DOM,
    // i.e. once afterEach (success) or onError (abort) releases it.
    const settled = new Promise<void>((resolve) => {
      releaseTransition = resolve
    })

    // Resolves the moment startViewTransition has captured the old snapshot,
    // which is when we let the navigation (and its DOM update) continue.
    let releaseNavigation: () => void = () => {}
    const ready = new Promise<void>((resolve) => {
      releaseNavigation = resolve
    })

    const start = (document as Document & { startViewTransition: StartViewTransition })
      .startViewTransition
    const transition = start(() => {
      releaseNavigation()
      return settled
    })
    // A skipped/interrupted transition rejects its promises; swallow so it never
    // surfaces as an unhandled rejection.
    transition.finished?.catch(() => {})

    return ready
  })

  const release = () => {
    releaseTransition?.()
    releaseTransition = undefined
  }
  router.afterEach(release)
  router.onError(release)
}
