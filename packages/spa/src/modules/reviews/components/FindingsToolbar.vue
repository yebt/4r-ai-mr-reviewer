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

// Shared toggle-chip look: active = accent border + faint accent fill; a zero
// count dims and disables the chip. Comfortable padding so the bar reads calm.
function chipClass(isActive: boolean, disabled: boolean): string {
  const base = 'inline-flex items-center gap-1.5 border px-2.5 py-1 text-xs transition-colors'
  if (disabled) return `${base} border-line/60 text-muted/40 cursor-not-allowed`
  if (isActive) return `${base} border-accent bg-accent/10 text-ink`
  return `${base} border-line text-muted hover:text-ink`
}
// The count badge inside a chip, dimmed relative to the chip label.
const countClass = 'text-[0.9em] opacity-70'
</script>

<template>
  <!-- Two calm bands: the filters (lens / severity / blocking) over a hairline,
       then the quieter view options (status / sort). Groups are clearly labelled
       and generously spaced so the bar never reads as a wall of chips. -->
  <div class="mb-4 flex flex-col gap-3">
    <!-- Phone-only counts (desktop surfaces them beside the filter toggle). -->
    <p class="text-muted text-xs sm:hidden">
      <span class="text-ink font-medium">{{ props.counts.total }}</span> findings
      <span aria-hidden="true">·</span>
      <span class="text-flame font-medium">{{ props.counts.blocking }}</span> blocking
    </p>

    <!-- Filters: lens + severity multi-selects and the blocking toggle. -->
    <div class="flex flex-wrap items-center gap-x-6 gap-y-2.5">
      <div class="flex flex-wrap items-center gap-2">
        <span class="label-mono">Lens</span>
        <div class="flex flex-wrap gap-1">
          <button
            v-for="d in props.dimensions"
            :key="d"
            type="button"
            class="font-mono"
            :class="chipClass(props.filters.dimensions.has(d), props.counts.byDimension[d] === 0)"
            :disabled="props.counts.byDimension[d] === 0"
            :aria-pressed="props.filters.dimensions.has(d)"
            :title="dimensionLabel[d]"
            @click="emit('toggleDimension', d)"
          >
            {{ dimCode(d) }}
            <span :class="countClass">{{ props.counts.byDimension[d] }}</span>
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <span class="label-mono">Severity</span>
        <div class="flex flex-wrap gap-1">
          <button
            v-for="s in props.severities"
            :key="s"
            type="button"
            class="capitalize"
            :class="chipClass(props.filters.severities.has(s), props.counts.bySeverity[s] === 0)"
            :disabled="props.counts.bySeverity[s] === 0"
            :aria-pressed="props.filters.severities.has(s)"
            @click="emit('toggleSeverity', s)"
          >
            {{ s }}
            <span :class="countClass">{{ props.counts.bySeverity[s] }}</span>
          </button>
        </div>
      </div>

      <button
        type="button"
        :class="[chipClass(props.filters.blockingOnly, props.counts.blocking === 0), 'gap-1']"
        :disabled="props.counts.blocking === 0"
        :aria-pressed="props.filters.blockingOnly"
        @click="emit('toggleBlockingOnly')"
      >
        <span class="i-lucide-flame text-flame text-sm" aria-hidden="true" />
        Blocking
        <span :class="countClass">{{ props.counts.blocking }}</span>
      </button>
    </div>

    <!-- View options: status segment + sort + reset, quieter, under a hairline. -->
    <div class="border-line/50 flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-t pt-3">
      <div class="flex flex-wrap items-center gap-2">
        <span class="label-mono">Status</span>
        <div class="flex" role="group" aria-label="Filter by status">
          <button
            v-for="st in statuses"
            :key="st.value"
            type="button"
            class="-ml-px border px-2.5 py-1 text-xs transition-colors first:ml-0"
            :class="
              props.filters.status === st.value
                ? 'border-accent bg-accent/10 text-ink z-10'
                : 'border-line text-muted hover:text-ink'
            "
            :aria-pressed="props.filters.status === st.value"
            @click="emit('setStatus', st.value)"
          >
            {{ st.label }}
          </button>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <label class="text-muted flex items-center gap-1.5 text-xs">
          <span class="label-mono">Sort</span>
          <select
            class="field-underline w-auto py-1"
            :value="props.sort"
            @change="emit('setSort', ($event.target as HTMLSelectElement).value as FindingSort)"
          >
            <option value="severity">Severity</option>
            <option value="file">File</option>
          </select>
        </label>

        <button v-if="props.active" type="button" class="btn-ghost text-xs" @click="emit('reset')">
          <span class="i-lucide-x text-sm" aria-hidden="true" />
          Reset
        </button>
      </div>
    </div>
  </div>
</template>
