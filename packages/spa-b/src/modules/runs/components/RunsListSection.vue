<script setup lang="ts">
/**
 * Runs list — organism. Mirrors `RepositoriesSection`'s dense-list-rows-
 * with-a-`⋯`-`DropdownMenu` shape, backed by `useRunsStore` (query + live
 * polling + the 4 action mutations), plus a repo/status filter bar and
 * per-row navigation to the detail route (`/runs/{id}`, see
 * `src/pages/runs/[id].vue`).
 *
 * `useReposStore` is deep-imported read-only, purely to resolve repo id →
 * name for the repo filter's options (never for repo mutations) — mirrors
 * the task's "read-only" boundary between the runs and repos modules.
 */
import { computed, onUnmounted, ref } from 'vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Badge, Button, ConfirmDialog, Icon, Select, Skeleton, Switch, Text } from '@shared/ui/design-system'
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

    <div
      v-if="store.runsState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="runs-loading-skeleton"
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

    <Alert v-else-if="store.runsState.status === 'error'" status="danger">
      <p>{{ store.error?.message ?? 'Failed to load runs' }}</p>
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

    <ul v-else class="flex flex-col gap-2">
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
                  <ConfirmDialog
                    title="Delete this run?"
                    description="Permanently removes this run's history. This cannot be undone."
                    confirm-label="Delete"
                    :pending="deletingId === run.id"
                    @confirm="handleDelete(run)"
                  >
                    <template #trigger>
                      <DropdownMenuItem
                        :disabled="store.isRemoving"
                        class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-danger-bg"
                        @select.prevent
                      >
                        Delete
                      </DropdownMenuItem>
                    </template>
                  </ConfirmDialog>
                </template>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
        </div>
      </li>
    </ul>
  </section>
</template>
