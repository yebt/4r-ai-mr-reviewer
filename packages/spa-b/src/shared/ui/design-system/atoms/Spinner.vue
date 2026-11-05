<script setup lang="ts">
/**
 * Token-colored loading spinner. Inherits `currentColor` so it reads correctly
 * inside any Button variant (accent text on accent bg, muted text elsewhere).
 * A faint full ring with a short arc: unlike an open icon, it still reads as a
 * spinner at 12px.
 */
import { computed } from 'vue'

const sizeMap = {
  xs: 'size-3',
  sm: 'size-3.5',
  md: 'size-4',
  lg: 'size-5',
  xl: 'size-6',
} as const

type SpinnerSize = keyof typeof sizeMap

const props = withDefaults(
  defineProps<{
    size?: SpinnerSize
  }>(),
  {
    size: 'md',
  },
)

const sizeClass = computed(() => sizeMap[props.size])
</script>

<template>
  <svg
    :class="[sizeClass, 'animate-spin motion-reduce:animate-none']"
    viewBox="0 0 24 24"
    fill="none"
    role="status"
    aria-label="Loading"
  >
    <circle
      class="opacity-25"
      data-testid="spinner-track"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      stroke-width="3"
    />
    <path d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
  </svg>
</template>
