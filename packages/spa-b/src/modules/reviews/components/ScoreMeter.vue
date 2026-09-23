<script setup lang="ts">
/**
 * Visual score gauge (U2) — replaces the old plain "Score 61" text +
 * recommendation Badge in the detail page header with a compact horizontal
 * meter: the numeric 0-100 `score`, a filled bar, and the recommendation
 * label, all colored by `recommendation` (approve -> success token,
 * request_changes -> danger token, comment -> warning token) so the overall
 * read (good/bad/neutral) is visible at a glance before reading any text.
 *
 * Pure presentational atom-ish molecule — no store/API access, mirroring the
 * rest of `modules/reviews/components/`.
 */
import { computed } from 'vue'
import { Text } from '@shared/ui/design-system'
import { RECOMMENDATION_LABELS } from '../labels'
import type { ReviewRecommendation } from '../types'

const props = defineProps<{
  /** 0-100 review score. Clamped defensively in case the server ever sends out-of-range data. */
  score: number
  recommendation: ReviewRecommendation
}>()

const TONE_CLASSES: Record<ReviewRecommendation, { bar: string; badgeBg: string; badgeText: string }> = {
  approve: { bar: 'bg-success-solid', badgeBg: 'bg-success-bg', badgeText: 'text-success-text' },
  request_changes: { bar: 'bg-danger-solid', badgeBg: 'bg-danger-bg', badgeText: 'text-danger-text' },
  comment: { bar: 'bg-warning-solid', badgeBg: 'bg-warning-bg', badgeText: 'text-warning-text' },
}

const clampedScore = computed(() => Math.max(0, Math.min(100, props.score)))
const tone = computed(() => TONE_CLASSES[props.recommendation])
</script>

<template>
  <div
    class="flex min-w-0 items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 sm:max-w-sm"
    role="meter"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-valuenow="clampedScore"
    :aria-label="`Review score ${clampedScore} out of 100, ${RECOMMENDATION_LABELS[recommendation]}`"
  >
    <div class="flex shrink-0 flex-col items-center leading-none">
      <Text size="xl" class="font-semibold">{{ clampedScore }}</Text>
      <Text muted size="xs">/ 100</Text>
    </div>
    <div class="flex min-w-0 flex-1 flex-col gap-1.5">
      <div class="h-2 w-full overflow-hidden rounded-full bg-bg-hover">
        <div
          class="h-full rounded-full transition-[width] duration-300"
          :class="tone.bar"
          :style="{ width: `${clampedScore}%` }"
        />
      </div>
      <span
        class="inline-flex w-fit items-center rounded-full px-2 py-0.5 text-xs font-medium"
        :class="[tone.badgeBg, tone.badgeText]"
      >
        {{ RECOMMENDATION_LABELS[recommendation] }}
      </span>
    </div>
  </div>
</template>
