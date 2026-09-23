<script setup lang="ts">
/**
 * Reviews list — organism. First cut: list only (no launch modal, no detail
 * route — those land in a later milestone). Follows the same shape as
 * `ProvidersSection`: query states (loading skeleton / error+retry / empty /
 * list) rendered here, data fetching + mutations owned by `../store.ts`.
 *
 * There is no global reviews endpoint, so the list is a per-repo fan-out
 * (see `store.ts`) — expect the empty state to be the common case until
 * reviews actually run against a connected repo.
 *
 * Row actions collapse into a single `⋯` `DropdownMenu`, same as
 * ProvidersSection's row actions: the safe ones (Retry, Archive/Unarchive)
 * listed first/undecorated, Approve below a separator, and Discard last,
 * danger-styled and behind a `ConfirmDialog` (mirrors ProvidersSection's
 * Delete).
 */
import { computed, ref } from 'vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Badge, Button, ConfirmDialog, Icon, Skeleton, Switch, Text } from '@shared/ui/design-system'
import { REVIEW_FILTERS, filterReviewsByKey, type ReviewFilterKey } from '../filters'
import { useReviewsStore } from '../store'
import ReviewStatusChip from './ReviewStatusChip.vue'
import type { ReviewRecommendation, ReviewWithRepo } from '../types'

const store = useReviewsStore()

const activeFilter = ref<ReviewFilterKey>('all')

const filterCounts = computed(() =>
  REVIEW_FILTERS.map((filter) => ({
    ...filter,
    count: filterReviewsByKey(store.reviews, filter.key).length,
  })),
)

const filteredReviews = computed(() => filterReviewsByKey(store.reviews, activeFilter.value))

function shortId(id: string): string {
  return id.slice(0, 8)
}

/**
 * `model`/`sourceBranch`/`targetBranch` are optional on the DTO, so never
 * join in an empty segment — matches `ProvidersSection`'s `subtitle()`.
 */
function metaLine(review: ReviewWithRepo): string {
  const parts: string[] = [review.contextMode]
  if (review.model) parts.push(review.model)
  parts.push(shortId(review.id))
  return parts.join(' · ')
}

const recommendationLabels: Record<ReviewRecommendation, string> = {
  approve: 'Approve',
  request_changes: 'Request changes',
  comment: 'Comment',
}

const pendingActionId = ref<string | null>(null)

async function handleRetry(review: ReviewWithRepo) {
  pendingActionId.value = review.id
  try {
    await store.retry(review.id)
  } catch {
    // no-op — store already toasted the error
  } finally {
    pendingActionId.value = null
  }
}

async function handleToggleArchive(review: ReviewWithRepo) {
  pendingActionId.value = review.id
  try {
    if (review.archived) {
      await store.unarchive(review.id)
    } else {
      await store.archive(review.id)
    }
  } catch {
    // no-op — store already toasted the error
  } finally {
    pendingActionId.value = null
  }
}

async function handleApprove(review: ReviewWithRepo) {
  pendingActionId.value = review.id
  try {
    await store.approve(review.id)
  } catch {
    // no-op — store already toasted the error
  } finally {
    pendingActionId.value = null
  }
}

const discardingId = ref<string | null>(null)
async function handleDiscard(review: ReviewWithRepo) {
  discardingId.value = review.id
  try {
    await store.discard(review.id)
  } catch {
    // no-op — store already toasted the error
  } finally {
    discardingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex min-w-0 flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Reviews</Text>
        <Text muted size="sm" class="truncate">AI code reviews run across every connected repository.</Text>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-1" role="tablist" aria-label="Filter reviews by status">
        <button
          v-for="filter in filterCounts"
          :key="filter.key"
          type="button"
          role="tab"
          :aria-selected="activeFilter === filter.key"
          class="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          :class="
            activeFilter === filter.key
              ? 'bg-accent-subtle-bg text-accent-text'
              : 'text-text-muted hover:bg-bg-hover hover:text-text'
          "
          @click="activeFilter = filter.key"
        >
          {{ filter.label }}
          <Badge :status="activeFilter === filter.key ? 'info' : 'neutral'">{{ filter.count }}</Badge>
        </button>
      </div>

      <label class="flex shrink-0 items-center gap-2 text-sm text-text-muted">
        Show archived
        <Switch :model-value="store.archived" @update:model-value="store.archived = $event" />
      </label>
    </div>

    <div
      v-if="store.state.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="reviews-loading-skeleton"
    >
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-48" />
          <Skeleton class="h-3 w-32" />
        </div>
        <Skeleton class="h-8 w-8" />
      </div>
    </div>

    <Alert v-else-if="store.state.status === 'error'" status="danger">
      <p>{{ store.error?.message ?? 'Failed to load reviews' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="filteredReviews.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="list" size="lg" class="text-text-muted" />
      <Text muted>No reviews yet.</Text>
      <Text muted size="sm">
        Reviews launched against a connected repository will show up here.
      </Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="review in filteredReviews"
        :key="review.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">{{ review.repoName }} · !{{ review.mrIid }}</Text>
            <ReviewStatusChip :status="review.status" />
          </div>
          <Text muted size="sm" class="truncate">{{ metaLine(review) }}</Text>
          <div v-if="review.status === 'done'" class="mt-0.5 flex flex-wrap items-center gap-2">
            <Badge status="neutral">{{ recommendationLabels[review.recommendation] }}</Badge>
            <Text muted size="sm">Score {{ review.score }}</Text>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button
                variant="ghost"
                size="sm"
                :aria-label="`More actions for review !${review.mrIid}`"
              >
                <Icon name="ellipsis" size="sm" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuPortal>
              <DropdownMenuContent
                align="end"
                :side-offset="4"
                class="z-30 min-w-44 rounded-md border border-line bg-bg-panel-raised p-1 shadow-token-lg"
              >
                <DropdownMenuItem
                  :disabled="pendingActionId === review.id"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                  @select="handleRetry(review)"
                >
                  Retry
                </DropdownMenuItem>
                <DropdownMenuItem
                  :disabled="pendingActionId === review.id"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                  @select="handleToggleArchive(review)"
                >
                  {{ review.archived ? 'Unarchive' : 'Archive' }}
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <DropdownMenuItem
                  v-if="review.status === 'awaiting_approval'"
                  :disabled="pendingActionId === review.id"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                  @select="handleApprove(review)"
                >
                  Approve
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Discard review !${review.mrIid}?`'
                  description="This permanently removes the review. This cannot be undone."
                  confirm-label="Discard"
                  danger
                  :pending="discardingId === review.id"
                  @confirm="handleDiscard(review)"
                >
                  <template #trigger>
                    <DropdownMenuItem
                      class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[highlighted]:bg-danger-bg"
                      @select.prevent
                    >
                      Discard
                    </DropdownMenuItem>
                  </template>
                </ConfirmDialog>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
        </div>
      </li>
    </ul>
  </section>
</template>
