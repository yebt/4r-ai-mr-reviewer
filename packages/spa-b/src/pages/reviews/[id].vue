<script setup lang="ts">
/**
 * Review detail page (`/reviews/:id`, file-based route: `reviews/[id].vue`,
 * sibling of `reviews/index.vue` which is the list). Read-only: renders the
 * full `Review` from `GET /reviews/{id}` — summary, findings grouped by
 * dimension, and a collapsible reasoning trail. Humanize/publish are
 * deferred to a later milestone.
 *
 * Data comes from `modules/reviews/detail.ts`'s `useReviewDetail`
 * composable, which polls every 2.5s while the review is non-terminal
 * (`isReviewActive`) and the tab is visible — see that file's doc comment.
 * `repoName` is resolved read-only from `useReposStore().repos` (deep
 * import into `modules/repos`, per this milestone's scope) so the header
 * reads "repo-name · !123" the same way the list row does, falling back to
 * the raw `repoId` if the repo isn't in the store's cache.
 *
 * The row actions (Retry/Archive/Approve/Discard) reuse `useReviewsStore`'s
 * mutations rather than duplicating them — a success there only invalidates
 * the *list* query key (`['reviews', ...]`), so this page also calls its own
 * `refetch()` afterwards to pick up the new status. Discard removes the
 * review, so it navigates back to the list instead.
 */
import { computed, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Alert,
  Badge,
  Button,
  ConfirmDialog,
  Skeleton,
  Text,
} from '@shared/ui/design-system'
import { useReposStore } from '@modules/repos/store'
import { useReviewDetail } from '@modules/reviews/detail'
import { FINDING_DIMENSIONS, FINDING_SEVERITY_BADGE, groupFindingsByDimension } from '@modules/reviews/findings'
import { RECOMMENDATION_LABELS } from '@modules/reviews/labels'
import { useReviewsStore } from '@modules/reviews/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'

const route = useRoute('/reviews/[id]')
const router = useRouter()

const reviewId = computed(() => route.params.id)

const { review, state, error, refetch, pausePolling } = useReviewDetail(reviewId)
onUnmounted(pausePolling)

const reposStore = useReposStore()
const repoName = computed(() => {
  const current = review.value
  if (!current) return ''
  return reposStore.repos.find((repo) => repo.id === current.repoId)?.name ?? current.repoId
})

const groupedFindings = computed(() => (review.value ? groupFindingsByDimension(review.value.findings) : null))

const reviewsStore = useReviewsStore()
type PendingAction = 'retry' | 'archive' | 'unarchive' | 'approve' | 'discard'
const pendingAction = ref<PendingAction | null>(null)

async function runAction(action: PendingAction, run: () => Promise<unknown>) {
  pendingAction.value = action
  try {
    await run()
    await refetch()
  } catch {
    // no-op — the store already toasted the error
  } finally {
    pendingAction.value = null
  }
}

function handleRetry() {
  if (review.value) void runAction('retry', () => reviewsStore.retry(review.value!.id))
}

function handleToggleArchive() {
  if (!review.value) return
  const action: PendingAction = review.value.archived ? 'unarchive' : 'archive'
  const mutate = review.value.archived ? reviewsStore.unarchive : reviewsStore.archive
  void runAction(action, () => mutate(review.value!.id))
}

function handleApprove() {
  if (review.value) void runAction('approve', () => reviewsStore.approve(review.value!.id))
}

async function handleDiscard() {
  if (!review.value) return
  pendingAction.value = 'discard'
  try {
    await reviewsStore.discard(review.value.id)
    router.push('/reviews')
  } catch {
    // no-op — the store already toasted the error
  } finally {
    pendingAction.value = null
  }
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <RouterLink to="/reviews" class="inline-flex w-fit items-center text-sm text-text-muted hover:text-text">
      ← Reviews
    </RouterLink>

    <div v-if="state.status === 'pending'" class="flex flex-col gap-3" data-testid="review-detail-skeleton">
      <Skeleton class="h-6 w-64" />
      <Skeleton class="h-4 w-40" />
      <Skeleton class="h-32 w-full" />
    </div>

    <Alert v-else-if="state.status === 'error'" status="danger">
      <p>{{ error?.message ?? 'Failed to load review' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">Retry</Button>
    </Alert>

    <div v-else-if="review" class="flex flex-col gap-6">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="flex min-w-0 flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text as="h1" size="xl" class="truncate font-semibold tracking-tight">
              {{ repoName }} · !{{ review.mrIid }}
            </Text>
            <ReviewStatusChip :status="review.status" />
            <Badge v-if="review.archived" status="neutral">Archived</Badge>
          </div>
          <Text muted size="sm">
            {{ review.contextMode }}<span v-if="review.model"> · {{ review.model }}</span>
          </Text>
          <div v-if="review.status === 'done'" class="mt-0.5 flex flex-wrap items-center gap-2">
            <Badge status="neutral">{{ RECOMMENDATION_LABELS[review.recommendation] }}</Badge>
            <Text muted size="sm">Score {{ review.score }}</Text>
            <Text muted size="sm">{{ review.inputTokens }} in / {{ review.outputTokens }} out tokens</Text>
          </div>
        </div>

        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <Button
            v-if="review.status === 'error' || review.status === 'cancelled'"
            variant="outline"
            size="sm"
            :loading="pendingAction === 'retry'"
            @click="handleRetry"
          >
            Retry
          </Button>
          <Button
            variant="outline"
            size="sm"
            :loading="pendingAction === 'archive' || pendingAction === 'unarchive'"
            @click="handleToggleArchive"
          >
            {{ review.archived ? 'Unarchive' : 'Archive' }}
          </Button>
          <Button
            v-if="review.status === 'awaiting_approval'"
            variant="outline"
            size="sm"
            :loading="pendingAction === 'approve'"
            @click="handleApprove"
          >
            Approve
          </Button>
          <ConfirmDialog
            :title='`Discard review !${review.mrIid}?`'
            description="This permanently removes the review. This cannot be undone."
            confirm-label="Discard"
            danger
            :pending="pendingAction === 'discard'"
            @confirm="handleDiscard"
          >
            <template #trigger>
              <Button variant="ghost" size="sm" class="text-danger-text hover:bg-danger-bg">Discard</Button>
            </template>
          </ConfirmDialog>
        </div>
      </div>

      <Alert v-if="review.error" status="danger" title="Review error">
        {{ review.error }}
      </Alert>

      <div v-if="review.summary" class="rounded-lg border border-line-subtle bg-bg-panel p-4">
        <Text as="h2" size="lg" class="mb-2 font-semibold">Summary</Text>
        <Text class="whitespace-pre-wrap">{{ review.summary }}</Text>
      </div>

      <div class="flex flex-col gap-4">
        <Text as="h2" size="lg" class="font-semibold">Findings</Text>

        <Text v-if="review.findings.length === 0" muted size="sm">No findings.</Text>

        <template v-else>
          <div v-for="dimension in FINDING_DIMENSIONS" :key="dimension">
            <div v-if="groupedFindings![dimension].length > 0" class="flex flex-col gap-2">
              <Text as="h3" size="md" class="font-medium capitalize">{{ dimension }}</Text>
              <ul class="flex flex-col gap-2">
                <li
                  v-for="finding in groupedFindings![dimension]"
                  :key="finding.index"
                  class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-bg-panel p-3"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <Badge :status="FINDING_SEVERITY_BADGE[finding.severity]">{{ finding.severity }}</Badge>
                    <Badge v-if="finding.blocking" status="danger">Blocking</Badge>
                    <Text mono size="sm" muted>{{ finding.file }}:{{ finding.line }}</Text>
                  </div>
                  <Text class="font-medium">{{ finding.issue }}</Text>
                  <Text muted size="sm">{{ finding.why }}</Text>
                  <Text muted size="sm">Fix: {{ finding.fix }}</Text>
                </li>
              </ul>
            </div>
          </div>
        </template>
      </div>

      <details v-if="review.reasonings.length > 0" class="rounded-lg border border-line-subtle bg-bg-panel">
        <summary class="cursor-pointer select-none px-4 py-3 text-sm font-medium text-text">Reasoning</summary>
        <div class="flex flex-col gap-3 border-t border-line-subtle px-4 py-3">
          <div v-for="(reasoning, i) in review.reasonings" :key="i" class="flex flex-col gap-1">
            <Text size="sm" class="font-medium">{{ reasoning.phase }}</Text>
            <Text muted size="sm" class="whitespace-pre-wrap">{{ reasoning.content }}</Text>
          </div>
        </div>
      </details>
    </div>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Review" } }
</route>
