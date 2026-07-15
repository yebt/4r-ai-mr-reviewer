import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { Finding } from '@shared/api/types'
import { useFindingFilters } from '@modules/reviews/useFindingFilters'

// Minimal finding factory using the real enum values (dimension/severity).
function finding(partial: Partial<Finding> & { index: number }): Finding {
  return {
    index: partial.index,
    dimension: partial.dimension ?? 'risk',
    severity: partial.severity ?? 'low',
    file: partial.file ?? '',
    line: partial.line ?? 0,
    issue: partial.issue ?? `issue ${partial.index}`,
    why: partial.why ?? '',
    fix: partial.fix ?? '',
    blocking: partial.blocking ?? false,
    published: partial.published ?? false,
  }
}

const fixtures: Finding[] = [
  finding({ index: 0, dimension: 'risk', severity: 'high', file: 'b.ts', line: 20, blocking: true, published: true }),
  finding({ index: 1, dimension: 'readability', severity: 'low', file: 'a.ts', line: 5 }),
  finding({ index: 2, dimension: 'reliability', severity: 'high', file: 'a.ts', line: 2, blocking: false }),
  finding({ index: 3, dimension: 'risk', severity: 'medium', file: 'c.ts', line: 1, blocking: true }),
  finding({ index: 4, dimension: 'resilience', severity: 'medium', file: 'a.ts', line: 2, published: true }),
]

function ids(list: Finding[]): number[] {
  return list.map((f) => f.index)
}

describe('useFindingFilters', () => {
  it('filters by dimension', () => {
    const t = useFindingFilters(ref(fixtures))
    t.toggleDimension('risk')
    expect(ids(t.visible.value).sort()).toEqual([0, 3])
    // Multiple dimensions union.
    t.toggleDimension('readability')
    expect(ids(t.visible.value).sort()).toEqual([0, 1, 3])
  })

  it('filters by severity', () => {
    const t = useFindingFilters(ref(fixtures))
    t.toggleSeverity('high')
    expect(ids(t.visible.value).sort()).toEqual([0, 2])
  })

  it('filters by blockingOnly', () => {
    const t = useFindingFilters(ref(fixtures))
    t.toggleBlockingOnly()
    expect(ids(t.visible.value).sort()).toEqual([0, 3])
  })

  it('filters by status', () => {
    const t = useFindingFilters(ref(fixtures))
    t.setStatus('published')
    expect(ids(t.visible.value).sort()).toEqual([0, 4])
    t.setStatus('unpublished')
    expect(ids(t.visible.value).sort()).toEqual([1, 2, 3])
    t.setStatus('all')
    expect(t.visible.value.length).toBe(5)
  })

  it('sorts by severity: blocking first, then rank, stable by index within ties', () => {
    const t = useFindingFilters(ref(fixtures))
    // blocking: 0 (high), 3 (medium) come first, ordered by rank then index.
    // non-blocking: 2 (high), 4 (medium), 1 (low).
    expect(ids(t.visible.value)).toEqual([0, 3, 2, 4, 1])
  })

  it('sorts by file, then line, then index', () => {
    const t = useFindingFilters(ref(fixtures))
    t.setSort('file')
    // a.ts:2 index2, a.ts:2 index4 (tie -> index), a.ts:5 index1, b.ts:20 index0, c.ts:1 index3
    expect(ids(t.visible.value)).toEqual([2, 4, 1, 0, 3])
  })

  it('counts reflect the full list regardless of active filters', () => {
    const t = useFindingFilters(ref(fixtures))
    t.toggleDimension('risk')
    t.setStatus('unpublished')
    t.toggleBlockingOnly()
    // visible is filtered down…
    expect(ids(t.visible.value)).toEqual([3])
    // …but counts still describe all 5 findings.
    expect(t.counts.value.total).toBe(5)
    expect(t.counts.value.blocking).toBe(2)
    expect(t.counts.value.byDimension).toEqual({
      risk: 2,
      readability: 1,
      reliability: 1,
      resilience: 1,
    })
    expect(t.counts.value.bySeverity).toEqual({ high: 2, medium: 2, low: 1 })
  })

  it('reset clears every filter and restores severity sort', () => {
    const t = useFindingFilters(ref(fixtures))
    t.toggleDimension('risk')
    t.toggleSeverity('high')
    t.toggleBlockingOnly()
    t.setStatus('published')
    t.setSort('file')
    expect(t.active.value).toBe(true)
    t.reset()
    expect(t.active.value).toBe(false)
    expect(t.sort.value).toBe('severity')
    expect(t.visible.value.length).toBe(5)
  })
})
