<script setup lang="ts">
import type { Review } from '@shared/api/types'
import { recommendationClass, recommendationLabel, statusLabel } from '@modules/reviews/format'

// A compact "this MR already has a review" indicator that links to the verdict.
// Renders nothing when there is no review, so callers can pass a possibly-absent
// lookup straight through. Used on the open-MR list so an MR and its review read
// as one relationship instead of the same number appearing in two sections.
defineProps<{ review?: Review }>()
</script>

<template>
  <RouterLink
    v-if="review"
    :to="`/reviews/${review.id}`"
    class="mt-1 inline-flex items-center gap-1.5 font-mono text-[0.66rem] tracking-wider uppercase hover:underline"
  >
    <span class="i-lucide-scan-search text-muted shrink-0 text-xs" aria-hidden="true" />
    <span class="text-muted">reviewed</span>
    <template v-if="review.status === 'done'">
      <span :class="recommendationClass[review.recommendation]"
        >· {{ recommendationLabel(review.recommendation) }}</span
      >
      <span class="text-muted">· {{ review.score }}</span>
    </template>
    <span v-else class="text-muted">· {{ statusLabel[review.status] }}</span>
    <span class="i-lucide-arrow-right text-muted shrink-0 text-xs" aria-hidden="true" />
  </RouterLink>
</template>
