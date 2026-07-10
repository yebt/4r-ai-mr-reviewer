<script setup lang="ts">
import type { Review } from '@shared/api/types'
import { recommendationClass, recommendationLabel } from '@modules/reviews/format'

defineProps<{ review: Review }>()
</script>

<template>
  <section class="border-line/50 border-b pb-6">
    <div class="flex items-end justify-between gap-6">
      <div>
        <div class="label-mono">Recommendation</div>
        <div
          class="mt-1 text-2xl font-semibold tracking-tight"
          :class="recommendationClass[review.recommendation]"
        >
          {{ recommendationLabel(review.recommendation) }}
        </div>
      </div>
      <div class="text-right">
        <div class="label-mono">Score</div>
        <div class="text-ink mt-1 font-mono text-2xl font-semibold">
          {{ review.score }}<span class="text-muted text-base">/100</span>
        </div>
      </div>
    </div>

    <p v-if="review.summary" class="text-muted mt-4 max-w-2xl text-sm">{{ review.summary }}</p>

    <div class="label-mono mt-4 flex flex-wrap gap-x-6 gap-y-1">
      <span>mode {{ review.contextMode }}</span>
      <span>{{ review.findings.length }} findings</span>
      <span>tokens {{ review.inputTokens }} in / {{ review.outputTokens }} out</span>
    </div>
  </section>
</template>
