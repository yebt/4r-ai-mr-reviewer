import { describe, expect, it, vi } from 'vitest'
import type { RouteRecordNormalized } from 'vue-router'
import { selectPrefetchLoaders } from './prefetch'

function makeRoute(path: string, defaultComponent: unknown): RouteRecordNormalized {
  return {
    path,
    components: { default: defaultComponent },
  } as unknown as RouteRecordNormalized
}

describe('selectPrefetchLoaders', () => {
  it('picks the lazy loaders for the allowlisted paths, in allowlist order', () => {
    const runsLoader = vi.fn()
    const runDetailLoader = vi.fn()
    const routes = [
      makeRoute('/reviews', vi.fn()),
      makeRoute('/reviews/:id', vi.fn()),
      makeRoute('/settings', vi.fn()),
      // Deliberately out of PREFETCH_PATHS order in the input.
      makeRoute('/runs/:id', runDetailLoader),
      makeRoute('/runs', runsLoader),
      makeRoute('/design', vi.fn()),
    ]

    const loaders = selectPrefetchLoaders(routes)

    expect(loaders).toHaveLength(5)
    expect(loaders[0]).toBe(routes[4]?.components?.default) // /runs
    expect(loaders[1]).toBe(runDetailLoader) // /runs/:id
  })

  it('skips a path that is not registered', () => {
    const routes = [makeRoute('/runs', vi.fn())]

    const loaders = selectPrefetchLoaders(routes)

    expect(loaders).toHaveLength(1)
  })

  it('skips a route whose default view is not a lazy loader (e.g. an eagerly-imported component object)', () => {
    const routes = [makeRoute('/runs', { name: 'EagerRunsPage' })]

    const loaders = selectPrefetchLoaders(routes)

    expect(loaders).toHaveLength(0)
  })

  it('returns an empty array when none of the allowlisted paths are registered', () => {
    expect(selectPrefetchLoaders([makeRoute('/design', vi.fn())])).toEqual([])
  })
})
