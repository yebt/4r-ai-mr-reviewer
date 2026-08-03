<script setup lang="ts">
definePage({ meta: { title: 'Action' } })
import { computed, onUnmounted, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn, useTitle } from '@vueuse/core'
import { ApiError, errorMessage } from '@shared/api/client'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import type { ConfirmDecision } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { useReposStore } from '@modules/repos/store'
import { isRunActive, routineKindLabel, runTitle } from '@modules/routines/format'
import RoutineRunDetail from '@modules/routines/components/RoutineRunDetail.vue'

const route = useRoute()
const router = useRouter()
const store = useRoutinesStore()
const repos = useReposStore()

const runId = computed(() => (route.params as { id: string }).id)
const run = computed(() => store.runById(runId.value))

// Tab title: the routine kind plus the MR it acts on (e.g. "Release · !42"), so
// an Action tab is identifiable at a glance. Falls back to the generic label
// from definePage while the run is still loading.
useTitle(
  computed(() =>
    run.value
      ? `${routineKindLabel[run.value.kind]} · ${runTitle(run.value)} - AI Review`
      : 'Action - AI Review',
  ),
)

// Load lifecycle for the fetch-by-id: `loading` only spins when the run is not
// already cached (e.g. arriving from the list vs. a cold deep link); `notFound`
// distinguishes a 404 from a transient error so the copy can differ.
const loading = ref(false)
const loadError = ref<string | null>(null)
const notFound = ref(false)

// The single-run endpoint does not populate repoName, so resolve the repo's
// display name from the repos store (the same source the review view uses) and
// fall back to the run's own hints. Drives the "jump to repo" link + label.
const repoName = computed(() => {
  const r = run.value
  if (!r) return ''
  return repos.items.find((x) => x.id === r.repoId)?.name || r.repoName || r.repoId
})

async function load(id: string) {
  loadError.value = null
  notFound.value = false
  loading.value = store.runById(id) == null
  try {
    await store.refresh(id)
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) notFound.value = true
    else loadError.value = errorMessage(e)
  } finally {
    loading.value = false
  }
}

// --- Live polling ---
// While the run is pending/running, refresh it on the same 2.5s cadence used
// across the app. awaiting_confirmation/blocked/done are resting states, so the
// loop pauses for them and resumes after a confirm/resume flips it back.
const { pause, resume: resumePoll, isActive } = useIntervalFn(
  async () => {
    const r = run.value
    if (!r || !isRunActive(r.status)) {
      pause()
      return
    }
    await store.refresh(r.id).catch(() => {})
  },
  2500,
  { immediate: false },
)

function startPolling() {
  if (!isActive.value && run.value && isRunActive(run.value.status)) resumePoll()
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

// --- Skip a blocked run's failed step ---
// Offered by RoutineRunDetail only when the failed step is non-critical. On
// success the step is re-queued and the run flips back to running, so resume
// polling. A 409 means the step is essential / the run is no longer blocked; the
// server's message is surfaced as a friendly toast.
const skipping = ref(false)
async function onSkip(id: string) {
  skipping.value = true
  try {
    await store.skip(id)
    startPolling()
    toast.success('Step skipped')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    skipping.value = false
  }
}

// --- Confirmation gate ---
// Track the decision in flight so the run-detail spins only the clicked button.
const confirmingDecision = ref<ConfirmDecision | null>(null)
async function onConfirm(id: string, decision: ConfirmDecision) {
  confirmingDecision.value = decision
  try {
    await store.confirm(id, decision)
    startPolling()
    toast.success(decision === 'merge' ? 'Merging release…' : 'Left for manual merge')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    confirmingDecision.value = null
  }
}

// --- Cancel this action ---
// Offered while the run is still cancelable (RoutineRunDetail hides the button
// once terminal). A destructive confirm gates the abort; on success the run
// flips to `cancelled` and Delete takes over. A 409 (already terminal) is
// toasted as a backstop.
const cancelling = ref(false)
async function onCancel(id: string) {
  const r = run.value
  const ok = await confirm({
    title: 'Cancel action',
    message: `Cancel "${r ? runTitle(r) : 'this action'}"? This aborts the routine run; it cannot be resumed.`,
    danger: true,
  })
  if (!ok) return
  cancelling.value = true
  try {
    await store.cancel(id)
    pause()
    toast.success('Action cancelled')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    cancelling.value = false
  }
}

// --- Delete this action ---
// Offered only for a non-running run (RoutineRunDetail hides the button while
// running); a running run's 409 is a backstop toasted below. On success the run
// is gone from every cache, so navigate back to the Actions list.
const deleting = ref(false)
async function onDelete(id: string) {
  const r = run.value
  const ok = await confirm({
    title: 'Delete action',
    message: `Delete "${r ? runTitle(r) : 'this action'}"? This removes the routine run.`,
    danger: true,
  })
  if (!ok) return
  deleting.value = true
  try {
    await store.remove(id)
    toast.success('Action deleted')
    router.push('/actions')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    deleting.value = false
  }
}

// Prefer the captured MR title in the page header; fall back to the kind + IID
// label for a run that has not captured a title yet or a legacy run.
const headerTitle = computed(() => {
  const r = run.value
  if (!r) return 'Routine run'
  return r.state.mrTitle || `${routineKindLabel[r.kind]} !${r.mrIid}`
})

const crumbs = computed(() => {
  const items: { label: string; to?: string }[] = [{ label: 'Actions', to: '/actions' }]
  const r = run.value
  if (r) items.push({ label: `${routineKindLabel[r.kind]} !${r.mrIid}` })
  return items
})
watchEffect(() => setBreadcrumbs(crumbs.value))

watch(
  runId,
  async (id) => {
    pause()
    if (repos.items.length === 0) repos.fetchAll()
    await load(id)
    startPolling()
  },
  { immediate: true },
)
onUnmounted(pause)
</script>

<template>
  <div>
    <PageHeader :title="headerTitle" />

    <p v-if="loading && !run" class="text-muted py-3 text-sm">Loading…</p>
    <template v-else-if="notFound">
      <p class="text-muted py-3 text-sm">This routine run no longer exists.</p>
      <RouterLink to="/actions" class="text-accent text-sm hover:underline">
        Back to Actions
      </RouterLink>
    </template>
    <p v-else-if="loadError && !run" class="text-danger py-3 text-sm">{{ loadError }}</p>

    <template v-else-if="run">
      <!-- Jump to the source repository. The run belongs to a routine, so deep
           link straight to the repo's Routines tab. -->
      <RouterLink
        :to="`/repos/${run.repoId}?tab=routines`"
        class="border-line/60 bg-surface hover:border-accent mb-4 inline-flex items-center gap-2 border px-3 py-2 text-sm transition-colors"
      >
        <span class="i-lucide-folder-git-2 text-muted text-sm" aria-hidden="true" />
        <span class="text-ink">{{ repoName }}</span>
        <span class="i-lucide-arrow-up-right text-muted text-sm" aria-hidden="true" />
      </RouterLink>

      <RoutineRunDetail
        :run="run"
        :resuming="resuming"
        :skipping="skipping"
        :confirming="confirmingDecision"
        :deleting="deleting"
        :cancelling="cancelling"
        @resume="onResume"
        @skip="onSkip"
        @confirm="onConfirm"
        @delete="onDelete"
        @cancel="onCancel"
      />
    </template>
  </div>
</template>
