<script setup lang="ts">
/**
 * Runs list — organism. Mirrors `RepositoriesSection`'s dense-list-rows-
 * with-a-`⋯`-`DropdownMenu` shape, backed by `useRunsStore` (cursor-paginated
 * infinite list + live polling + the 4 action mutations), plus a repo/status
 * filter bar and per-row navigation to the detail route (`/runs/{id}`, see
 * `src/pages/runs/[id].vue`).
 *
 * `useReposStore` is deep-imported read-only, purely to resolve repo id →
 * name for the repo filter's options (never for repo mutations) — mirrors
 * the task's "read-only" boundary between the runs and repos modules.
 *
 * Virtualized + infinite scroll: rows have variable height (titles/badges
 * wrap), so the list is rendered with `@tanstack/vue-virtual`'s dynamic-size
 * pattern — only `rowVirtualizer.getVirtualItems()` mount, each measured via
 * `measureElement` on its wrapper `<li>`, absolutely positioned by
 * `transform: translateY(item.start - scrollMargin)` inside a `<ul>` sized to
 * `getTotalSize()`. `getScrollElement` reads the real scrolling ancestor
 * (`AppShell`'s active `<main>`, injected via `useScrollContainer` — the
 * window never scrolls, so `useWindowVirtualizer` would be wrong here).
 * `scrollMargin` is the list's offset from the top of that scroll element's
 * *content* (the page header/filter bar sit above it inside the same
 * `<main>`); without it the virtualizer would think row 0 starts at the very
 * top of `<main>` and mis-map every scroll position. It's computed as
 * `listRect.top - scrollRect.top + scrollEl.scrollTop`, which stays correct
 * at any scroll offset (the `+scrollTop` cancels out however far the list
 * has already scrolled), and is re-measured on mount, whenever the scroll
 * element changes, and via `useResizeObserver` on both the list and the
 * scroll container (e.g. the filter bar wrapping on resize). Instead of the
 * old sentinel/IntersectionObserver, a watcher on the last virtual item
 * calls `store.loadMore()` once it's within `LOAD_MORE_THRESHOLD` rows of
 * the end, guarded by `store.hasMore && !store.isLoadingMore`. Client-side
 * filters (`filterRuns`) only see the pages loaded so far — a known v1
 * limitation, not a bug.
 *
 * Lazy row menus: mounting a Reka `DropdownMenuRoot` + `DropdownMenuTrigger`
 * (context providers, popper/portal wiring, `Button` + `Icon`) for every
 * *visible* row — even though almost nobody opens one — measurably costs
 * main-thread time on a warm navigation with 30+ rows. Only the row whose
 * menu is open (`activeMenuId`) mounts the real Reka `DropdownMenuRoot`; every
 * other row renders a lightweight plain `<button>` that's visually identical
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
 * Cancel's confirmation is lifted to section level (`confirmCancelId`),
 * mirroring Delete's already-solved `confirmDeleteId` pattern below, instead
 * of nesting a `ConfirmDialog` inside the row's `DropdownMenuContent` as
 * before: since the whole per-row `DropdownMenuRoot` now unmounts on close
 * (rather than always staying mounted with only its `Presence`-gated content
 * toggling), nesting the Cancel confirmation there would risk it being torn
 * down mid-flow by the same close that unmounts its row. Lifting it removes
 * that risk entirely and reuses one well-tested flow for both actions.
 */
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import { useVirtualizer } from '@tanstack/vue-virtual'
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
import { useScrollContainer } from '@shared/composables/useScrollContainer'
import { useReposStore } from '@modules/repos/store'
import { focusFirstAvailable, neighborId } from '@shared/composables/focusAfterRemoval'
import { useAutoPageWhileEmpty } from '@shared/composables/useAutoPageWhileEmpty'
import RunStatusChip from './RunStatusChip.vue'
import { useRunsStore } from '../store'
import {
  ALL_REPOS_VALUE,
  ALL_STATUSES_VALUE,
  RUN_STATUS_FILTER_OPTIONS,
  filterRuns,
  repoFilterOptions,
} from '../filters'
import { flowLabel, formatDateTime, isRunActive, isRunCancelable, routineKindLabel, runTitle } from '../format'
import type { RoutineRun } from '../types'

const store = useRunsStore()
const reposStore = useReposStore()

const repoFilter = ref(ALL_REPOS_VALUE)
const statusFilter = ref(ALL_STATUSES_VALUE)

const repoOptions = computed(() => repoFilterOptions(store.runs, reposStore.repos))
const filteredRuns = computed(() =>
  filterRuns(store.runs, { repoId: repoFilter.value, status: statusFilter.value }),
)

const filtersActive = computed(() => repoFilter.value !== ALL_REPOS_VALUE || statusFilter.value !== ALL_STATUSES_VALUE)

function clearFilters() {
  repoFilter.value = ALL_REPOS_VALUE
  statusFilter.value = ALL_STATUSES_VALUE
}

// Filters run client-side over the loaded pages and the virtualized list only
// pages in when rows exist — so keep paging while a filter matches nothing but
// older pages remain, instead of claiming "no match" too early.
const { searching } = useAutoPageWhileEmpty({
  filtersActive,
  matchCount: computed(() => filteredRuns.value.length),
  hasMore: computed(() => store.hasMore),
  blocked: computed(() => store.isLoading || store.isLoadingMore || !!store.error),
  loadMore: () => store.loadMore(),
})

// The store's polling `watch` only reacts to `hasActiveRun`/document
// visibility — it has no idea whether this page is still mounted. Without
// this, navigating away from `/runs` while a run is active leaves the
// 2.5s `useIntervalFn` refetching in the background forever.
onUnmounted(() => {
  store.pausePolling()
})

// Per-row pending id, mirroring `RepositoriesSection`'s `deletingId` pattern
// — scopes the ConfirmDialog/DropdownMenu-item loading spinner to the row
// actually being cancelled or deleted rather than every row at once.
const cancellingId = ref<string | null>(null)
const deletingId = ref<string | null>(null)

// The Delete ConfirmDialog is deliberately NOT nested inside the row's
// DropdownMenu (unlike Cancel above): nesting a Reka AlertDialog (always
// modal) inside a Reka DropdownMenu (also modal by default) — both
// portal-rendered — stacks two focus-trapping modal layers on top of each
// other. Selecting Delete would leave the DropdownMenuContent open behind
// the AlertDialogOverlay instead of cleanly closing it first. Driving a
// single ConfirmDialog from this id + a hidden, programmatically-clicked
// trigger keeps the dropdown's normal close-on-select behavior intact and
// opens the dialog only after the menu has started closing.
const confirmDeleteId = ref<string | null>(null)
const confirmDeleteRun = computed(() => store.runs.find((run) => run.id === confirmDeleteId.value) ?? null)
const deleteConfirmTriggerRef = ref<HTMLButtonElement | null>(null)

// Cancel's ConfirmDialog, lifted for the same reason as Delete's above — see
// the "Lazy row menus" comment at the top of the file for why this became
// necessary once the row's DropdownMenuRoot itself unmounts on close.
const confirmCancelId = ref<string | null>(null)
const confirmCancelRun = computed(() => store.runs.find((run) => run.id === confirmCancelId.value) ?? null)
const cancelConfirmTriggerRef = ref<HTMLButtonElement | null>(null)

function openCancelConfirm(run: RoutineRun) {
  confirmCancelId.value = run.id
  nextTick(() => {
    cancelConfirmTriggerRef.value?.click()
  })
}

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

// Per-row plain trigger buttons, keyed by run id. Used only to return focus
// to a row's `⋯` button once its Reka menu unmounts — the plain button and
// the real Reka trigger are different DOM nodes, so Reka's own focus-return
// (which targets its own trigger) can't do this across the swap.
const rowMenuTriggerRefs = new Map<string, HTMLButtonElement>()

function registerRowMenuTriggerRef(runId: string, el: Element | ComponentPublicInstance | null) {
  if (el instanceof HTMLButtonElement) {
    rowMenuTriggerRefs.set(runId, el)
  } else {
    rowMenuTriggerRefs.delete(runId)
  }
}

function openRowMenu(runId: string) {
  activeMenuId.value = runId
}

// Reka's `v-model:open` callback for the active row's (controlled)
// DropdownMenuRoot. This branch only renders while `open` is `true`, so this
// only ever fires `false` — Escape, an outside click, or a plain item
// `@select` (Cancel/Delete no longer live inside the menu, so neither needs
// `.prevent` here). Unmount back to the plain button and return focus to it.
function handleRowMenuOpenChange(runId: string, open: boolean) {
  if (open) return
  activeMenuId.value = null
  nextTick(() => {
    rowMenuTriggerRefs.get(runId)?.focus()
  })
}

// --- Virtualized list -----------------------------------------------------
// See the top-of-file comment for the overall approach.
const scrollEl = useScrollContainer()
const listContainerRef = ref<HTMLElement | null>(null)

// Distance from the top of the scroll element's *content* to the top of the
// list container, in px. Recomputed below; passed to the virtualizer as
// `scrollMargin` and subtracted back out of each item's `translateY`.
const scrollMargin = ref(0)

function measureScrollMargin() {
  const listNode = listContainerRef.value
  const scrollNode = scrollEl.value
  if (!listNode || !scrollNode) return
  const listRect = listNode.getBoundingClientRect()
  const scrollRect = scrollNode.getBoundingClientRect()
  // `+ scrollTop` makes this scroll-position independent: as the user
  // scrolls, `listRect.top - scrollRect.top` shrinks by exactly as much as
  // `scrollTop` grows, so the sum stays constant (it only changes if the
  // content *above* the list — header, filter bar — actually resizes).
  scrollMargin.value = listRect.top - scrollRect.top + scrollNode.scrollTop
}

onMounted(measureScrollMargin)
watch(scrollEl, measureScrollMargin, { immediate: true })
useResizeObserver(listContainerRef, measureScrollMargin)
useResizeObserver(scrollEl, measureScrollMargin)

const ESTIMATED_ROW_HEIGHT = 96

const rowVirtualizer = useVirtualizer<HTMLElement, HTMLElement>(
  computed(() => ({
    count: filteredRuns.value.length,
    getScrollElement: () => scrollEl.value,
    // Rough single-line row height: py-2.5 padding (20px) + title line
    // (24px) + gap-1 (4px) + repo line (20px) + gap-1 (4px) + badges row
    // (24px). Rows with wrapped titles/badges measure taller at runtime via
    // `measureElement` below — this is only the initial estimate.
    estimateSize: () => ESTIMATED_ROW_HEIGHT,
    overscan: 8,
    getItemKey: (index) => filteredRuns.value[index]?.id ?? index,
    scrollMargin: scrollMargin.value,
    // Without this, jsdom (0×0 layout in unit tests) would compute an empty
    // visible range and render zero rows — this seeds a first range so the
    // list (and its existing assertions) render without a real layout pass.
    initialRect: { width: 1024, height: 800 },
    // Vue can hand a row to the `measureElement` ref before inserting it into
    // the document; reading layout from a detached node yields 0px, which
    // collapsed rows to size 0 and made the later real measurements cascade
    // into scroll adjustments (the list opened ~1000–1600px scrolled down).
    // For a not-yet-connected row keep the cached size (or the estimate); the
    // ResizeObserver TanStack attaches reports the real size once it's laid out.
    measureElement: (el, entry, instance) => {
      const box = entry?.borderBoxSize?.[0]
      if (box) return Math.round(box.blockSize)
      if (!el.isConnected) {
        const index = Number(el.getAttribute('data-index'))
        return instance.measurementsCache[index]?.size ?? ESTIMATED_ROW_HEIGHT
      }
      return el.offsetHeight
    },
  })),
)

// Pairs each rendered virtual slot with its run, skipping any index that
// (in principle, given `count` === `filteredRuns.length`) has no backing
// row — keeps the template free of `noUncheckedIndexedAccess` optional
// chaining on every field.
const virtualRows = computed(() =>
  rowVirtualizer.value.getVirtualItems().flatMap((item) => {
    const run = filteredRuns.value[item.index]
    return run ? [{ item, run }] : []
  }),
)

// Replaces the old sentinel/IntersectionObserver: once the last rendered
// row is within LOAD_MORE_THRESHOLD of the end of the currently-loaded
// (filtered) list, fetch the next page.
const LOAD_MORE_THRESHOLD = 5
watch(virtualRows, (rows) => {
  const lastRow = rows[rows.length - 1]
  if (!lastRow) return
  if (
    lastRow.item.index >= filteredRuns.value.length - 1 - LOAD_MORE_THRESHOLD &&
    store.hasMore &&
    !store.isLoadingMore
  ) {
    store.loadMore()
  }
})

function openDeleteConfirm(run: RoutineRun) {
  confirmDeleteId.value = run.id
  nextTick(() => {
    deleteConfirmTriggerRef.value?.click()
  })
}

async function handleArchive(run: RoutineRun) {
  try {
    await store.archiveRun(run.id)
  } catch {
    // no-op — store already toasted the error
  }
}

async function handleUnarchive(run: RoutineRun) {
  try {
    await store.unarchiveRun(run.id)
  } catch {
    // no-op — store already toasted the error
  }
}

async function handleCancel(run: RoutineRun) {
  cancellingId.value = run.id
  try {
    await store.cancelRun(run.id)
    confirmCancelId.value = null
  } catch {
    // no-op — store already toasted the error
  } finally {
    cancellingId.value = null
  }
}

async function handleDelete(run: RoutineRun) {
  deletingId.value = run.id
  try {
    const after = neighborId(filteredRuns.value.map((r) => r.id), run.id)
    await store.removeRun(run.id)
    confirmDeleteId.value = null
    // The removed row owned focus; hand it to the next row's ⋯ (or the heading).
    await nextTick()
    focusFirstAvailable(after ? rowMenuTriggerRefs.get(after) : null, document.getElementById('runs-heading'))
  } catch {
    // no-op — store already toasted the error
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex min-w-0 flex-col gap-1">
        <div class="flex items-center gap-2">
          <Text as="h2" id="runs-heading" tabindex="-1" size="xl" class="font-semibold tracking-tight focus:outline-none">Runs</Text>
          <span
            v-if="store.isPolling"
            class="flex items-center gap-1 text-xs text-text-muted"
            data-testid="runs-live-indicator"
          >
            <span class="size-1.5 animate-pulse rounded-full bg-success-solid motion-reduce:animate-none" />
            Live
          </span>
        </div>
        <Text muted size="sm" class="truncate">Routine runs across every connected repository.</Text>
      </div>

      <label class="flex shrink-0 items-center gap-2 text-sm text-text-muted">
        Show archived
        <Switch v-model="store.archived" aria-label="Show archived runs" />
      </label>
    </div>

    <div class="flex flex-wrap items-end gap-3">
      <div class="flex w-full flex-col gap-1 sm:w-48">
        <label for="runs-repo-filter" class="text-sm text-text-muted">Repository</label>
        <Select id="runs-repo-filter" v-model="repoFilter" :items="repoOptions" />
      </div>
      <div class="flex w-full flex-col gap-1 sm:w-48">
        <label for="runs-status-filter" class="text-sm text-text-muted">Status</label>
        <Select id="runs-status-filter" v-model="statusFilter" :items="RUN_STATUS_FILTER_OPTIONS" />
      </div>
    </div>

    <div v-if="store.isLoading" class="flex flex-col gap-2" data-testid="runs-loading-skeleton">
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
      <p>{{ resolveErrorMessage(store.error, 'Failed to load runs') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="filteredRuns.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="list" size="lg" class="text-text-muted" />
      <template v-if="searching">
        <Spinner size="sm" />
        <Text muted data-testid="runs-searching">Looking through older runs…</Text>
      </template>
      <Text v-else-if="store.runs.length === 0" muted>{{ store.archived ? 'No archived runs.' : 'No runs yet.' }}</Text>
      <template v-else>
        <Text muted>No runs match the selected filters.</Text>
        <Button variant="outline" size="sm" @click="clearFilters">Clear filters</Button>
      </template>
    </div>

    <template v-else>
      <ul ref="listContainerRef" class="relative" :style="{ height: `${rowVirtualizer.getTotalSize()}px` }">
        <li
          v-for="{ item, run } in virtualRows"
          :key="run.id"
          :data-index="item.index"
          :ref="(el) => rowVirtualizer.measureElement(el as HTMLElement)"
          class="absolute inset-x-0 top-0 pb-2"
          :style="{ transform: `translateY(${item.start - scrollMargin}px)` }"
        >
          <div
            class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
            :class="{ 'opacity-60': run.archived }"
            data-testid="run-row"
          >
            <RouterLink
              :to="`/runs/${run.id}`"
              :aria-label="`View run ${runTitle(run)}`"
              class="flex min-w-0 flex-1 flex-col gap-1 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <div class="flex flex-wrap items-center gap-2">
                <Text class="truncate font-semibold">{{ runTitle(run) }}</Text>
                <Badge v-if="run.archived" status="neutral" data-testid="run-archived-badge">Archived</Badge>
              </div>
              <Text muted size="sm" class="truncate">{{ run.repoName ?? 'Unknown repo' }}</Text>
              <div class="flex flex-wrap items-center gap-2">
                <RunStatusChip :status="run.status" />
                <Badge status="neutral">{{ routineKindLabel[run.kind] }}</Badge>
                <Badge v-if="run.kind === 'release' && flowLabel(run.flow)" status="neutral">
                  {{ flowLabel(run.flow) }}
                  <template v-if="run.sourceBranch && run.targetBranch">
                    — {{ run.sourceBranch }}→{{ run.targetBranch }}
                  </template>
                </Badge>
                <Text muted size="xs">{{ formatDateTime(run.updatedAt) }}</Text>
              </div>
            </RouterLink>

            <div class="flex shrink-0 items-center gap-1">
              <!-- Closed state: a lightweight plain button, visually identical to
                   the real trigger below (see the "Lazy row menus" file comment). -->
              <button
                v-if="activeMenuId !== run.id"
                :ref="(el) => registerRowMenuTriggerRef(run.id, el)"
                type="button"
                :class="ROW_MENU_TRIGGER_CLASS"
                :aria-label="`More actions for ${runTitle(run)}`"
                aria-haspopup="menu"
                aria-expanded="false"
                @click.stop="openRowMenu(run.id)"
              >
                <Icon name="ellipsis" size="sm" />
              </button>

              <!-- Open state: the real Reka menu, mounted only for this one row,
                   rendered already-open so Reka moves focus in immediately. -->
              <DropdownMenuRoot v-else :open="true" @update:open="(open) => handleRowMenuOpenChange(run.id, open)">
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="sm" :aria-label="`More actions for ${runTitle(run)}`" @click.stop>
                    <Icon name="ellipsis" size="sm" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuPortal>
                  <DropdownMenuContent
                    align="end"
                    :side-offset="4"
                    class="z-30 min-w-40 rounded-md border border-line bg-bg-panel-raised p-1 shadow-token-lg"
                  >
                    <DropdownMenuItem
                      v-if="!run.archived && !isRunActive(run.status)"
                      :disabled="store.isArchiving"
                      class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                      @select="handleArchive(run)"
                    >
                      Archive
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      v-else-if="run.archived"
                      :disabled="store.isUnarchiving"
                      class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                      @select="handleUnarchive(run)"
                    >
                      Unarchive
                    </DropdownMenuItem>

                    <template v-if="isRunCancelable(run.status)">
                      <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                      <DropdownMenuItem
                        :disabled="store.isCancelling"
                        class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-danger-bg"
                        @select="openCancelConfirm(run)"
                      >
                        Cancel
                      </DropdownMenuItem>
                    </template>

                    <template v-else>
                      <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                      <DropdownMenuItem
                        :disabled="store.isRemoving"
                        class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-danger-bg"
                        @select="openDeleteConfirm(run)"
                      >
                        Delete
                      </DropdownMenuItem>
                    </template>
                  </DropdownMenuContent>
                </DropdownMenuPortal>
              </DropdownMenuRoot>
            </div>
          </div>
        </li>
      </ul>

      <div
        v-if="store.isLoadingMore"
        class="flex items-center justify-center gap-2 py-3 text-sm text-text-muted"
        data-testid="runs-loading-more"
      >
        <Spinner size="sm" />
        Loading more…
      </div>
    </template>

    <!-- One Delete and one Cancel ConfirmDialog per section, driven by
         confirmDeleteId/confirmCancelId — see the "Lazy row menus" file
         comment for why both are lifted out of the per-row DropdownMenu. -->
    <ConfirmDialog
      title="Delete this run?"
      description="Permanently removes this run's history. This cannot be undone."
      confirm-label="Delete"
      :pending="!!confirmDeleteRun && deletingId === confirmDeleteRun.id"
      @confirm="confirmDeleteRun && handleDelete(confirmDeleteRun)"
    >
      <template #trigger>
        <button ref="deleteConfirmTriggerRef" type="button" class="hidden" tabindex="-1" aria-hidden="true" />
      </template>
    </ConfirmDialog>

    <ConfirmDialog
      title="Cancel this run?"
      description="Stops the routine run in progress. This cannot be undone."
      confirm-label="Cancel run"
      :pending="!!confirmCancelRun && cancellingId === confirmCancelRun.id"
      @confirm="confirmCancelRun && handleCancel(confirmCancelRun)"
    >
      <template #trigger>
        <button ref="cancelConfirmTriggerRef" type="button" class="hidden" tabindex="-1" aria-hidden="true" />
      </template>
    </ConfirmDialog>
  </section>
</template>
