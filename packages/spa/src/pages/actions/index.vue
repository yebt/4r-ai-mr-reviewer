<script setup lang="ts">
definePage({ meta: { title: 'Actions' } })
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import Skeleton from '@shared/components/ui/Skeleton.vue'
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

// --- Archive ---
// Only terminal runs (done/cancelled) can be archived; running/blocked/awaiting
// are left alone. Archive is reversible via Show archived → Restore.
const isTerminal = (s: string) => s === 'done' || s === 'cancelled'
const archivable = computed(() => runs.value.filter((r) => isTerminal(r.status)))
const archivingId = ref<string | null>(null)
const archivingAll = ref(false)

async function archiveRun(run: RoutineRun) {
  archivingId.value = run.id
  try {
    await store.archive(run.id)
    toast.success('Action archived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingId.value = null
  }
}

async function archiveAll() {
  const targets = [...archivable.value]
  if (targets.length === 0 || archivingAll.value) return
  const ok = await confirm({
    title: 'Archive finished actions',
    message: `Archive ${targets.length} finished ${targets.length === 1 ? 'action' : 'actions'}? They move to the archived list — you can restore them anytime.`,
  })
  if (!ok) return
  archivingAll.value = true
  try {
    for (const run of targets) await store.archive(run.id)
    toast.success(`Archived ${targets.length} ${targets.length === 1 ? 'action' : 'actions'}`)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingAll.value = false
  }
}

// --- Archived list ---
const showArchived = ref(false)
const archivedLoaded = ref(false)
const archived = computed(() => store.archivedRuns)
async function toggleArchived() {
  showArchived.value = !showArchived.value
  if (showArchived.value && !archivedLoaded.value) {
    await store.listRecentArchived()
    archivedLoaded.value = true
  }
}
async function unarchiveRun(run: RoutineRun) {
  archivingId.value = run.id
  try {
    await store.unarchive(run.id)
    toast.success('Action restored')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingId.value = null
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

    <div class="mb-4 flex flex-wrap items-center justify-end gap-2">
      <button
        v-if="archivable.length > 0"
        class="btn-ghost text-xs"
        :disabled="archivingAll"
        title="Archive every finished action"
        @click="archiveAll"
      >
        <span
          :class="archivingAll ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive'"
          class="text-sm"
          aria-hidden="true"
        />
        Archive all ({{ archivable.length }})
      </button>
      <button class="btn-ghost text-xs" @click="toggleArchived">
        <span
          :class="showArchived ? 'i-lucide-eye-off' : 'i-lucide-archive'"
          class="text-sm"
          aria-hidden="true"
        />
        {{ showArchived ? 'Hide archived' : 'Show archived' }}
      </button>
    </div>

    <!-- Skeleton rows while the first load is in flight and nothing is cached. -->
    <ul v-if="loading" class="border-line/50 border-t" aria-hidden="true">
      <li v-for="n in 5" :key="n" class="row justify-between">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <Skeleton w="w-48" h="h-4" />
          <Skeleton w="w-20" h="h-3" />
          <Skeleton w="w-16" h="h-3" />
        </div>
        <Skeleton w="w-24" h="h-3" />
      </li>
    </ul>
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
          <span
            class="text-ink min-w-0 truncate text-sm group-hover:underline"
            :style="{ viewTransitionName: `action-${run.id}` }"
          >
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
            v-if="isTerminal(run.status)"
            type="button"
            class="btn-ghost text-xs"
            :disabled="archivingId === run.id"
            :aria-label="`Archive ${runTitle(run)}`"
            title="Archive"
            @click="archiveRun(run)"
          >
            <span
              :class="archivingId === run.id ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive'"
              class="text-sm"
              aria-hidden="true"
            />
          </button>
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

    <!-- Archived actions (loaded on demand). Restore returns a run to the active
         list; archived runs are terminal, so no live actions are offered. -->
    <template v-if="showArchived">
      <h2 class="section-title text-muted mt-8 mb-3 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Archived
      </h2>
      <p v-if="store.archivedLoading && archived.length === 0" class="text-muted py-3 text-sm">
        Loading…
      </p>
      <p v-else-if="store.archivedError" class="text-danger py-3 text-sm">
        {{ store.archivedError }}
      </p>
      <p v-else-if="archived.length === 0" class="text-muted py-3 text-sm">No archived actions.</p>
      <ul v-else class="border-line/50 border-t">
        <li v-for="run in archived" :key="run.id" class="row flex-wrap justify-between gap-y-1">
          <RouterLink
            :to="`/actions/${run.id}`"
            class="group flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1"
          >
            <span class="text-ink min-w-0 truncate text-sm group-hover:underline">
              {{ runTitle(run) }}
            </span>
            <span class="text-muted text-xs">{{ repoLabel(run) }}</span>
            <RoutineStatusChip :status="run.status" />
            <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
          </RouterLink>
          <div class="flex shrink-0 items-center gap-1">
            <span v-if="run.updatedAt" class="label-mono">{{ formatDateTime(run.updatedAt) }}</span>
            <button
              type="button"
              class="btn-ghost text-xs"
              :disabled="archivingId === run.id"
              :aria-label="`Restore ${runTitle(run)}`"
              title="Restore"
              @click="unarchiveRun(run)"
            >
              <span
                :class="
                  archivingId === run.id ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive-restore'
                "
                class="text-sm"
                aria-hidden="true"
              />
            </button>
          </div>
        </li>
      </ul>
    </template>
  </div>
</template>
