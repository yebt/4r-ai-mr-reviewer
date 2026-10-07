import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { useAutoPageWhileEmpty } from './useAutoPageWhileEmpty'

function setup(initial: { filtersActive?: boolean; matchCount?: number; hasMore?: boolean; blocked?: boolean }) {
  const filtersActive = ref(initial.filtersActive ?? true)
  const matchCount = ref(initial.matchCount ?? 0)
  const hasMore = ref(initial.hasMore ?? true)
  const blocked = ref(initial.blocked ?? false)
  const loadMore = vi.fn()
  const { searching } = useAutoPageWhileEmpty({ filtersActive, matchCount, hasMore, blocked, loadMore })
  return { filtersActive, matchCount, hasMore, blocked, loadMore, searching }
}

describe('useAutoPageWhileEmpty', () => {
  it('loads the next page when filters match nothing and more pages exist', () => {
    const s = setup({})
    expect(s.searching.value).toBe(true)
    expect(s.loadMore).toHaveBeenCalledTimes(1)
  })

  it('does not page without active filters, with matches, or when exhausted', () => {
    expect(setup({ filtersActive: false }).loadMore).not.toHaveBeenCalled()
    expect(setup({ matchCount: 2 }).loadMore).not.toHaveBeenCalled()
    const done = setup({ hasMore: false })
    expect(done.loadMore).not.toHaveBeenCalled()
    expect(done.searching.value).toBe(false)
  })

  it('waits while a fetch is in flight, then keeps paging until a match shows up', async () => {
    const s = setup({ blocked: true })
    expect(s.loadMore).not.toHaveBeenCalled()
    s.blocked.value = false
    await nextTick()
    expect(s.loadMore).toHaveBeenCalledTimes(1)
    s.blocked.value = true // page 2 loading
    await nextTick()
    s.blocked.value = false // page 2 settled, still no match
    await nextTick()
    expect(s.loadMore).toHaveBeenCalledTimes(2)
    s.matchCount.value = 1 // page 3 matched
    await nextTick()
    expect(s.searching.value).toBe(false)
    expect(s.loadMore).toHaveBeenCalledTimes(2)
  })
})
