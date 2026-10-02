<script setup lang="ts">
/**
 * Small tab strip for the humanize feature (slice 2) — "Original" plus one
 * tab per humanized run (V1, V2, ...). No Tabs atom exists in the design
 * system, so this hand-rolls plain styled buttons rather than a Reka Tabs
 * primitive — reused by both the summary card and each finding card.
 */
import { ORIGINAL } from '../humanize'

const props = defineProps<{
  /** Number of humanized runs (excludes the "Original" tab). */
  tabCount: number
  /** Active tab: `ORIGINAL` or a 0-based run index. */
  active: number
}>()

const emit = defineEmits<{
  select: [tab: number]
}>()

function tabClass(isActive: boolean): string {
  return isActive
    ? 'rounded-md bg-accent-subtle-bg px-2 py-1 text-xs font-medium text-accent-text-strong'
    : 'rounded-md px-2 py-1 text-xs font-medium text-text-muted hover:bg-bg-hover'
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-1">
    <button type="button" :class="tabClass(props.active === ORIGINAL)" @click="emit('select', ORIGINAL)">
      Original
    </button>
    <button
      v-for="tab in props.tabCount"
      :key="tab"
      type="button"
      :class="tabClass(props.active === tab - 1)"
      @click="emit('select', tab - 1)"
    >
      V{{ tab }}
    </button>
  </div>
</template>
