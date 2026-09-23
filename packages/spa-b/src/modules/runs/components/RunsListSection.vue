<script setup lang="ts">
/**
 * Runs list — organism, first cut: global list + live polling only (no
 * launch modal, no detail route — those land in a later milestone). Mirrors
 * `RepositoriesSection`'s dense-list-rows-with-a-`⋯`-`DropdownMenu` shape,
 * backed by `useRunsStore` (query + live polling + the 4 action mutations).
 */
import { ref } from 'vue'
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Badge, Button, ConfirmDialog, Icon, Skeleton, Switch, Text } from '@shared/ui/design-system'
import RunStatusChip from './RunStatusChip.vue'
import { useRunsStore } from '../store'
import { flowLabel, formatDateTime, isRunCancelable, routineKindLabel, runTitle } from '../format'
import type { RoutineRun } from '../types'

const store = useRunsStore()

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
      v-else-if="store.runs.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="list" size="lg" class="text-text-muted" />
      <Text muted>{{ store.archived ? 'No archived runs.' : 'No runs yet.' }}</Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="run in store.runs"
        :key="run.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
        data-testid="run-row"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">{{ run.repoName ?? 'Unknown repo' }}</Text>
            <Text muted size="sm" class="truncate">{{ runTitle(run) }}</Text>
          </div>
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
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${runTitle(run)}`">
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
                  v-if="!run.archived"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="handleArchive(run)"
                >
                  Archive
                </DropdownMenuItem>
                <DropdownMenuItem
                  v-else
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
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
                        class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[highlighted]:bg-danger-bg"
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
                    description="Permanently removes this run's history."
                    confirm-label="Delete"
                    :pending="deletingId === run.id"
                    @confirm="handleDelete(run)"
                  >
                    <template #trigger>
                      <DropdownMenuItem
                        class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[highlighted]:bg-danger-bg"
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
