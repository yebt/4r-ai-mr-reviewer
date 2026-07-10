<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ value: number }>()
const width = computed(() => `${Math.min(100, Math.max(0, props.value))}%`)
</script>

<template>
  <div class="bg-line/40 h-2 w-full" role="img" :aria-label="`Score ${value} of 100`">
    <div class="bg-accent h-full" :style="{ width }" />
  </div>
</template>
