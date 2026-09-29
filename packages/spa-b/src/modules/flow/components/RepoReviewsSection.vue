<script setup lang="ts">
/**
 * Reviews tab — organism. A repo's reviews (`GET /repos/{id}/reviews`),
 * newest first, each row linking to the review detail page
 * (`src/pages/reviews/[id].vue`). Shares its `@pinia/colada` query key
 * (`repoReviewsQueryKey(repoId, false)`) with `MergeRequestListSection`'s
 * best-effort latest-review-status lookup when this tab's own "Show
 * archived" switch is off, so switching between the MRs and Reviews tabs
 * for the same repo only ever fetches the active list once. Toggling the
 * switch on reads a second, disjoint cache entry (`repoReviewsQueryKey(repoId,
 * true)`, backed by `listRepoReviews`'s `?archived=1`) — mirrors
 * `RunsListSection.vue`'s "Show archived" `Switch` and `ReviewsListSection.
 * vue`'s dimmed-row + "Archived" `Badge` treatment for the global lists.
 */
import { computed, ref } from 'vue'
import { useQuery } from '@pinia/colada'
import { Alert, Badge, Button, Icon, Skeleton, Switch, Text } from '@shared/ui/design-system'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { listRepoReviews, repoReviewsQueryKey } from '@modules/reviews/api'
import { RECOMMENDATION_LABELS, ReviewStatusChip } from '@modules/reviews'

const props = defineProps<{ repoId: string }>()

const archived = ref(false)

const { data, state, error, refetch } = useQuery({
  key: () => repoReviewsQueryKey(props.repoId, archived.value),
  query: () => listRepoReviews(props.repoId, archived.value),
})

const reviews = computed(() => data.value ?? [])

// `Intl.DateTimeFormat` construction is the expensive part of formatting a
// date — cache one module-level instance and reuse it, mirroring
// `modules/runs/format.ts#formatDateTime`'s reasoning.
const reviewDateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : reviewDateTimeFormatter.format(d)
}
</script>

<template>
  <section class="flex flex-col gap-3">
    <div class="flex items-center justify-end">
      <label class="flex shrink-0 items-center gap-2 text-sm text-text-muted">
        Show archived
        <Switch v-model="archived" aria-label="Show archived reviews" />
      </label>
    </div>

    <div v-if="state.status === 'pending'" class="flex flex-col gap-2" data-testid="flow-reviews-loading-skeleton">
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <Skeleton class="h-4 w-48" />
      </div>
    </div>

    <Alert v-else-if="error" status="danger">
      <p>{{ resolveErrorMessage(error, 'Failed to load reviews') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="reviews.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="list" size="lg" class="text-text-muted" />
      <Text muted>{{ archived ? 'No archived reviews for this repository.' : 'No reviews for this repository yet.' }}</Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li v-for="review in reviews" :key="review.id">
        <RouterLink
          :to="`/reviews/${review.id}`"
          class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          :class="review.archived ? 'opacity-60' : ''"
        >
          <div class="flex min-w-0 flex-1 flex-col gap-1">
            <div class="flex flex-wrap items-center gap-2">
              <Text muted size="sm" mono>!{{ review.mrIid }}</Text>
              <ReviewStatusChip :status="review.status" />
              <Badge v-if="review.archived" status="neutral">Archived</Badge>
            </div>
            <div v-if="review.status === 'done'" class="flex flex-wrap items-center gap-2">
              <Badge status="neutral">{{ RECOMMENDATION_LABELS[review.recommendation] }}</Badge>
              <Text muted size="xs">Score {{ review.score }}</Text>
            </div>
          </div>
          <Text muted size="xs" class="shrink-0">{{ formatDate(review.createdAt) }}</Text>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
