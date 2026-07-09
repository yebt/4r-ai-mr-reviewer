<script setup lang="ts">
import type { Review } from '@shared/api/types'
import { recommendationClass, recommendationLabel } from '@modules/reviews/format'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'

defineProps<{
  items: Review[]
  loading?: boolean
  error?: string | null
}>()
</script>

<template>
  <div>
    <p v-if="loading" class="py-3 text-sm text-muted">Loading reviews…</p>
    <p v-else-if="error" class="py-3 text-sm text-danger">{{ error }}</p>
    <p v-else-if="items.length === 0" class="py-3 text-sm text-muted">No reviews yet.</p>

    <ul v-else class="border-t border-line/50">
      <li v-for="rv in items" :key="rv.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-3">
            <RouterLink :to="`/reviews/${rv.id}`" class="font-mono text-sm text-ink hover:text-accent">
              !{{ rv.mrIid }}
            </RouterLink>
            <ReviewStatusChip :status="rv.status" />
          </div>
          <div class="label-mono mt-0.5">{{ rv.contextMode }}</div>
        </div>
        <div class="shrink-0 text-right">
          <div v-if="rv.status === 'done'" class="text-sm" :class="recommendationClass[rv.recommendation]">
            {{ recommendationLabel(rv.recommendation) }}
          </div>
          <div v-if="rv.status === 'done'" class="label-mono mt-0.5">score {{ rv.score }}</div>
          <div v-else-if="rv.status === 'error'" class="text-xs text-danger">failed</div>
        </div>
      </li>
    </ul>
  </div>
</template>
