<script setup lang="ts">
import type { FindingCounts, FindingSort } from '@modules/reviews/useFindingFilters'
import type { ActiveFilter, FilterField } from '@shared/components/ui/filter-builder'
import FilterBuilder from '@shared/components/ui/FilterBuilder.vue'

// Presentational triage toolbar. The finding-filter state lives in
// useFindingFilters (in the page); this component only renders the progressive
// FilterBuilder over that state's builder `model`/`fields` and hosts the Sort
// control in the builder's trailing slot. Filter changes flow back through the
// `modelValue` v-model; the page maps them onto the typed filter state.
const props = defineProps<{
  fields: FilterField[]
  modelValue: ActiveFilter[]
  sort: FindingSort
  counts: FindingCounts
}>()

const emit = defineEmits<{
  'update:modelValue': [ActiveFilter[]]
  setSort: [s: FindingSort]
}>()
</script>

<template>
  <div class="mb-4 flex flex-col gap-3">
    <!-- Counts readout — phone only. On desktop the page's own findings summary
         line ("N findings · N blocking · Showing X of Y") already carries it, so
         showing it here too would duplicate it; the phone filter sheet has no
         such line, so it keeps this one. -->
    <p class="text-muted text-xs sm:hidden">
      <span class="text-ink font-medium">{{ props.counts.total }}</span> findings
      <span aria-hidden="true">·</span>
      <span class="text-flame font-medium">{{ props.counts.blocking }}</span> blocking
    </p>

    <FilterBuilder
      :fields="props.fields"
      :model-value="props.modelValue"
      @update:model-value="emit('update:modelValue', $event)"
    >
      <template #trailing>
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
      </template>
    </FilterBuilder>
  </div>
</template>
