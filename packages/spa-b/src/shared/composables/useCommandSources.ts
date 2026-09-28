/**
 * Extensible command-palette source registry (docs/ui-patterns.md #2): lets
 * feature modules contribute their own searchable groups (e.g. Flow's
 * "Repositories") to the ⌘K palette without the shared design-system
 * component importing anything from `@modules/*` — CommandPalette.vue only
 * ever reads this registry, never a specific module.
 *
 * The registry is a single module-level reactive Map, so every caller
 * (feature `commandSource.ts` files registering, CommandPalette.vue reading)
 * shares the same instance — no provide/inject or app-level plugin needed.
 */
import { reactive } from 'vue'
import { useFilter } from 'reka-ui'

export interface CommandItem {
  /** Stable unique id across every registered source — used as the
   * Combobox item value/key, so it must never collide with a nav item's
   * `to` or another source's item id. */
  id: string
  label: string
  hint?: string
  icon?: string
  /** Route path this item navigates to on selection. */
  to: string
  /** Extra terms matched in addition to `label`/`hint` (e.g. a repo's
   * remote URL), never shown in the UI. */
  keywords?: string[]
}

export interface CommandSource {
  id: string
  /** Group heading rendered above this source's items, same style as the
   * built-in "Navigate"/"Actions" groups. */
  heading: string
  /** Called fresh on every render so the source can reflect live data
   * (e.g. a Pinia Colada query) without the registry itself being reactive
   * to that data's internals. */
  items: () => CommandItem[]
}

const sources = reactive(new Map<string, CommandSource>())

/**
 * Registers a command source; returns an unregister function for cleanup
 * (e.g. from a composable's caller on unmount, though app-wide sources are
 * typically registered once for the lifetime of the app).
 */
export function registerCommandSource(source: CommandSource): () => void {
  sources.set(source.id, source)
  return () => {
    sources.delete(source.id)
  }
}

/** Read-only view of every registered source, in registration order. */
export function useCommandSources() {
  return sources
}

const { contains } = useFilter({ sensitivity: 'base' })

function matchesCommandItem(item: CommandItem, term: string): boolean {
  if (contains(item.label, term)) return true
  if (item.hint && contains(item.hint, term)) return true
  return (item.keywords ?? []).some((keyword) => contains(keyword, term))
}

/** While typing, cap each source's visible results to keep the list usable. */
const CAP_WITH_TERM = 8
/** With an empty query, show fewer per source so the built-in groups still
 * dominate the initial (unfiltered) view. */
const CAP_EMPTY_TERM = 5

/**
 * Locale-aware, case-insensitive filter + cap over one source's items —
 * matches against `label`, `hint`, and `keywords`. Empty/whitespace-only
 * queries return the first `CAP_EMPTY_TERM` items unfiltered; a real query
 * returns up to `CAP_WITH_TERM` matches. Pure and framework-agnostic so it
 * can be unit-tested without mounting CommandPalette.vue.
 */
export function filterCommandItems(items: CommandItem[], query: string): CommandItem[] {
  const term = query.trim()
  if (!term) return items.slice(0, CAP_EMPTY_TERM)
  return items.filter((item) => matchesCommandItem(item, term)).slice(0, CAP_WITH_TERM)
}
