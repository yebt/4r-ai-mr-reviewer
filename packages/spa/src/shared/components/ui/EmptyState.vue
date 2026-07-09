<script setup lang="ts">
defineProps<{ icon?: string; title: string; hint?: string }>()
</script>

<template>
  <div class="flex flex-col items-center gap-2 py-12 text-center">
    <span v-if="icon" :class="icon" class="text-2xl text-muted/40" aria-hidden="true" />
    <p class="text-sm text-ink">{{ title }}</p>
    <p v-if="hint" class="max-w-xs text-xs text-muted/70">{{ hint }}</p>
    <div v-if="$slots.action" class="mt-2">
      <slot name="action" />
    </div>
  </div>
</template>
