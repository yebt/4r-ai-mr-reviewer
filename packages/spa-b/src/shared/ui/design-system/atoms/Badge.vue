<script setup lang="ts">
/**
 * Status chip using the semantic status tokens (neutral/success/warning/
 * danger/info). Neutral falls back to the panel/line/muted-text triad since
 * there is no "neutral" status token in tokens.css.
 */
import { computed } from 'vue'

type BadgeStatus = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

const props = withDefaults(
  defineProps<{
    status?: BadgeStatus
  }>(),
  {
    status: 'neutral',
  },
)

// Tailwind's scanner needs literal class strings, so map statuses to full
// classes rather than interpolating `status` into a template literal.
const statusClassMap: Record<BadgeStatus, string> = {
  neutral: 'border border-line-subtle bg-bg-panel text-text-muted',
  success: 'border border-transparent bg-success-bg text-success-text',
  warning: 'border border-transparent bg-warning-bg text-warning-text',
  danger: 'border border-transparent bg-danger-bg text-danger-text',
  info: 'border border-transparent bg-info-bg text-info-text',
}

const statusClass = computed(() => statusClassMap[props.status])
</script>

<template>
  <span
    :class="statusClass"
    :data-status="status"
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
  >
    <slot />
  </span>
</template>
