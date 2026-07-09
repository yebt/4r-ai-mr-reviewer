import { reactive } from 'vue'

export interface Crumb {
  label: string
  to?: string
}

// Module singleton: the breadcrumb bar lives permanently in the app shell and
// reads this state, so pages just declare their trail. Reserving the bar's
// space avoids the layout shift caused by breadcrumbs appearing after data loads.
const state = reactive<{ items: Crumb[] }>({ items: [] })

export function setBreadcrumbs(items: Crumb[]) {
  state.items = items
}

export function useBreadcrumbs() {
  return state
}
