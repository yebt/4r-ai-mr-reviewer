/**
 * Generic cursor-based infinite-list accumulator. Framework-pure (no store
 * or component imports) so the accumulation/merge logic is unit-testable
 * without mounting anything — see `useInfiniteList.spec.ts`.
 *
 * Consumers supply a `fetchPage(cursor)` function (e.g. a thin wrapper
 * around `requestPage` from `@shared/api/client`) that resolves one page:
 * `{ items, nextCursor }`, keyset-paginated, newest-first. This composable
 * owns accumulation, `hasMore`, loading flags, and the polling-friendly
 * `refreshFirstPage()` merge — it never fetches on its own; callers drive
 * `loadInitial()`/`loadMore()`/`refreshFirstPage()` explicitly.
 */
import { computed, ref, type Ref } from 'vue'

export interface InfiniteListPage<T> {
  items: T[]
  nextCursor: string | null
}

export type FetchPage<T> = (cursor: string | null) => Promise<InfiniteListPage<T>>

export interface UseInfiniteListOptions<T> {
  /** Identity extractor used by `refreshFirstPage`'s merge-by-id. Defaults to `item.id`. */
  getId?: (item: T) => string
}

export interface UseInfiniteList<T> {
  items: Ref<T[]>
  isLoading: Ref<boolean>
  isLoadingMore: Ref<boolean>
  hasMore: Ref<boolean>
  error: Ref<unknown>
  loadInitial: () => Promise<void>
  loadMore: () => Promise<void>
  reset: () => void
  /** Merges a freshly-fetched first page into `items` by id (exposed mainly for tests). */
  mergeFirstPage: (pageItems: T[]) => void
  refreshFirstPage: () => Promise<void>
}

function defaultGetId<T>(item: T): string {
  return (item as { id: string }).id
}

export function useInfiniteList<T>(
  fetchPage: FetchPage<T>,
  opts: UseInfiniteListOptions<T> = {},
): UseInfiniteList<T> {
  const getId = opts.getId ?? defaultGetId<T>

  const items = ref<T[]>([]) as Ref<T[]>
  const cursor = ref<string | null>(null)
  const hasLoadedOnce = ref(false)

  const isLoading = ref(false)
  const isLoadingMore = ref(false)
  const isRefreshing = ref(false)
  const error = ref<unknown>(null)

  // Present only once an initial page has been fetched and it carried a
  // next cursor — before the first load this is false, which correctly
  // keeps a not-yet-loaded list from offering a "load more" affordance.
  const hasMore = computed(() => cursor.value !== null)

  async function loadInitial(): Promise<void> {
    if (isLoading.value) return
    isLoading.value = true
    error.value = null
    try {
      const page = await fetchPage(null)
      items.value = page.items
      cursor.value = page.nextCursor
      hasLoadedOnce.value = true
    } catch (err) {
      error.value = err
    } finally {
      isLoading.value = false
    }
  }

  async function loadMore(): Promise<void> {
    if (isLoading.value || isLoadingMore.value || !hasMore.value) return
    isLoadingMore.value = true
    error.value = null
    try {
      const page = await fetchPage(cursor.value)
      items.value = [...items.value, ...page.items]
      cursor.value = page.nextCursor
    } catch (err) {
      error.value = err
    } finally {
      isLoadingMore.value = false
    }
  }

  function reset(): void {
    items.value = []
    cursor.value = null
    error.value = null
    hasLoadedOnce.value = false
  }

  /**
   * Merges a freshly-fetched first page into the accumulated `items` by id:
   * items already present keep their position and get their fields replaced
   * (update in place — a status flip on an active run doesn't jump rows or
   * reset scroll); items not previously present are prepended, newest-first,
   * ahead of everything else; anything beyond the first page that the fresh
   * fetch didn't touch is left exactly where it was.
   */
  function mergeFirstPage(pageItems: T[]): void {
    const currentIds = new Set(items.value.map(getId))
    const pageById = new Map(pageItems.map((item) => [getId(item), item]))

    const updatedExisting = items.value.map((item) => pageById.get(getId(item)) ?? item)
    const genuinelyNew = pageItems.filter((item) => !currentIds.has(getId(item)))

    items.value = [...genuinelyNew, ...updatedExisting]
  }

  async function refreshFirstPage(): Promise<void> {
    if (isLoading.value || isRefreshing.value) return
    isRefreshing.value = true
    error.value = null
    try {
      const page = await fetchPage(null)
      if (!hasLoadedOnce.value) {
        // Never actually loaded (e.g. the initial load failed) — behave like loadInitial.
        items.value = page.items
        cursor.value = page.nextCursor
        hasLoadedOnce.value = true
      } else {
        mergeFirstPage(page.items)
        // The cursor tracks pagination past what's currently loaded; a
        // first-page refresh never rewinds it — only loadInitial/reset do.
      }
    } catch (err) {
      error.value = err
    } finally {
      isRefreshing.value = false
    }
  }

  return {
    items,
    isLoading,
    isLoadingMore,
    hasMore,
    error,
    loadInitial,
    loadMore,
    reset,
    mergeFirstPage,
    refreshFirstPage,
  }
}
