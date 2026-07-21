import { reactive } from 'vue'
import { useLocalStorage } from '@vueuse/core'

export interface Crumb {
  label: string
  to?: string
}

// Module singleton: the breadcrumb bar lives permanently in the app shell and
// reads this state, so pages just declare their trail. Reserving the bar's
// space avoids the layout shift caused by breadcrumbs appearing after data loads.
//
// `pinnable` is opted into per-view (setBreadcrumbs(..., { pinnable: true })) so
// the sticky toggle only shows where it makes sense; it resets on navigation.
const state = reactive<{ items: Crumb[]; pinnable: boolean }>({ items: [], pinnable: false })

// The pin preference persists across navigations/reloads — a user who pins the
// bar in one review expects it stuck in the next.
const sticky = useLocalStorage('breadcrumbs:sticky', false)

export function setBreadcrumbs(items: Crumb[], opts: { pinnable?: boolean } = {}) {
  state.items = items
  state.pinnable = opts.pinnable ?? false
}

export function toggleBreadcrumbSticky() {
  sticky.value = !sticky.value
}

export function useBreadcrumbs() {
  return { state, sticky, toggleSticky: toggleBreadcrumbSticky }
}
