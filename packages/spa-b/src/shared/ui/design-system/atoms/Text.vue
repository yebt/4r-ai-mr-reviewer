<script setup lang="ts">
/**
 * Type-scale helper for body text. Maps to the `text-xs`..`text-3xl` scale
 * defined in tokens.css; never hardcode a font-size elsewhere.
 */
import { computed } from 'vue'

type TextSize = 'xs' | 'sm' | 'base' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'

const props = withDefaults(
  defineProps<{
    /** Rendered element/component. */
    as?: string
    size?: TextSize
    /** Use the muted (secondary) text color. */
    muted?: boolean
    /** Use the monospace font (for measured/data values). */
    mono?: boolean
  }>(),
  {
    as: 'p',
    size: 'base',
    muted: false,
    mono: false,
  },
)

// Literal class map so Tailwind's static scanner can find these utilities.
const sizeClassMap: Record<TextSize, string> = {
  xs: 'text-xs',
  sm: 'text-sm',
  base: 'text-base',
  md: 'text-md',
  lg: 'text-lg',
  xl: 'text-xl',
  '2xl': 'text-2xl',
  '3xl': 'text-3xl',
}

const sizeClass = computed(() => sizeClassMap[props.size])
</script>

<template>
  <component
    :is="as"
    :class="[sizeClass, muted ? 'text-text-muted' : 'text-text', mono ? 'font-mono' : 'font-sans']"
  >
    <slot />
  </component>
</template>
