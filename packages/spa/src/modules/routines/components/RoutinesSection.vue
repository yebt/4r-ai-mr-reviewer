<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useIntervalFn } from '@vueuse/core'
import { api, errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { MergeRequest, RoutineRun } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { isRunActive } from '@modules/routines/format'
import ReleaseModal, {
  type ReleaseSubmit,
} from '@modules/routines/components/ReleaseModal.vue'
import RoutineRunList from '@modules/routines/components/RoutineRunList.vue'

const props = defineProps<{ repoId: string; mergeRequests: MergeRequest[] }>()

const router = useRouter()
const store = useRoutinesStore()

const runs = computed(() => store.runsFor(props.repoId))
// Stale-while-revalidate: only spin when nothing is cached yet.
const loading = computed(() => store.listLoading && runs.value.length === 0)

// Only development-targeting MRs can start a dev-flow release; the backend
// rejects other targets, so we gate the trigger here to avoid a guaranteed error.
const releasableMrs = computed(() =>
  props.mergeRequests.filter((mr) => mr.targetBranch === 'development'),
)

// --- Release trigger modal (dev + main flows) ---
const modalOpen = ref(false)
const modalFlow = ref<'dev' | 'main'>('dev')
const modalIid = ref<number | null>(null)
const submitting = ref(false)
const modalMergeRequest = computed(
  () => props.mergeRequests.find((mr) => mr.iid === modalIid.value) ?? null,
)

function openDevRelease(iid: number) {
  modalFlow.value = 'dev'
  modalIid.value = iid
  modalOpen.value = true
}
function openMainRelease() {
  modalFlow.value = 'main'
  modalIid.value = null
  modalOpen.value = true
}
function closeModal() {
  if (!submitting.value) modalOpen.value = false
}

async function submit(payload: ReleaseSubmit) {
  submitting.value = true
  try {
    const run =
      payload.flow === 'main'
        ? await store.createMainRelease(props.repoId, payload.input)
        : await store.createRelease(props.repoId, payload.input)
    modalOpen.value = false
    toast.success(
      payload.flow === 'main' ? 'Release to main started' : `Release started for !${run.mrIid}`,
    )
    // Land on the run's dedicated detail page to watch it live; the detail,
    // confirmation gate, and resume all live there now.
    router.push(`/actions/${run.id}`)
  } catch (e) {
    // 400 (bad target) / 409 (duplicate active run) surface here.
    toast.error(errorMessage(e))
  } finally {
    submitting.value = false
  }
}

// --- Live polling ---
// Rows navigate to the dedicated detail page, but the list itself keeps the
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
  await store.list(props.repoId)
  startPolling()
})
onUnmounted(pause)

// Any newly-active run (after a fresh list, create, or poll refresh) starts the
// poll.
watch(runs, () => startPolling())

// --- Archived runs (loaded on demand for this repo) ---
const showArchived = ref(false)
const archivedRuns = ref<RoutineRun[]>([])
const archivedLoading = ref(false)
const archivedError = ref<string | null>(null)
const archivedLoaded = ref(false)
async function fetchArchived() {
  archivedLoading.value = true
  archivedError.value = null
  try {
    archivedRuns.value = await api.listRepoRoutines(props.repoId, true)
    archivedLoaded.value = true
  } catch (e) {
    archivedError.value = errorMessage(e)
  } finally {
    archivedLoading.value = false
  }
}
async function toggleArchived() {
  showArchived.value = !showArchived.value
  if (showArchived.value && !archivedLoaded.value) await fetchArchived()
}
// After an archive (active list) or restore (archived list), keep both in sync.
async function onActiveChanged() {
  if (archivedLoaded.value) await fetchArchived()
}
async function onArchivedChanged() {
  await Promise.all([store.list(props.repoId), fetchArchived()])
}
</script>

<template>
  <section>
    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
      <h2 class="section-title flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Routines
      </h2>
      <button type="button" class="btn-line text-xs" @click="openMainRelease">
        <span class="i-lucide-rocket text-sm" aria-hidden="true" />
        Release to main
      </button>
    </div>
    <p class="text-muted mb-4 text-xs">
      Start a release from a merge request (dev flow) or cut a development → main release, then open
      its run to watch the step ledger update live and answer its confirmation gate.
    </p>

    <!-- Dev-flow triggers: one Release action per development-targeting MR. -->
    <div class="mb-6">
      <h3 class="label-mono mb-2">Release from merge request</h3>
      <p v-if="releasableMrs.length === 0" class="text-muted py-2 text-sm">
        No open merge requests targeting <span class="font-mono">development</span>.
      </p>
      <ul v-else class="border-line/50 border-t">
        <li v-for="mr in releasableMrs" :key="mr.iid" class="row flex-wrap justify-between gap-y-2">
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-muted font-mono text-xs">!{{ mr.iid }}</span>
              <a
                :href="mr.webUrl"
                target="_blank"
                rel="noreferrer"
                class="text-ink hover:text-accent truncate text-sm"
              >
                {{ mr.title }}
              </a>
            </div>
            <div class="label-mono mt-0.5">{{ mr.sourceBranch }} → {{ mr.targetBranch }}</div>
          </div>
          <button
            type="button"
            class="btn-line text-xs"
            :aria-label="`Release !${mr.iid}`"
            @click="openDevRelease(mr.iid)"
          >
            <span class="i-lucide-rocket text-sm" aria-hidden="true" />
            Release
          </button>
        </li>
      </ul>
    </div>

    <div class="mb-2 flex items-center justify-between gap-3">
      <h3 class="label-mono">Runs</h3>
      <button class="btn-ghost text-xs" @click="toggleArchived">
        <span
          :class="showArchived ? 'i-lucide-eye-off' : 'i-lucide-archive'"
          class="text-sm"
          aria-hidden="true"
        />
        {{ showArchived ? 'Hide archived' : 'Show archived' }}
      </button>
    </div>
    <RoutineRunList
      :items="runs"
      :loading="loading"
      :error="store.listError"
      archivable
      @changed="onActiveChanged"
    />

    <template v-if="showArchived">
      <h3 class="label-mono text-muted mt-6 mb-2">Archived</h3>
      <p v-if="archivedLoading && archivedRuns.length === 0" class="text-muted py-2 text-sm">
        Loading…
      </p>
      <p v-else-if="archivedError" class="text-danger py-2 text-sm">{{ archivedError }}</p>
      <p v-else-if="archivedRuns.length === 0" class="text-muted py-2 text-sm">
        No archived runs.
      </p>
      <RoutineRunList v-else :items="archivedRuns" archivable @changed="onArchivedChanged" />
    </template>

    <ReleaseModal
      :open="modalOpen"
      :flow="modalFlow"
      :repo-id="repoId"
      :merge-request="modalMergeRequest"
      :submitting="submitting"
      @submit="submit"
      @close="closeModal"
    />
  </section>
</template>
