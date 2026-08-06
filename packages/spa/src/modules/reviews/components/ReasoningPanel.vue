<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Review } from '@shared/api/types'
import { phaseLabel } from '@modules/reviews/format'

// Renders the model's per-phase reasoning (chain-of-thought) captured during the
// 4R run. The list grows on each poll while the review is running (it is bound to
// the reactive review, so new phases render as they arrive) and stays viewable
// once the review is done.
const props = defineProps<{ review: Review }>()

const reasonings = computed(() => props.review.reasonings)
const running = computed(
  () => props.review.status === 'pending' || props.review.status === 'running',
)
const lastIndex = computed(() => reasonings.value.length - 1)

// Per-entry open overrides keyed by index. With no explicit user toggle an entry
// defaults open only while the review runs AND it is the most recent capture — so
// the latest reasoning auto-expands as each phase lands, then collapses once a
// newer phase arrives or the review finishes (default collapsed when done).
const overrides = ref<Record<number, boolean>>({})

function isOpen(i: number): boolean {
  const override = overrides.value[i]
  if (override !== undefined) return override
  return running.value && i === lastIndex.value
}

function toggle(i: number) {
  overrides.value = { ...overrides.value, [i]: !isOpen(i) }
}
</script>

<template>
  <section v-if="reasonings.length" class="border-line/50 mb-6 border-b pb-4">
    <h2 class="section-title mb-3 flex items-center gap-2">
      <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
      Reasoning
    </h2>
    <div class="flex flex-col gap-2">
      <div v-for="(r, i) in reasonings" :key="i" class="border-line/50 border">
        <button
          type="button"
          class="text-ink flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm"
          :aria-expanded="isOpen(i)"
          @click="toggle(i)"
        >
          <span class="flex items-center gap-2">
            <span class="label-mono">{{ phaseLabel(r.phase) }}</span>
            <span
              v-if="running && i === lastIndex"
              class="i-lucide-loader-circle text-accent animate-spin text-sm"
              aria-hidden="true"
            />
          </span>
          <span
            class="i-lucide-chevron-down text-muted text-sm transition-transform"
            :class="isOpen(i) ? 'rotate-180' : ''"
            aria-hidden="true"
          />
        </button>
        <p
          v-show="isOpen(i)"
          class="text-muted border-line/50 border-t px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap"
        >
          {{ r.content }}
        </p>
      </div>
    </div>
  </section>

  <!-- While the review runs with nothing captured yet, a subtle hint (never a
       scary empty box). Renders nothing once the review finishes with no
       reasoning captured. -->
  <p v-else-if="running" class="text-muted mb-6 flex items-center gap-2 text-xs">
    <span class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
    Waiting for model reasoning…
  </p>
</template>
