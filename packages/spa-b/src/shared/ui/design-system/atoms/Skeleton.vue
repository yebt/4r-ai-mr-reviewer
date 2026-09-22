<script setup lang="ts">
/**
 * Token-based loading placeholder. Size it with `width`/`height` (string or
 * number, treated as px) or via a passthrough class (`class="h-4 w-32"`) on
 * the root element — both work. Pulses via Tailwind's `animate-pulse`,
 * which already respects `prefers-reduced-motion` through Tailwind's
 * built-in `motion-reduce:` variant (no shimmer → a static block).
 */
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    width?: string | number
    height?: string | number
    /** Fully rounded (pill/circle/avatar placeholder) instead of `radius-md`. */
    rounded?: boolean
  }>(),
  {
    rounded: false,
  },
)

const style = computed(() => ({
  width: typeof props.width === 'number' ? `${props.width}px` : props.width,
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
}))
</script>

<template>
  <div
    :style="style"
    :class="rounded ? 'rounded-full' : 'rounded-md'"
    class="animate-pulse bg-bg-hover motion-reduce:animate-none"
    aria-hidden="true"
  />
</template>
