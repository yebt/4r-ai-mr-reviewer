import { effectScope } from 'vue'
import { useMediaQuery } from '@vueuse/core'

let shared: ReturnType<typeof useMediaQuery> | undefined

// Single shared phone-breakpoint media query. A detached effect scope keeps the
// one matchMedia listener alive for the app's lifetime, so N cards reusing this
// don't each register (and dispose) their own.
export function useIsPhone() {
  if (!shared) {
    shared = effectScope(true).run(() => useMediaQuery('(max-width: 639px)'))
  }
  return shared!
}
