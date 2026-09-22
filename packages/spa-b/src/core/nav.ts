/**
 * Central nav model — the single source of truth for app destinations.
 * Consumed by AppSidebar (desktop), AppBottomNav (mobile), and
 * CommandPalette (⌘K) so navigation never drifts between the three
 * surfaces. See docs/ui-patterns.md #1 and #8 for the responsive rationale.
 */
import { useFilter } from 'reka-ui'

declare module 'vue-router' {
  interface RouteMeta {
    /** Optional per-route title override for AppHeader; falls back to the
     * matching nav item's label, then a generic default. */
    title?: string
  }
}

export type NavSection = 'primary' | 'secondary'

export interface NavItem {
  /** Human-readable destination name, shown in every nav surface. */
  label: string
  /** Lucide icon name (`lucide:*`), rendered via the Icon atom. */
  icon: string
  /** Route path this item links to. */
  to: string
  /**
   * `primary` items appear in the desktop sidebar and as mobile bottom tabs.
   * `secondary` items appear in the desktop sidebar and the mobile "More"
   * drawer only — kept off the bottom tab bar per the 3–5 tab guidance.
   */
  section: NavSection
}

export const navItems: NavItem[] = [
  { label: 'Home', icon: 'lucide:home', to: '/', section: 'primary' },
  { label: 'Reviews', icon: 'lucide:git-pull-request', to: '/reviews', section: 'primary' },
  { label: 'Runs', icon: 'lucide:activity', to: '/runs', section: 'primary' },
  { label: 'Settings', icon: 'lucide:settings', to: '/settings', section: 'secondary' },
]

export const primaryNavItems: NavItem[] = navItems.filter((item) => item.section === 'primary')
export const secondaryNavItems: NavItem[] = navItems.filter((item) => item.section === 'secondary')

/**
 * Active-state helper shared by every nav surface: exact match for the root
 * route, prefix match for everything else so a nested route (e.g.
 * `/reviews/123`) still highlights its top-level destination.
 */
export function isNavItemActive(item: NavItem, currentPath: string): boolean {
  if (item.to === '/') return currentPath === '/'
  return currentPath === item.to || currentPath.startsWith(`${item.to}/`)
}

const { contains } = useFilter({ sensitivity: 'base' })

/**
 * Locale-aware, case-insensitive filter over the nav model — used by the
 * command palette's "Navigate" group. Empty/whitespace-only queries return
 * every item unchanged (no query = show everything).
 */
export function filterNavItems(items: NavItem[], query: string): NavItem[] {
  const term = query.trim()
  if (!term) return items
  return items.filter((item) => contains(item.label, term))
}
