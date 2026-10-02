<script setup lang="ts">
/**
 * Status alert banner using the same status token vocabulary as Badge
 * (bg-{status}-bg / text-{status}-text), plus a subtle {status}-solid/30
 * border. Replaces the hand-rolled `role="alert"` banners duplicated across
 * modules (connection results, form errors, etc.).
 */
import { computed } from 'vue'
import Icon from '../atoms/Icon.vue'

export type AlertStatus = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

const props = withDefaults(
  defineProps<{
    status?: AlertStatus
    title?: string
    dismissible?: boolean
  }>(),
  {
    status: 'info',
    dismissible: false,
  },
)

const emit = defineEmits<{
  dismiss: []
}>()

// Tailwind's scanner needs literal class strings, so map statuses to full
// classes rather than interpolating `status` into a template literal.
const statusClassMap: Record<AlertStatus, string> = {
  neutral: 'border border-line bg-bg-hover text-text-muted',
  success: 'border border-success-solid/30 bg-success-bg text-success-text',
  warning: 'border border-warning-solid/30 bg-warning-bg text-warning-text',
  danger: 'border border-danger-solid/30 bg-danger-bg text-danger-text',
  info: 'border border-info-solid/30 bg-info-bg text-info-text',
}

const iconNameMap: Record<AlertStatus, string> = {
  neutral: 'info',
  success: 'circle-check',
  warning: 'triangle-alert',
  danger: 'circle-x',
  info: 'info',
}

const statusClass = computed(() => statusClassMap[props.status])
const iconName = computed(() => iconNameMap[props.status])
</script>

<template>
  <div role="alert" :data-status="status" :class="statusClass" class="flex items-start gap-2 rounded-md p-3 text-sm">
    <Icon :name="iconName" size="sm" class="mt-0.5 shrink-0" />
    <div class="min-w-0 flex-1">
      <p v-if="title" class="font-medium">{{ title }}</p>
      <div :class="{ 'mt-0.5': title }">
        <slot />
      </div>
    </div>
    <button
      v-if="dismissible"
      type="button"
      class="shrink-0 rounded-md p-0.5 opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      aria-label="Dismiss"
      @click="emit('dismiss')"
    >
      <Icon name="x" size="sm" />
    </button>
  </div>
</template>
