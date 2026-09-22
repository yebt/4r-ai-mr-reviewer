<script setup lang="ts">
/**
 * Thin wrapper around @iconify/vue. One atom, one job: render an icon at a
 * consistent size step. Icon set is Lucide (`lucide:*`) per DESIGN.md.
 */
import { computed } from 'vue'
import { Icon as Iconify } from '@iconify/vue'

const sizeMap = {
  xs: 'size-3',
  sm: 'size-3.5',
  md: 'size-4',
  lg: 'size-5',
  xl: 'size-6',
} as const

type IconSize = keyof typeof sizeMap

const props = withDefaults(
  defineProps<{
    /** Iconify icon identifier, e.g. `lucide:settings`. */
    name: string
    size?: IconSize
  }>(),
  {
    size: 'md',
  },
)

const sizeClass = computed(() => sizeMap[props.size])
</script>

<template>
  <Iconify :icon="name" :class="sizeClass" aria-hidden="true" />
</template>
