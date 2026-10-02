<script setup lang="ts">
/**
 * Token-colored loading spinner. Inherits `currentColor` so it reads correctly
 * inside any Button variant (accent text on accent bg, muted text elsewhere).
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
    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
    <path
      class="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 0 1 8-8V0C5.373 0 0 5.373 0 12h4Z"
    />
  </svg>
</template>
