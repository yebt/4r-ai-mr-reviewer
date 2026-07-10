<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ values: number[]; max?: number }>(), { max: 100 })

const W = 100
const H = 30

const coords = computed(() => {
  const n = props.values.length
  return props.values.map((v, i) => {
    const x = n <= 1 ? W : (i / (n - 1)) * W
    const y = H - (Math.min(props.max, Math.max(0, v)) / props.max) * H
    return [x, y] as const
  })
})

const line = computed(() =>
  coords.value.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(' '),
)
const area = computed(() => (coords.value.length ? `0,${H} ${line.value} ${W},${H}` : ''))
</script>

<template>
  <svg
    :viewBox="`0 0 ${W} ${H}`"
    class="text-accent h-8 w-full"
    preserveAspectRatio="none"
    role="img"
    aria-label="Score trend"
  >
    <polygon v-if="area" :points="area" fill="currentColor" fill-opacity="0.12" />
    <polyline
      v-if="coords.length"
      :points="line"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      vector-effect="non-scaling-stroke"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
</template>
