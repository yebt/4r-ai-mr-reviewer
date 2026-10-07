<script setup lang="ts">
/**
 * Reviews list — organism. Follows the same shape as `ProvidersSection`:
 * query states (loading skeleton / error+retry / empty / list) rendered
 * here, data fetching + mutations owned by `../store.ts`.
 *
 * The list is backed by the global `GET /reviews` endpoint, cursor-paginated
 * via `useReviewsStore` (`useInfiniteList`-backed — see that module's doc).
 * Infinite scroll mirrors `RunsListSection.vue`'s pre-virtualization
 * approach: a zero-height sentinel after the list is watched with
 * `useIntersectionObserver` (root = viewport, so it fires regardless of which
 * ancestor actually scrolls); entering view triggers `store.loadMore()` while
 * `store.hasMore`, guarded on `!store.isLoadingMore` so an already-in-view
 * sentinel (e.g. a short filtered result) doesn't loop. Client-side filters
 * (`filterReviews`) only see the pages loaded so far — a known v1 limitation,
 * not a bug. There's no per-repo fan-out anymore, so there's no
 * `failedRepoNames`/warning concept to surface here either — a failed request
 * is a single `store.error`.
 *
 * Each row navigates to `/reviews/{id}` (the detail page) on click; the
 * archived toggle returns archived rows (`review.archived`), rendered
 * visually distinct (`opacity-60` + an "Archived" Badge) so it's obvious the
 * list is showing the archived set. The trailing `⋯` action column stops
 * click propagation so opening the menu (or any action inside it) never
 * also navigates the row.
 *
 * Lazy row menus: mounting a Reka `DropdownMenuRoot` + `DropdownMenuTrigger`
 * (context providers, popper/portal wiring, `Button` + `Icon`) for every
 * *rendered* row — even though almost nobody opens one — measurably costs
 * main-thread time on a warm navigation with many rows, exactly as found in
 * `RunsListSection.vue`. Only the row whose menu is open (`activeMenuId`)
 * mounts the real Reka `DropdownMenuRoot`; every other row renders a
 * lightweight plain `<button>` that's visually identical
 * (`ROW_MENU_TRIGGER_CLASS` mirrors `<Button variant="ghost" size="sm">`) and
 * carries the same a11y attributes (`aria-label`, `aria-haspopup="menu"`,
 * `aria-expanded="false"`). Clicking it sets `activeMenuId`, swapping that
 * row's plain button for the real trigger, rendered already-open
 * (`:open="true"`) so Reka moves focus into the menu the same way it would
 * for a normal click/keyboard-activated trigger. Closing (`@update:open`
 * firing `false` — Escape, outside click, or a plain item `@select`) clears
 * `activeMenuId`, unmounting the Reka subtree and swapping back to the plain
 * button, whose ref map (`rowMenuTriggerRefs`) is used to return focus to it.
 *
 * Row actions collapse into a single `⋯` menu, same as ProvidersSection's row
 * actions: the safe ones (Retry, Archive/Unarchive) listed first/undecorated,
 * Approve below a separator, and Discard last, danger-styled.
 *
 * Discard's confirmation is lifted to section level (`confirmDiscardId`),
 * mirroring `RunsListSection.vue`'s `confirmDeleteId`/`confirmCancelId`
 * pattern, instead of nesting a `ConfirmDialog` inside the row's
 * `DropdownMenuContent` as before: since the whole per-row `DropdownMenuRoot`
 * now unmounts on close (rather than always staying mounted with only its
 * `Presence`-gated content toggling), nesting the Discard confirmation there
 * would risk it being torn down mid-flow by the same close that unmounts its
 * row. Lifting it removes that risk entirely and reuses one well-tested flow,
 * driven by a hidden programmatic trigger, exactly like Delete/Cancel do.
 *
 * The repo + status filter bar mirrors `RunsListSection`'s exactly: two
 * `Select`s driven by local refs, filtered client-side via `../filters.ts`'s
 * pure `filterReviews`/`repoFilterOptions` (replaces the earlier 5-tab
 * status-only filter, which had no repo axis).
 */
import { computed, nextTick, ref } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { useRouter } from 'vue-router'
import { useIntersectionObserver } from '@vueuse/core'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import {
  Alert,
  Badge,
  Button,
  ConfirmDialog,
  Icon,
  Select,
  Skeleton,
  Spinner,
  Switch,
  Text,
} from '@shared/ui/design-system'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import {
  ALL_REPOS_VALUE,
  ALL_STATUSES_VALUE,
  REVIEW_STATUS_FILTER_OPTIONS,
  filterReviews,
  repoFilterOptions,
} from '../filters'
import { focusFirstAvailable, neighborId } from '@shared/composables/focusAfterRemoval'
import { useAutoPageWhileEmpty } from '@shared/composables/useAutoPageWhileEmpty'
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

const filtersActive = computed(() => repoFilter.value !== ALL_REPOS_VALUE || statusFilter.value !== ALL_STATUSES_VALUE)

function clearFilters() {
  repoFilter.value = ALL_REPOS_VALUE
  statusFilter.value = ALL_STATUSES_VALUE
}

// Filters run client-side over the loaded pages, and the sentinel below only
// renders when rows match — so keep paging while a filter matches nothing but
// older pages remain, instead of claiming "no match" too early.
const { searching } = useAutoPageWhileEmpty({
  filtersActive,
  matchCount: computed(() => filteredReviews.value.length),
  hasMore: computed(() => store.hasMore),
  blocked: computed(() => store.isLoading || store.isLoadingMore || !!store.error),
  loadMore: () => store.loadMore(),
})

// Infinite-scroll sentinel — only rendered while the list has rows (see the
// template's v-else branch), so this never fires against a detached/empty
// element. Guarded on `hasMore`/`isLoadingMore`, same as `RunsListSection.vue`.
const sentinelRef = ref<HTMLElement | null>(null)
useIntersectionObserver(sentinelRef, ([entry]) => {
  if (entry?.isIntersecting && store.hasMore && !store.isLoadingMore) {
    store.loadMore()
  }
})

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

// --- Lazy row action menu --------------------------------------------------
// See the "Lazy row menus" comment at the top of the file for the mounting
// strategy this implements.
const activeMenuId = ref<string | null>(null)

// Mirrors `<Button variant="ghost" size="sm">` (see Button.vue's
// `variantClassMap`/`sizeClassMap`) so the plain button is visually
// indistinguishable from the real Reka trigger it stands in for.
const ROW_MENU_TRIGGER_CLASS =
  'inline-flex shrink-0 items-center justify-center whitespace-nowrap font-medium transition-colors ' +
  'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring ' +
  'disabled:cursor-not-allowed disabled:opacity-50 bg-transparent text-text hover:bg-bg-hover ' +
  'h-7 gap-1.5 rounded-md px-2.5 text-xs'

// Per-row plain trigger buttons, keyed by review id. Used only to return
// focus to a row's `⋯` button once its Reka menu unmounts — the plain button
// and the real Reka trigger are different DOM nodes, so Reka's own
// focus-return (which targets its own trigger) can't do this across the swap.
const rowMenuTriggerRefs = new Map<string, HTMLButtonElement>()

function registerRowMenuTriggerRef(reviewId: string, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLButtonElement) {
    rowMenuTriggerRefs.set(reviewId, el)
  } else {
    rowMenuTriggerRefs.delete(reviewId)
  }
}

function openRowMenu(reviewId: string) {
  activeMenuId.value = reviewId
}

// Reka's `v-model:open` callback for the active row's (controlled)
// DropdownMenuRoot. This branch only renders while `open` is `true`, so this
// only ever fires `false` — Escape, an outside click, or a plain item
// `@select` (Discard no longer lives inside the menu, so it doesn't need
// `.prevent` here). Unmount back to the plain button and return focus to it.
function handleRowMenuOpenChange(reviewId: string, open: boolean) {
  if (open) return
  activeMenuId.value = null
  nextTick(() => {
    rowMenuTriggerRefs.get(reviewId)?.focus()
  })
}

// Discard's ConfirmDialog, lifted for the reason explained in the "Lazy row
// menus" file comment: driving a single dialog from this id + a hidden,
// programmatically-clicked trigger keeps the menu's normal close-on-select
// behavior intact and opens the dialog only after the menu has started
// closing, instead of nesting it inside the (now-unmounting) menu content.
const confirmDiscardId = ref<string | null>(null)
const confirmDiscardReview = computed(() => store.reviews.find((review) => review.id === confirmDiscardId.value) ?? null)
const discardConfirmTriggerRef = ref<HTMLButtonElement | null>(null)

function openDiscardConfirm(review: ReviewWithRepo) {
  confirmDiscardId.value = review.id
  nextTick(() => {
    discardConfirmTriggerRef.value?.click()
  })
}

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
    const after = neighborId(filteredReviews.value.map((r) => r.id), review.id)
    await store.discard(review.id)
    confirmDiscardId.value = null
    // The removed row owned focus; hand it to the next row's ⋯ (or the heading).
    await nextTick()
    focusFirstAvailable(after ? rowMenuTriggerRefs.get(after) : null, document.getElementById('reviews-heading'))
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
        <Text as="h2" id="reviews-heading" tabindex="-1" size="xl" class="font-semibold tracking-tight focus:outline-none">Reviews</Text>
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

    <div
      v-if="store.isLoading"
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

    <Alert v-else-if="store.error" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load reviews') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="filteredReviews.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="list" size="lg" class="text-text-muted" />
      <template v-if="searching">
        <Spinner size="sm" />
        <Text muted data-testid="reviews-searching">Looking through older reviews…</Text>
      </template>
      <template v-else-if="store.reviews.length === 0">
        <Text muted>No reviews yet.</Text>
        <Text muted size="sm">Reviews launched against a connected repository will show up here.</Text>
      </template>
      <template v-else>
        <Text muted>No reviews match the selected filters.</Text>
        <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
      </template>
    </div>

    <template v-else>
      <ul class="flex flex-col gap-2">
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

          <div class="flex shrink-0 items-center gap-1" @click.stop @keydown.stop>
            <!-- Closed state: a lightweight plain button, visually identical to
                 the real trigger below (see the "Lazy row menus" file comment). -->
            <button
              v-if="activeMenuId !== review.id"
              :ref="(el) => registerRowMenuTriggerRef(review.id, el)"
              type="button"
              :class="ROW_MENU_TRIGGER_CLASS"
              :aria-label="`More actions for review !${review.mrIid}`"
              aria-haspopup="menu"
              aria-expanded="false"
              @click.stop="openRowMenu(review.id)"
            >
              <Icon name="ellipsis" size="sm" />
            </button>

            <!-- Open state: the real Reka menu, mounted only for this one row,
                 rendered already-open so Reka moves focus in immediately. -->
            <DropdownMenuRoot v-else :open="true" @update:open="(open) => handleRowMenuOpenChange(review.id, open)">
              <DropdownMenuTrigger as-child>
                <Button
                  variant="ghost"
                  size="sm"
                  :aria-label="`More actions for review !${review.mrIid}`"
                  @click.stop
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
                  <DropdownMenuItem
                    :disabled="discardingId === review.id"
                    class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-danger-bg"
                    @select="openDiscardConfirm(review)"
                  >
                    Discard
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenuPortal>
            </DropdownMenuRoot>
          </div>
        </li>
      </ul>

      <div ref="sentinelRef" class="h-px w-full" aria-hidden="true" data-testid="reviews-load-more-sentinel" />

      <div
        v-if="store.isLoadingMore"
        class="flex items-center justify-center gap-2 py-3 text-sm text-text-muted"
        data-testid="reviews-loading-more"
      >
        <Spinner size="sm" />
        Loading more…
      </div>
    </template>

    <!-- Discard's ConfirmDialog, lifted to section level — see the "Lazy row
         menus" file comment for why. -->
    <ConfirmDialog
      :title="confirmDiscardReview ? `Discard review !${confirmDiscardReview.mrIid}?` : 'Discard this review?'"
      description="This permanently removes the review. This cannot be undone."
      confirm-label="Discard"
      danger
      :pending="!!confirmDiscardReview && discardingId === confirmDiscardReview.id"
      :restore-focus="() => (confirmDiscardId ? rowMenuTriggerRefs.get(confirmDiscardId) : null)"
      @confirm="confirmDiscardReview && handleDiscard(confirmDiscardReview)"
    >
      <template #trigger>
        <button ref="discardConfirmTriggerRef" type="button" class="hidden" tabindex="-1" aria-hidden="true" />
      </template>
    </ConfirmDialog>
  </section>
</template>
