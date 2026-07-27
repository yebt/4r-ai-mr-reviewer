import type { PreflightCheck } from '@shared/api/types'

// Visual mapping for a preflight check status. Each status carries its own icon,
// token color class, and a short screen-reader label so the state is conveyed by
// text/icon and never by color alone.
export interface CheckStatusUi {
  icon: string
  class: string
  srLabel: string
}

export const checkStatusUi: Record<PreflightCheck['status'], CheckStatusUi> = {
  ok: { icon: 'i-lucide-check', class: 'text-ok', srLabel: 'Allowed' },
  fail: { icon: 'i-lucide-x', class: 'text-danger', srLabel: 'Blocked' },
  unknown: { icon: 'i-lucide-triangle-alert', class: 'text-warn', srLabel: 'Unknown' },
}
