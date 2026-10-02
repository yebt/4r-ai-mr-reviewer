<script setup lang="ts">
/**
 * Type-scale helper for headings. `level` picks the semantic tag (h1-h6) and
 * its default size; pass `size` to decouple visual size from semantics.
 */
import { computed } from 'vue'

type HeadingLevel = 1 | 2 | 3 | 4 | 5 | 6
type HeadingSize = 'base' | 'md' | 'lg' | 'xl' | '2xl' | '3xl'

const levelSizeMap: Record<HeadingLevel, HeadingSize> = {
  1: '3xl',
  2: '2xl',
  3: 'xl',
  4: 'lg',
  5: 'md',
  6: 'base',
}

const props = withDefaults(
  defineProps<{
    level?: HeadingLevel
    size?: HeadingSize
  }>(),
  {
    level: 2,
  },
)

// Literal class map so Tailwind's static scanner can find these utilities.
const sizeClassMap: Record<HeadingSize, string> = {
  base: 'text-base',
  md: 'text-md',
  lg: 'text-lg',
  xl: 'text-xl',
  '2xl': 'text-2xl',
  '3xl': 'text-3xl',
}

const tag = computed(() => `h${props.level}`)
const sizeClass = computed(() => sizeClassMap[props.size ?? levelSizeMap[props.level]])
</script>

<template>
  <component :is="tag" :class="[sizeClass, 'font-semibold tracking-tight text-text']">
    <slot />
  </component>
</template>
