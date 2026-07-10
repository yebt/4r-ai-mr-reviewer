<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: number }>()
const width = computed(() => `${Math.min(100, Math.max(0, props.value))}%`)
</script>

<template>
  <div class="h-2 w-full bg-line/40" role="img" :aria-label="`Score ${value} of 100`">
    <div class="h-full bg-accent" :style="{ width }" />
  </div>
</template>
