<script setup lang="ts">
import type { Dimension, Severity } from '@shared/api/types'
import { dimensionLabel } from '@modules/reviews/format'
import type {
  FindingCounts,
  FindingFilters,
  FindingSort,
  FindingStatus,
} from '@modules/reviews/useFindingFilters'

// Presentational triage toolbar. State + helpers are owned by useFindingFilters
// in the page; this component only renders them and forwards intent via events.
const props = defineProps<{
  counts: FindingCounts
  filters: FindingFilters
  sort: FindingSort
  dimensions: Dimension[]
  severities: Severity[]
  active: boolean
}>()

const emit = defineEmits<{
  toggleDimension: [d: Dimension]
  toggleSeverity: [s: Severity]
  toggleBlockingOnly: []
  setStatus: [s: FindingStatus]
  setSort: [s: FindingSort]
  reset: []
}>()

// "R1"…"R4" short code taken from the shared dimension label ("R1 Risk").
function dimCode(d: Dimension): string {
  return dimensionLabel[d].split(' ')[0] ?? d
}

const statuses: { value: FindingStatus; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'published', label: 'Published' },
  { value: 'unpublished', label: 'Unpublished' },
]

// Shared toggle-chip look, mirroring the humanize tabs in FindingCard: active =
// accent border + faint accent fill; a zero count dims and disables the chip.
function chipClass(isActive: boolean, disabled: boolean): string {
  if (disabled) return 'border-line/60 text-muted/40 cursor-not-allowed'
  if (isActive) return 'border-accent text-ink bg-accent/10'
  return 'border-line text-muted hover:text-ink'
}
</script>

<template>
  <div class="mb-4 flex flex-col gap-3">
    <!-- Counts summary -->
    <p class="text-muted text-xs">
      <span class="text-ink font-medium">{{ props.counts.total }}</span> findings
      <span aria-hidden="true">·</span>
      <span class="text-flame font-medium">{{ props.counts.blocking }}</span> blocking
    </p>

    <!-- Dimension filters -->
    <div class="flex flex-wrap items-center gap-1.5">
      <span class="label-mono">dimension</span>
      <button
        v-for="d in props.dimensions"
        :key="d"
        type="button"
        class="inline-flex items-center gap-1.5 border px-2 py-0.5 font-mono text-xs transition-colors"
        :class="chipClass(props.filters.dimensions.has(d), props.counts.byDimension[d] === 0)"
        :disabled="props.counts.byDimension[d] === 0"
        :aria-pressed="props.filters.dimensions.has(d)"
        @click="emit('toggleDimension', d)"
      >
        {{ dimCode(d) }}
        <span class="text-muted">{{ props.counts.byDimension[d] }}</span>
      </button>
    </div>

    <!-- Severity filters -->
    <div class="flex flex-wrap items-center gap-1.5">
      <span class="label-mono">severity</span>
      <button
        v-for="s in props.severities"
        :key="s"
        type="button"
        class="inline-flex items-center gap-1.5 border px-2 py-0.5 font-mono text-xs uppercase transition-colors"
        :class="chipClass(props.filters.severities.has(s), props.counts.bySeverity[s] === 0)"
        :disabled="props.counts.bySeverity[s] === 0"
        :aria-pressed="props.filters.severities.has(s)"
        @click="emit('toggleSeverity', s)"
      >
        {{ s }}
        <span class="text-muted">{{ props.counts.bySeverity[s] }}</span>
      </button>
    </div>

    <!-- Blocking toggle, status control, sort, reset -->
    <div class="flex flex-wrap items-center gap-3">
      <button
        type="button"
        class="inline-flex items-center gap-1.5 border px-2 py-0.5 font-mono text-xs transition-colors"
        :class="chipClass(props.filters.blockingOnly, props.counts.blocking === 0)"
        :disabled="props.counts.blocking === 0"
        :aria-pressed="props.filters.blockingOnly"
        @click="emit('toggleBlockingOnly')"
      >
        Blocking
        <span class="text-muted">{{ props.counts.blocking }}</span>
      </button>

      <div class="flex items-center gap-1" role="group" aria-label="Filter by status">
        <button
          v-for="st in statuses"
          :key="st.value"
          type="button"
          class="border px-2 py-0.5 text-xs transition-colors"
          :class="chipClass(props.filters.status === st.value, false)"
          :aria-pressed="props.filters.status === st.value"
          @click="emit('setStatus', st.value)"
        >
          {{ st.label }}
        </button>
      </div>

      <label class="text-muted flex items-center gap-1.5 text-xs">
        <span class="label-mono">sort</span>
        <select
          class="field-underline w-auto py-1"
          :value="props.sort"
          @change="emit('setSort', ($event.target as HTMLSelectElement).value as FindingSort)"
        >
          <option value="severity">Severity</option>
          <option value="file">File</option>
        </select>
      </label>

      <button
        v-if="props.active"
        type="button"
        class="btn-ghost text-xs"
        @click="emit('reset')"
      >
        <span class="i-lucide-x text-sm" aria-hidden="true" />
        Reset
      </button>
    </div>
  </div>
</template>
