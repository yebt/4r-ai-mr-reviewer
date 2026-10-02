import { inject, provide, ref } from 'vue'
import type { InjectionKey, Ref } from 'vue'

/**
 * Injection key for the ancestor element that actually scrolls the app's
 * content. `AppShell` owns two structural `<main>` elements (desktop /
 * mobile, see AppShell.vue) and only ever mounts one of them at a time — a
 * window scrollbar never appears (the desktop wrapper is `h-dvh
 * overflow-hidden`). Anything that needs to know which element scrolls
 * (e.g. a `@tanstack/vue-virtual` virtualizer's `getScrollElement`) reads it
 * through this injection instead of querying the DOM or assuming
 * `window`/`document.scrollingElement`.
 */
export const scrollContainerKey: InjectionKey<Ref<HTMLElement | null>> = Symbol('scrollContainer')

/** Called once by `AppShell` to publish its live scroll element. */
export function provideScrollContainer(el: Ref<HTMLElement | null>): void {
  provide(scrollContainerKey, el)
}

/**
 * Reads the ancestor scroll element. Falls back to a local, always-null ref
 * when used outside `AppShell` (e.g. a component mounted standalone in a
 * unit test) so callers never have to guard against an unset injection.
 */
export function useScrollContainer(): Ref<HTMLElement | null> {
  return inject(scrollContainerKey, () => ref<HTMLElement | null>(null), true)
}
