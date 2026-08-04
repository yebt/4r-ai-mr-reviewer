<script setup lang="ts">
// `viewTransitionName`, when set, names the title as a View Transitions
// shared element so a list row with the same name morphs into this header on
// navigation. Optional and inert when unset or where the API is unsupported.
defineProps<{ label?: string; title: string; viewTransitionName?: string }>()
</script>

<template>
  <header class="border-line/50 mb-8 flex items-end justify-between gap-4 border-b pb-4">
    <div>
      <div v-if="label" class="label-mono">{{ label }}</div>
      <h1
        class="mt-1.5 text-2xl font-semibold tracking-tight"
        :style="viewTransitionName ? { viewTransitionName } : undefined"
      >
        {{ title }}
      </h1>
    </div>
    <div class="flex items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
