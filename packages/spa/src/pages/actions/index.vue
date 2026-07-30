<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { RoutineRun } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { formatDateTime, isRunActive, routineKindLabel, runTitle } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'
import RoutineBranchChip from '@modules/routines/components/RoutineBranchChip.vue'

const store = useRoutinesStore()

const runs = computed(() => store.recentRuns)
// Stale-while-revalidate: only spin when nothing is cached yet.
const loading = computed(() => store.listLoading && runs.value.length === 0)

// Repo names arrive on the recent-runs payload but are dropped by a single-run
// refresh (the poller), so cache them by repo id and fall back to the cache.
const repoNames = ref<Record<string, string>>({})
watch(
  runs,
  (list) => {
    for (const r of list) if (r.repoName) repoNames.value[r.repoId] = r.repoName
  },
  { immediate: true },
)
function repoLabel(run: RoutineRun): string {
  return run.repoName || repoNames.value[run.repoId] || 'repo'
}

// --- Live polling ---
// Rows navigate to a dedicated detail page now, but the list itself keeps the
// statuses fresh: while any run is pending/running, refresh those runs on the
// same 2.5s cadence used elsewhere. awaiting_confirmation is NOT active, so the
// loop idles once every run is resting or waiting on the user.
const { pause, resume: resumePoll, isActive } = useIntervalFn(
  async () => {
    const active = runs.value.filter((r) => isRunActive(r.status))
    if (active.length === 0) {
      pause()
      return
    }
    await Promise.all(active.map((r) => store.refresh(r.id).catch(() => {})))
  },
  2500,
  { immediate: false },
)

function startPolling() {
  if (!isActive.value && runs.value.some((r) => isRunActive(r.status))) resumePoll()
}

onMounted(async () => {
  await store.listRecent()
  startPolling()
})
onUnmounted(pause)

// Any newly-active run (after a fresh list or a poll refresh) starts the poll.
watch(runs, () => startPolling())

// --- Delete an action ---
// Running actions are never offered a delete button, so the backend 409 is only
// a backstop; the row disappears via the store cache removal on success.
const deletingId = ref<string | null>(null)
async function remove(run: RoutineRun) {
  const ok = await confirm({
    title: 'Delete action',
    message: `Delete "${runTitle(run)}"? This removes the routine run.`,
    danger: true,
  })
  if (!ok) return
  deletingId.value = run.id
  try {
    await store.remove(run.id)
    toast.success('Action deleted')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <div>
    <PageHeader title="Actions" />
    <p class="text-muted mb-4 text-xs">
      Recent routine runs across all repositories. Open one to watch its step ledger update live and
      answer its confirmation gate.
    </p>

    <p v-if="loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="store.listError" class="text-danger py-3 text-sm">{{ store.listError }}</p>
    <EmptyState
      v-else-if="runs.length === 0"
      icon="i-lucide-git-merge"
      title="No routine runs yet"
      hint="Start a release or approve-and-tag routine from a repository to see it here."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="run in runs" :key="run.id" class="row flex-wrap justify-between gap-y-1">
        <RouterLink
          :to="`/actions/${run.id}`"
          class="group flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1"
        >
          <!-- Primary label: the MR title when captured, else the !{mrIid}
               fallback. Truncates so a long title never overflows the row. -->
          <span class="text-ink group-hover:text-accent min-w-0 truncate text-sm">
            {{ runTitle(run) }}
          </span>
          <span class="text-muted text-xs">{{ repoLabel(run) }}</span>
          <!-- The IID is a secondary hint only once the title already leads. -->
          <span v-if="run.state.mrTitle && run.mrIid" class="text-muted font-mono text-xs">
            !{{ run.mrIid }}
          </span>
          <RoutineStatusChip :status="run.status" />
          <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
          <!-- Secondary branch/flow indicator; renders only when known. -->
          <RoutineBranchChip :run="run" />
        </RouterLink>
        <div class="flex shrink-0 items-center gap-1">
          <span v-if="run.updatedAt" class="label-mono">{{ formatDateTime(run.updatedAt) }}</span>
          <button
            v-if="run.status !== 'running'"
            type="button"
            class="btn-ghost hover:text-danger"
            :disabled="deletingId === run.id"
            :aria-label="`Delete ${runTitle(run)}`"
            @click="remove(run)"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
