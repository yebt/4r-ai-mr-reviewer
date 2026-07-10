<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ approve: number; requestChanges: number; comment: number }>()

const total = computed(() => props.approve + props.requestChanges + props.comment)

// Status colors (validated: CVD ΔE 20, contrast pass) — always shipped with a label.
const segments = computed(() => {
  const t = total.value || 1
  return [
    {
      key: 'approve',
      label: 'Approve',
      count: props.approve,
      cls: 'bg-ok',
      pct: (props.approve / t) * 100,
    },
    {
      key: 'comment',
      label: 'Comment',
      count: props.comment,
      cls: 'bg-warn',
      pct: (props.comment / t) * 100,
    },
    {
      key: 'request',
      label: 'Request changes',
      count: props.requestChanges,
      cls: 'bg-danger',
      pct: (props.requestChanges / t) * 100,
    },
  ]
})
</script>

<template>
  <div>
    <div class="flex h-2 w-full gap-[2px]">
      <template v-for="s in segments" :key="s.key">
        <div v-if="s.count > 0" :class="s.cls" :style="{ width: `${s.pct}%` }" />
      </template>
    </div>
    <ul class="mt-3 flex flex-col gap-1.5">
      <li v-for="s in segments" :key="s.key" class="flex items-center gap-2 text-xs">
        <span class="inline-block h-2 w-2" :class="s.cls" aria-hidden="true" />
        <span class="text-muted">{{ s.label }}</span>
        <span class="text-ink ml-auto font-mono">{{ s.count }}</span>
      </li>
    </ul>
  </div>
</template>
