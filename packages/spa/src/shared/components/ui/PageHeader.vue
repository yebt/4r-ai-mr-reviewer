<script setup lang="ts">
defineProps<{ label?: string; title: string }>()
</script>

<template>
  <header class="mb-6 flex items-end justify-between gap-4">
    <div>
      <div v-if="label" class="label-mono">{{ label }}</div>
      <h1 class="mt-1 text-2xl font-semibold tracking-tight">{{ title }}</h1>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
