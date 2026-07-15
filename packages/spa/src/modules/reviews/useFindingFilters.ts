import { computed, reactive, ref, toValue, type MaybeRefOrGetter } from 'vue'
import type { Dimension, Finding, Severity } from '@shared/api/types'
import { severityRank } from './format'

export type FindingStatus = 'all' | 'published' | 'unpublished'
export type FindingSort = 'severity' | 'file'

export interface FindingFilters {
  dimensions: Set<Dimension>
  severities: Set<Severity>
  blockingOnly: boolean
  status: FindingStatus
}

export interface FindingCounts {
  total: number
  blocking: number
  byDimension: Record<Dimension, number>
  bySeverity: Record<Severity, number>
}

const DIMENSIONS: Dimension[] = ['risk', 'readability', 'reliability', 'resilience']
const SEVERITIES: Severity[] = ['high', 'medium', 'low']

/**
 * Triage state over a reactive list of findings. Pure (no DOM/UnoCSS): holds the
 * active filters + sort, derives the `visible` (filtered + sorted) list, and
 * `counts` computed over the FULL list so the counts strip always shows the
 * whole picture regardless of what is filtered out.
 */
export function useFindingFilters(source: MaybeRefOrGetter<Finding[]>) {
  const filters = reactive<FindingFilters>({
    dimensions: new Set<Dimension>(),
    severities: new Set<Severity>(),
    blockingOnly: false,
    status: 'all',
  })
  const sort = ref<FindingSort>('severity')

  const all = computed(() => toValue(source) ?? [])

  const visible = computed<Finding[]>(() => {
    const list = all.value.filter((f) => {
      if (filters.dimensions.size && !filters.dimensions.has(f.dimension)) return false
      if (filters.severities.size && !filters.severities.has(f.severity)) return false
      if (filters.blockingOnly && !f.blocking) return false
      if (filters.status === 'published' && !f.published) return false
      if (filters.status === 'unpublished' && f.published) return false
      return true
    })

    // Slice before sorting so the original array is never mutated in place.
    const sorted = list.slice()
    if (sort.value === 'severity') {
      // Blocking first, then most severe, stable within ties by original index.
      sorted.sort((a, b) => {
        if (a.blocking !== b.blocking) return a.blocking ? -1 : 1
        const rank = severityRank[b.severity] - severityRank[a.severity]
        if (rank !== 0) return rank
        return a.index - b.index
      })
    } else {
      // File path, then line, then original index.
      sorted.sort((a, b) => {
        if (a.file !== b.file) return a.file < b.file ? -1 : 1
        if (a.line !== b.line) return a.line - b.line
        return a.index - b.index
      })
    }
    return sorted
  })

  const counts = computed<FindingCounts>(() => {
    const byDimension = { risk: 0, readability: 0, reliability: 0, resilience: 0 } as Record<
      Dimension,
      number
    >
    const bySeverity = { high: 0, medium: 0, low: 0 } as Record<Severity, number>
    let blocking = 0
    for (const f of all.value) {
      byDimension[f.dimension]++
      bySeverity[f.severity]++
      if (f.blocking) blocking++
    }
    return { total: all.value.length, blocking, byDimension, bySeverity }
  })

  const active = computed(
    () =>
      filters.dimensions.size > 0 ||
      filters.severities.size > 0 ||
      filters.blockingOnly ||
      filters.status !== 'all',
  )

  function toggleDimension(d: Dimension) {
    if (filters.dimensions.has(d)) filters.dimensions.delete(d)
    else filters.dimensions.add(d)
  }

  function toggleSeverity(s: Severity) {
    if (filters.severities.has(s)) filters.severities.delete(s)
    else filters.severities.add(s)
  }

  function toggleBlockingOnly() {
    filters.blockingOnly = !filters.blockingOnly
  }

  function setStatus(s: FindingStatus) {
    filters.status = s
  }

  function setSort(s: FindingSort) {
    sort.value = s
  }

  function reset() {
    filters.dimensions.clear()
    filters.severities.clear()
    filters.blockingOnly = false
    filters.status = 'all'
    sort.value = 'severity'
  }

  return {
    filters,
    sort,
    visible,
    counts,
    active,
    dimensions: DIMENSIONS,
    severities: SEVERITIES,
    toggleDimension,
    toggleSeverity,
    toggleBlockingOnly,
    setStatus,
    setSort,
    reset,
  }
}
