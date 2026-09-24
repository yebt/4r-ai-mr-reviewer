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
 * Infinite scroll: a zero-height sentinel after the list is watched with
 * `useIntersectionObserver` (root = viewport — this observes correctly
 * regardless of which ancestor actually scrolls, since the sentinel still
 * enters the viewport when its scrolling ancestor, `AppShell`'s `<main>`,
 * scrolls); entering view triggers `store.loadMore()` while `store.hasMore`.
 * Client-side filters (`filterRuns`) only see the pages loaded so far — a
 * known v1 limitation, not a bug.
 *
 * Lazy row menus: each row's `DropdownMenuContent` is NOT manually gated
 * behind `v-if` — Reka UI's `MenuContent` already wraps its content in a
 * `Presence` that renders `null` (no content tree, no DOM) whenever the menu
 * isn't open (`present: forceMount || open`, and this component never passes
 * `force-mount`). So mounting 30+ rows mounts 30+ *triggers* but zero
 * `DropdownMenuContent` trees until a row's `⋯` is actually clicked.
 */
import { computed, nextTick, onUnmounted, ref } from 'vue'
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
import { useReposStore } from '@modules/repos/store'
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

// Infinite-scroll sentinel — only rendered while the list has rows (see the
// template's v-else branch), so this never fires against a detached/empty
// element. Guarded on `hasMore`/`isLoadingMore` so a sentinel that's already
// in view (e.g. a short filtered result) doesn't fire `loadMore()` in a loop.
const sentinelRef = ref<HTMLElement | null>(null)
useIntersectionObserver(sentinelRef, ([entry]) => {
  if (entry?.isIntersecting && store.hasMore && !store.isLoadingMore) {
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
  } catch {
    // no-op — store already toasted the error
  } finally {
    cancellingId.value = null
  }
}

async function handleDelete(run: RoutineRun) {
  deletingId.value = run.id
  try {
    await store.removeRun(run.id)
    confirmDeleteId.value = null
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
          <Text as="h2" size="xl" class="font-semibold tracking-tight">Runs</Text>
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
      <Text v-if="store.runs.length === 0" muted>{{ store.archived ? 'No archived runs.' : 'No runs yet.' }}</Text>
      <Text v-else muted>No runs match the selected filters.</Text>
    </div>

    <template v-else>
      <ul class="flex flex-col gap-2">
        <li
          v-for="run in filteredRuns"
          :key="run.id"
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
            <DropdownMenuRoot>
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
                    <ConfirmDialog
                      title="Cancel this run?"
                      description="Stops the routine run in progress. This cannot be undone."
                      confirm-label="Cancel run"
                      :pending="cancellingId === run.id"
                      @confirm="handleCancel(run)"
                    >
                      <template #trigger>
                        <DropdownMenuItem
                          :disabled="store.isCancelling"
                          class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-danger-bg"
                          @select.prevent
                        >
                          Cancel
                        </DropdownMenuItem>
                      </template>
                    </ConfirmDialog>
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
        </li>
      </ul>

      <div ref="sentinelRef" class="h-px w-full" aria-hidden="true" data-testid="runs-load-more-sentinel" />

      <div
        v-if="store.isLoadingMore"
        class="flex items-center justify-center gap-2 py-3 text-sm text-text-muted"
        data-testid="runs-loading-more"
      >
        <Spinner size="sm" />
        Loading more…
      </div>
    </template>

    <!-- One Delete ConfirmDialog per section, driven by confirmDeleteId —
         see the comment on confirmDeleteId above for why this is lifted out
         of the per-row DropdownMenu instead of nested like Cancel's. -->
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
  </section>
</template>
