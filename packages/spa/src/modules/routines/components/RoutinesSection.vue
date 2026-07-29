<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { ApproveAndTagInput, MergeRequest } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { isRunActive } from '@modules/routines/format'
import ApproveAndTagModal from '@modules/routines/components/ApproveAndTagModal.vue'
import RoutineRunList from '@modules/routines/components/RoutineRunList.vue'
import RoutineRunDetail from '@modules/routines/components/RoutineRunDetail.vue'

const props = defineProps<{ repoId: string; mergeRequests: MergeRequest[] }>()

const store = useRoutinesStore()

const runs = computed(() => store.runsFor(props.repoId))
// Stale-while-revalidate: only spin when nothing is cached yet.
const loading = computed(() => store.listLoading && runs.value.length === 0)

// Expanded run whose step ledger is shown. Read back through the store so the
// detail observes the same live object the poller refreshes.
const selectedRunId = ref<string | null>(null)
const selectedRun = computed(() =>
  selectedRunId.value ? store.runById(selectedRunId.value) : null,
)

function toggleSelect(id: string) {
  selectedRunId.value = selectedRunId.value === id ? null : id
}

// --- Trigger modal ---
const modalIid = ref<number | null>(null)
const submitting = ref(false)
const modalMergeRequest = computed(
  () => props.mergeRequests.find((mr) => mr.iid === modalIid.value) ?? null,
)

// Opened by the parent MR list ("Approve & tag" per row).
function open(iid: number) {
  modalIid.value = iid
}
function closeModal() {
  if (!submitting.value) modalIid.value = null
}
defineExpose({ open })

async function submit(input: ApproveAndTagInput) {
  submitting.value = true
  try {
    const run = await store.create(props.repoId, input)
    modalIid.value = null
    // Open and scroll to the freshly created run, then poll it live.
    selectedRunId.value = run.id
    startPolling()
    toast.success(`Routine started for !${run.mrIid}`)
  } catch (e) {
    // 409 (duplicate active run) / 400 (bad input) surface here.
    toast.error(errorMessage(e))
  } finally {
    submitting.value = false
  }
}

// --- Live polling ---
// While any run is pending/running, refresh those runs on the same cadence the
// review view uses (2.5s). The loop pauses itself once nothing is active, and is
// torn down on unmount.
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
    // The run flips back to pending/running — restart the live poll.
    startPolling()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    resuming.value = false
  }
}

onMounted(async () => {
  await store.list(props.repoId)
  startPolling()
})
onUnmounted(pause)

// Any newly-active run (after a fresh list, create, or resume) starts the poll.
watch(runs, () => startPolling())
</script>

<template>
  <section>
    <h2 class="section-title mb-3 flex items-center gap-2">
      <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
      Routines
    </h2>
    <p class="text-muted mb-3 text-xs">
      Run the approve &amp; tag routine on a merge request, then watch its step ledger update live.
    </p>

    <RoutineRunList
      :items="runs"
      :loading="loading"
      :error="store.listError"
      :selected-id="selectedRunId"
      @select="toggleSelect"
    />

    <RoutineRunDetail
      v-if="selectedRun"
      :run="selectedRun"
      :resuming="resuming"
      @resume="onResume"
    />

    <ApproveAndTagModal
      :open="modalIid !== null"
      :merge-request="modalMergeRequest"
      :submitting="submitting"
      @submit="submit"
      @close="closeModal"
    />
  </section>
</template>
