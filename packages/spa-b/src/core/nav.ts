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
  /** kebab-case lucide-vue-next icon name, rendered via the Icon atom. */
  icon: string
  /** Route path this item links to. */
  to: string
  /**
   * `primary` items appear in the desktop sidebar and as mobile bottom tabs.
   * `secondary` items appear in the desktop sidebar and the mobile "More"
   * drawer only — kept off the bottom tab bar per the 3–5 tab guidance.
   */
  section: NavSection
  /**
   * Reachable only via the command palette and the mobile "More" drawer —
   * kept out of the persistent desktop sidebar's flat list. Used by the
   * settings sub-pages (src/pages/settings/*.vue), which already have a
   * dedicated entry point (the Settings menu, /settings) and would just be
   * redundant clutter if also listed inline in the sidebar.
   */
  deepLinkOnly?: boolean
}

export const navItems: NavItem[] = [
  { label: 'Home', icon: 'house', to: '/', section: 'primary' },
  { label: 'Flow', icon: 'workflow', to: '/flow', section: 'primary' },
  { label: 'Reviews', icon: 'git-pull-request', to: '/reviews', section: 'primary' },
  { label: 'Runs', icon: 'activity', to: '/runs', section: 'primary' },
  { label: 'Settings', icon: 'settings', to: '/settings', section: 'secondary' },
  // The dedicated settings sub-pages (src/pages/settings/*.vue) — listed
  // here too so they're deep-linkable from the command palette and the
  // mobile "More" drawer, not just from the Settings menu itself.
  { label: 'Providers', icon: 'cable', to: '/settings/providers', section: 'secondary', deepLinkOnly: true },
  { label: 'Accounts', icon: 'user', to: '/settings/accounts', section: 'secondary', deepLinkOnly: true },
  { label: 'Telegram', icon: 'send', to: '/settings/telegram', section: 'secondary', deepLinkOnly: true },
  { label: 'Profiles', icon: 'users', to: '/settings/profiles', section: 'secondary', deepLinkOnly: true },
  { label: 'Repos', icon: 'git-branch', to: '/settings/repos', section: 'secondary', deepLinkOnly: true },
  {
    label: 'Notifications & Security',
    icon: 'shield',
    to: '/settings/notifications-security',
    section: 'secondary',
    deepLinkOnly: true,
  },
]

export const primaryNavItems: NavItem[] = navItems.filter((item) => item.section === 'primary')
export const secondaryNavItems: NavItem[] = navItems.filter((item) => item.section === 'secondary')
/** Every item the persistent desktop sidebar renders inline — primary
 * destinations plus top-level secondary entry points, excluding
 * `deepLinkOnly` items (see AppSidebar.vue). */
export const sidebarNavItems: NavItem[] = navItems.filter((item) => !item.deepLinkOnly)

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
