<script setup lang="ts">
import type { Review } from '@shared/api/types'
import {
  formatDateTime,
  isTerminal,
  recommendationClass,
  recommendationLabel,
  shortId,
} from '@modules/reviews/format'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'

defineProps<{
  items: Review[]
  loading?: boolean
  error?: string | null
  busyId?: string | null
}>()

const emit = defineEmits<{ archive: [id: string]; unarchive: [id: string] }>()
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading reviews…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No reviews yet.</p>

    <ul v-else class="border-line/50 border-t">
      <li v-for="rv in items" :key="rv.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-3">
            <RouterLink
              :to="`/reviews/${rv.id}`"
              class="text-ink hover:text-accent font-mono text-sm"
            >
              !{{ rv.mrIid }}
            </RouterLink>
            <ReviewStatusChip :status="rv.status" />
          </div>
          <div class="label-mono mt-0.5 flex flex-wrap gap-x-2">
            <span class="text-muted/70">#{{ shortId(rv.id) }}</span>
            <span>{{ rv.contextMode }}</span>
            <span v-if="rv.createdAt">{{ formatDateTime(rv.createdAt) }}</span>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-3">
          <div class="text-right">
            <div
              v-if="rv.status === 'done'"
              class="text-sm"
              :class="recommendationClass[rv.recommendation]"
            >
              {{ recommendationLabel(rv.recommendation) }}
            </div>
            <div v-if="rv.status === 'done'" class="label-mono mt-0.5">score {{ rv.score }}</div>
            <div v-else-if="rv.status === 'error'" class="text-danger text-xs">failed</div>
          </div>
          <button
            class="btn-ghost text-xs"
            :disabled="busyId === rv.id || (!rv.archived && !isTerminal(rv.status))"
            :aria-label="rv.archived ? `Unarchive review !${rv.mrIid}` : `Archive review !${rv.mrIid}`"
            :title="
              rv.archived
                ? 'Unarchive'
                : isTerminal(rv.status)
                  ? 'Archive'
                  : 'Cannot archive a running review'
            "
            @click="rv.archived ? emit('unarchive', rv.id) : emit('archive', rv.id)"
          >
            <span
              :class="rv.archived ? 'i-lucide-archive-restore' : 'i-lucide-archive'"
              class="text-sm"
              aria-hidden="true"
            />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
