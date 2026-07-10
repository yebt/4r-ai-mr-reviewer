<script setup lang="ts">
defineProps<{ icon?: string; title: string; hint?: string }>()
</script>

<template>
  <div class="flex flex-col items-center gap-2 py-12 text-center">
    <span v-if="icon" :class="icon" class="text-muted/40 text-2xl" aria-hidden="true" />
    <p class="text-ink text-sm">{{ title }}</p>
    <p v-if="hint" class="text-muted/70 max-w-xs text-xs">{{ hint }}</p>
    <div v-if="$slots.action" class="mt-2">
      <slot name="action" />
    </div>
  </div>
</template>
