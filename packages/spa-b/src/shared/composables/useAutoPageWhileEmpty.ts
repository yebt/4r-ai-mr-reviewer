import { computed, watch, type Ref } from 'vue'

export interface AutoPageWhileEmptyOptions {
  /** True while a client-side filter narrows the loaded rows. */
  filtersActive: Ref<boolean>
  /** Rows that survive the filters among everything loaded so far. */
  matchCount: Ref<number>
  /** Server has more pages beyond what is loaded. */
  hasMore: Ref<boolean>
  /** A fetch (first page, next page, or error state) is in flight or failed — do not start another. */
  blocked: Ref<boolean>
  loadMore: () => unknown
}

/**
 * Lists here filter client-side over the pages loaded so far, so a filter
 * with no match on page 1 would otherwise say "No … match" while older
 * pages that do match are still on the server (and the infinite-scroll
 * sentinel is not rendered in the empty state). While filters are active,
 * nothing matches and more pages exist, this keeps paging one page at a
 * time until a match appears or the list is exhausted.
 *
 * Returns `searching`: true while the list is empty-so-far but still
 * looking, so the empty state can say so instead of claiming "no match".
 */
export function useAutoPageWhileEmpty(opts: AutoPageWhileEmptyOptions) {
  const searching = computed(() => opts.filtersActive.value && opts.matchCount.value === 0 && opts.hasMore.value)

  watch(
    [searching, opts.blocked],
    ([isSearching, isBlocked]) => {
      if (isSearching && !isBlocked) void opts.loadMore()
    },
    { immediate: true },
  )

  return { searching }
}
