<script setup lang="ts">
/**
 * Reviews list — organism. Follows the same shape as `ProvidersSection`:
 * query states (loading skeleton / error+retry / empty / list) rendered
 * here, data fetching + mutations owned by `../store.ts`.
 *
 * There is no global reviews endpoint, so the list is a per-repo fan-out
 * (see `store.ts`) — expect the empty state to be the common case until
 * reviews actually run against a connected repo. The fan-out uses
 * `Promise.allSettled`, so one repo failing to load never empties the whole
 * list — `store.failedRepoCount` surfaces a non-blocking warning Alert above
 * the list instead.
 *
 * Each row navigates to `/reviews/{id}` (the detail page) on click; the
 * archived toggle returns archived rows (`review.archived`), rendered
 * visually distinct (`opacity-60` + an "Archived" Badge) so it's obvious the
 * list is showing the archived set. The trailing `⋯` action column stops
 * click propagation so opening the menu (or any action inside it) never
 * also navigates the row.
 *
 * Row actions collapse into a single `⋯` `DropdownMenu`, same as
 * ProvidersSection's row actions: the safe ones (Retry, Archive/Unarchive)
 * listed first/undecorated, Approve below a separator, and Discard last,
 * danger-styled and behind a `ConfirmDialog` (mirrors ProvidersSection's
 * Delete).
 *
 * The repo + status filter bar mirrors `RunsListSection`'s exactly: two
 * `Select`s driven by local refs, filtered client-side via `../filters.ts`'s
 * pure `filterReviews`/`repoFilterOptions` (replaces the earlier 5-tab
 * status-only filter, which had no repo axis).
 */
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Badge, Button, ConfirmDialog, Icon, Select, Skeleton, Switch, Text } from '@shared/ui/design-system'
import {
  ALL_REPOS_VALUE,
  ALL_STATUSES_VALUE,
  REVIEW_STATUS_FILTER_OPTIONS,
  filterReviews,
  repoFilterOptions,
} from '../filters'
import { RECOMMENDATION_LABELS } from '../labels'
import { useReviewsStore } from '../store'
import ReviewStatusChip from './ReviewStatusChip.vue'
import type { ReviewWithRepo } from '../types'

const store = useReviewsStore()
const router = useRouter()

function goToReview(review: ReviewWithRepo) {
  router.push(`/reviews/${review.id}`)
}

const repoFilter = ref(ALL_REPOS_VALUE)
const statusFilter = ref(ALL_STATUSES_VALUE)

const repoOptions = computed(() => repoFilterOptions(store.reviews))
const filteredReviews = computed(() =>
  filterReviews(store.reviews, { repoId: repoFilter.value, status: statusFilter.value }),
)

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

    <div class="flex items-center justify-end gap-3">
      <label class="flex shrink-0 items-center gap-2 text-sm text-text-muted">
        Show archived
        <Switch :model-value="store.archived" @update:model-value="store.archived = $event" />
      </label>
    </div>

    <div class="flex flex-wrap items-end gap-3">
      <div class="flex w-full flex-col gap-1 sm:w-48">
        <label for="reviews-repo-filter" class="text-sm text-text-muted">Repository</label>
        <Select id="reviews-repo-filter" v-model="repoFilter" :items="repoOptions" />
      </div>
      <div class="flex w-full flex-col gap-1 sm:w-48">
        <label for="reviews-status-filter" class="text-sm text-text-muted">Status</label>
        <Select id="reviews-status-filter" v-model="statusFilter" :items="REVIEW_STATUS_FILTER_OPTIONS" />
      </div>
    </div>

    <Alert v-if="store.failedRepoCount > 0" status="warning">
      Couldn't load reviews from {{ store.failedRepoCount }}
      {{ store.failedRepoCount === 1 ? 'repository' : 'repositories' }}.
    </Alert>

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
      <Text v-if="store.reviews.length === 0" muted>No reviews yet.</Text>
      <Text v-else muted>No reviews match the selected filters.</Text>
      <Text v-if="store.reviews.length === 0" muted size="sm">
        Reviews launched against a connected repository will show up here.
      </Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="review in filteredReviews"
        :key="review.id"
        role="link"
        tabindex="0"
        :aria-label="`View review ${review.repoName} !${review.mrIid}`"
        class="flex cursor-pointer items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        :class="review.archived ? 'opacity-60' : ''"
        @click="goToReview(review)"
        @keydown.enter="goToReview(review)"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">!{{ review.mrIid }}</Text>
            <ReviewStatusChip :status="review.status" />
            <Badge v-if="review.archived" status="neutral">Archived</Badge>
          </div>
          <Text muted size="sm" class="truncate">{{ review.repoName }}</Text>
          <Text muted size="sm" class="truncate">{{ metaLine(review) }}</Text>
          <div v-if="review.status === 'done'" class="mt-0.5 flex flex-wrap items-center gap-2">
            <Badge status="neutral">{{ RECOMMENDATION_LABELS[review.recommendation] }}</Badge>
            <Text muted size="sm">Score {{ review.score }}</Text>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-1" @click.stop>
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
                  v-if="review.status === 'error' || review.status === 'cancelled'"
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
