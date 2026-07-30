<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { ConfirmDecision, RoutineRun } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { formatDateTime, isRunActive, routineKindLabel } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'
import RoutineRunDetail from '@modules/routines/components/RoutineRunDetail.vue'

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

// Expanded run whose step ledger is shown. Read back through the store so the
// detail observes the same live object the poller refreshes.
const selectedRunId = ref<string | null>(null)
const selectedRun = computed(() =>
  selectedRunId.value ? store.runById(selectedRunId.value) : null,
)
function toggleSelect(id: string) {
  selectedRunId.value = selectedRunId.value === id ? null : id
}

// --- Live polling ---
// While any run is pending/running, refresh those runs on the same cadence the
// per-repo routines section uses (2.5s). awaiting_confirmation is NOT active, so
// the loop idles once every run is resting or waiting on the user, and resumes
// after a confirm/resume flips a run back to running.
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

// --- Resume a blocked run ---
const resuming = ref(false)
async function onResume(id: string) {
  resuming.value = true
  try {
    await store.resume(id)
    startPolling()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    resuming.value = false
  }
}

// --- Confirmation gate ---
// Tracks the run id + decision currently in flight so the run-detail spins the
// clicked button only.
const confirmingId = ref<string | null>(null)
const confirmingDecision = ref<ConfirmDecision | null>(null)
const detailConfirming = computed(() =>
  selectedRun.value && confirmingId.value === selectedRun.value.id
    ? confirmingDecision.value
    : null,
)

async function onConfirm(id: string, decision: ConfirmDecision) {
  confirmingId.value = id
  confirmingDecision.value = decision
  try {
    await store.confirm(id, decision)
    startPolling()
    toast.success(decision === 'merge' ? 'Merging release…' : 'Left for manual merge')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    confirmingId.value = null
    confirmingDecision.value = null
  }
}

onMounted(async () => {
  await store.listRecent()
  startPolling()
})
onUnmounted(pause)

// Any newly-active run (after a fresh list, resume, or confirm) starts the poll.
watch(runs, () => startPolling())
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
        <button
          type="button"
          class="flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1 text-left"
          :aria-expanded="selectedRunId === run.id"
          @click="toggleSelect(run.id)"
        >
          <span
            class="text-ink text-sm"
            :class="selectedRunId === run.id ? 'text-accent' : ''"
          >
            {{ repoLabel(run) }}
          </span>
          <span v-if="run.mrIid" class="text-muted font-mono text-xs">!{{ run.mrIid }}</span>
          <RoutineStatusChip :status="run.status" />
          <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
        </button>
        <div class="label-mono shrink-0">
          <span v-if="run.updatedAt">{{ formatDateTime(run.updatedAt) }}</span>
        </div>
      </li>
    </ul>

    <RoutineRunDetail
      v-if="selectedRun"
      :run="selectedRun"
      :resuming="resuming"
      :confirming="detailConfirming"
      @resume="onResume"
      @confirm="onConfirm"
    />
  </div>
</template>
