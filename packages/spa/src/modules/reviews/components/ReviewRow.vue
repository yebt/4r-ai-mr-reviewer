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
import ScoreMeter from '@shared/components/charts/ScoreMeter.vue'

// One review row, shared by ReviewList (per-repo) and the global reviews list.
// It owns every per-review action so the two lists can never drift apart again.
// `repoName`, when set, leads with "repoName · !iid" for cross-repo lists.
const props = defineProps<{ review: Review; busy?: boolean; repoName?: string }>()
defineEmits<{
  approve: [id: string]
  discard: [id: string, mrIid: number]
  retry: [id: string]
  archive: [id: string]
  unarchive: [id: string]
}>()

const busy = () => props.busy ?? false
</script>

<template>
  <li class="row justify-between gap-3">
    <div class="min-w-0">
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
        <RouterLink
          :to="`/reviews/${review.id}`"
          class="text-ink text-sm hover:underline"
          :class="repoName ? '' : 'font-mono'"
          :style="{ viewTransitionName: `review-${review.id}` }"
        >
          <template v-if="repoName">{{ repoName }} · </template>!{{ review.mrIid }}
        </RouterLink>
        <ReviewStatusChip :status="review.status" />
      </div>
      <div class="label-mono mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1">
        <!-- Model chip: leads the meta so two reviews of the same MR run on
             different models are told apart at a glance. Bordered mono pill,
             normal-cased so long slugs stay readable. -->
        <span
          v-if="review.model"
          class="border-line text-ink inline-flex items-center gap-1 border px-2 py-0.5 font-mono text-xs tracking-normal normal-case"
          :title="`Model: ${review.model}`"
        >
          <span class="i-lucide-cpu text-muted" aria-hidden="true" />
          {{ review.model }}
        </span>
        <span class="text-muted">#{{ shortId(review.id) }}</span>
        <span>{{ review.contextMode }}</span>
        <span v-if="review.createdAt">{{ formatDateTime(review.createdAt) }}</span>
      </div>
    </div>

    <div class="flex shrink-0 items-center gap-3">
      <!-- Verdict readout for a finished review. -->
      <div v-if="review.status === 'done'" class="text-right">
        <div class="text-sm" :class="recommendationClass[review.recommendation]">
          {{ recommendationLabel(review.recommendation) }}
        </div>
        <div class="label-mono mt-0.5">score {{ review.score }}</div>
        <ScoreMeter :value="review.score" class="mt-1 ml-auto w-16" />
      </div>

      <!-- Retry a failed (active) review — never a dead end. -->
      <button
        v-if="!review.archived && review.status === 'error'"
        class="btn-line text-xs"
        :disabled="busy()"
        :aria-label="`Retry review !${review.mrIid}`"
        title="Retry — clones this review and re-runs it"
        @click="$emit('retry', review.id)"
      >
        <span
          :class="busy() ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-refresh-cw'"
          class="text-sm"
          aria-hidden="true"
        />
        Retry
      </button>

      <!-- Approve / discard a held (awaiting-approval) review. -->
      <template v-if="!review.archived && review.status === 'awaiting_approval'">
        <button
          class="btn-line text-xs"
          :disabled="busy()"
          :aria-label="`Approve review !${review.mrIid}`"
          title="Approve and run"
          @click="$emit('approve', review.id)"
        >
          <span
            :class="busy() ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-play'"
            class="text-sm"
            aria-hidden="true"
          />
          Approve
        </button>
        <button
          class="btn-ghost text-danger text-xs"
          :disabled="busy()"
          :aria-label="`Discard review !${review.mrIid}`"
          title="Discard"
          @click="$emit('discard', review.id, review.mrIid)"
        >
          <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
        </button>
      </template>

      <!-- Archive / unarchive (not offered on a held review, which uses discard). -->
      <button
        v-if="review.archived || review.status !== 'awaiting_approval'"
        class="btn-ghost text-xs"
        :disabled="busy() || (!review.archived && !isTerminal(review.status))"
        :aria-label="
          review.archived ? `Unarchive review !${review.mrIid}` : `Archive review !${review.mrIid}`
        "
        :title="
          review.archived
            ? 'Unarchive'
            : isTerminal(review.status)
              ? 'Archive'
              : 'Cannot archive a running review'
        "
        @click="review.archived ? $emit('unarchive', review.id) : $emit('archive', review.id)"
      >
        <span
          :class="review.archived ? 'i-lucide-archive-restore' : 'i-lucide-archive'"
          class="text-sm"
          aria-hidden="true"
        />
      </button>
    </div>
  </li>
</template>
