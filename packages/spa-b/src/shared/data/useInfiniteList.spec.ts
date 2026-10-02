import { describe, expect, it, vi } from 'vitest'
import { useInfiniteList, type InfiniteListPage } from './useInfiniteList'

interface Item {
  id: string
  label: string
}

function item(id: string, label = id): Item {
  return { id, label }
}

/** Builds a `fetchPage` fed from a fixed cursor -> page lookup table. */
function fakeFetchPage(pages: Record<string, InfiniteListPage<Item>>) {
  return vi.fn(async (cursor: string | null): Promise<InfiniteListPage<Item>> => {
    const key = cursor ?? 'null'
    const page = pages[key]
    if (!page) throw new Error(`no fake page for cursor ${key}`)
    return page
  })
}

describe('useInfiniteList', () => {
  it('loadInitial sets items and the next cursor from the first page', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a'), item('b')], nextCursor: 'cursor-1' },
    })
    const list = useInfiniteList(fetchPage)

    await list.loadInitial()

    expect(list.items.value).toEqual([item('a'), item('b')])
    expect(list.hasMore.value).toBe(true)
    expect(list.isLoading.value).toBe(false)
    expect(fetchPage).toHaveBeenCalledWith(null)
  })

  it('loadMore appends the next page and advances the cursor', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a'), item('b')], nextCursor: 'cursor-1' },
      'cursor-1': { items: [item('c')], nextCursor: 'cursor-2' },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()

    await list.loadMore()

    expect(list.items.value).toEqual([item('a'), item('b'), item('c')])
    expect(list.hasMore.value).toBe(true)
    expect(fetchPage).toHaveBeenLastCalledWith('cursor-1')
  })

  it('hasMore is false once a page returns a null nextCursor', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a')], nextCursor: 'cursor-1' },
      'cursor-1': { items: [item('b')], nextCursor: null },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()
    expect(list.hasMore.value).toBe(true)

    await list.loadMore()

    expect(list.hasMore.value).toBe(false)
    expect(list.items.value).toEqual([item('a'), item('b')])
  })

  it('loadMore is a no-op once hasMore is false', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a')], nextCursor: null },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()

    await list.loadMore()

    expect(fetchPage).toHaveBeenCalledTimes(1)
    expect(list.items.value).toEqual([item('a')])
  })

  it('refreshFirstPage updates existing items in place and preserves already-loaded later pages', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a', 'A v1'), item('b', 'B v1')], nextCursor: 'cursor-1' },
      'cursor-1': { items: [item('c', 'C v1')], nextCursor: null },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()
    await list.loadMore()
    expect(list.items.value).toEqual([item('a', 'A v1'), item('b', 'B v1'), item('c', 'C v1')])

    // Poll tick: same first-page ids, but "a" changed and "b" is unchanged.
    fetchPage.mockImplementationOnce(async () => ({
      items: [item('a', 'A v2'), item('b', 'B v1')],
      nextCursor: 'cursor-1',
    }))
    await list.refreshFirstPage()

    // "a" updated in place, "b" untouched, "c" (older page) preserved untouched at the tail.
    expect(list.items.value).toEqual([item('a', 'A v2'), item('b', 'B v1'), item('c', 'C v1')])
  })

  it('refreshFirstPage prepends genuinely-new items ahead of everything else', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a')], nextCursor: null },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()

    fetchPage.mockImplementationOnce(async () => ({
      items: [item('new1'), item('a')],
      nextCursor: null,
    }))
    await list.refreshFirstPage()

    expect(list.items.value).toEqual([item('new1'), item('a')])
  })

  it('reset clears items, cursor, and error', async () => {
    const fetchPage = fakeFetchPage({
      null: { items: [item('a')], nextCursor: 'cursor-1' },
    })
    const list = useInfiniteList(fetchPage)
    await list.loadInitial()

    list.reset()

    expect(list.items.value).toEqual([])
    expect(list.hasMore.value).toBe(false)
    expect(list.error.value).toBeNull()
  })

  it('captures a rejected fetchPage into error instead of throwing', async () => {
    const failure = new Error('boom')
    const fetchPage = vi.fn(async () => {
      throw failure
    })
    const list = useInfiniteList(fetchPage)

    await list.loadInitial()

    expect(list.error.value).toBe(failure)
    expect(list.isLoading.value).toBe(false)
    expect(list.items.value).toEqual([])
  })
})
