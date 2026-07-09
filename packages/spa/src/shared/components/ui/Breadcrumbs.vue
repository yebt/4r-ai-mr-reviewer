<script setup lang="ts">
interface Crumb {
  label: string
  to?: string
}
defineProps<{ items: Crumb[] }>()
</script>

<template>
  <nav class="mb-4 flex flex-wrap items-center gap-2 label-mono">
    <template v-for="(c, i) in items" :key="i">
      <RouterLink v-if="c.to" :to="c.to" class="hover:text-ink">{{ c.label }}</RouterLink>
      <span v-else class="text-ink">{{ c.label }}</span>
      <span v-if="i < items.length - 1" class="text-muted/40" aria-hidden="true">/</span>
    </template>
  </nav>
</template>
