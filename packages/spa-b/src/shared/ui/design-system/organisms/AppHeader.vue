<script setup lang="ts">
/**
 * Desktop page header: eyebrow + title on the left, a free-form actions
 * slot on the right. Rendered by AppShell above the routed content; mobile
 * uses its own compact top bar (brand + search trigger) instead — see
 * AppShell.vue, which branches structurally rather than reflowing this.
 */
import Heading from '../atoms/Heading.vue'

withDefaults(
  defineProps<{
    eyebrow?: string
    title: string
  }>(),
  {
    eyebrow: undefined,
  },
)
</script>

<template>
  <header class="flex items-center justify-between gap-4 border-b border-line px-6 py-4">
    <div class="flex min-w-0 flex-col gap-0.5">
      <span
        v-if="eyebrow"
        class="text-xs font-medium tracking-wide text-text-muted uppercase"
      >
        {{ eyebrow }}
      </span>
      <Heading :level="1" size="xl" class="truncate">{{ title }}</Heading>
    </div>
    <div class="flex shrink-0 items-center gap-2">
      <slot name="actions" />
    </div>
  </header>
</template>
