<script setup lang="ts">
definePage({ meta: { title: 'Flow' } })
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTitle } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import type { Review } from '@shared/api/types'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import MergeRequestList from '@modules/reviews/components/MergeRequestList.vue'
import ReviewList from '@modules/reviews/components/ReviewList.vue'
import RoutinesSection from '@modules/routines/components/RoutinesSection.vue'
import RepoSettingsModal from '@modules/repos/components/RepoSettingsModal.vue'
import ReleaseModal, { type ReleaseSubmit } from '@modules/routines/components/ReleaseModal.vue'
import { isTerminal } from '@modules/reviews/format'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useProvidersStore } from '@modules/providers/store'
import { useRoutinesStore } from '@modules/routines/store'

// The Flow workspace for a single repository: the repo lives in the URL
// (/flow/:repoId), so refresh, back and deep links all land on the same repo.
// A sticky three-tab bar (MRs · Reviews · Actions) drives the work; a header
// switcher jumps between repos and a gear opens the per-repo settings modal.
const route = useRoute()
const router = useRouter()
const repos = useReposStore()
const reviews = useReviewsStore()
const providers = useProvidersStore()
const routines = useRoutinesStore()

const repoId = computed(() => (route.params as { repoId: string }).repoId)
const repo = computed(() => repos.items.find((r) => r.id === repoId.value) ?? null)

// True once the initial repo list has settled, so a genuinely unknown repo id
// shows "not found" instead of flashing it while the list is still loading.
const ready = ref(false)

useTitle(computed(() => (repo.value ? `${repo.value.name} - Flow` : 'Flow')))

// --- Data loading (per repo; re-runs when the URL repo changes) ---
function loadRepoData(id: string) {
  void reviews.fetchReviews(id)
  void reviews.fetchMergeRequests(id)
  void routines.list(id)
}

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  providers.fetchAll()
  ready.value = true
  if (repo.value) loadRepoData(repoId.value)
})

// Switching repos navigates to a new /flow/:repoId; this component is reused, so
// react to the id change by (re)loading the new repo's data.
watch(repoId, (id) => {
  if (id && repo.value) loadRepoData(id)
})

// --- Tabs (synced to ?tab= so refresh/back keep the active tab) ---
type Tab = 'mrs' | 'reviews' | 'actions'
const TABS: { id: Tab; label: string; icon: string }[] = [
  { id: 'mrs', label: 'MRs', icon: 'i-lucide-git-pull-request' },
  { id: 'reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { id: 'actions', label: 'Actions', icon: 'i-lucide-git-merge' },
]
function isTab(v: unknown): v is Tab {
  return v === 'mrs' || v === 'reviews' || v === 'actions'
}
const tab = ref<Tab>(isTab(route.query.tab) ? route.query.tab : 'mrs')
watch(tab, (t) => {
  router.replace({ query: { ...route.query, tab: t } })
})

// --- Working data ---
const mrs = computed(() => reviews.mergeRequestsFor(repoId.value))
const mrsLoading = computed(() => reviews.mrsLoading && mrs.value.length === 0)
const repoReviews = computed(() => reviews.reviewsFor(repoId.value))
const reviewsLoading = computed(() => reviews.listLoading && repoReviews.value.length === 0)
const defaultProviderId = computed(
  () => repo.value?.providerId || providers.items.find((p) => p.isDefault)?.id || '',
)
const reviewByMr = computed<Record<number, Review>>(() => {
  const m: Record<number, Review> = {}
  for (const rv of repoReviews.value) {
    const cur = m[rv.mrIid]
    if (!cur || rv.createdAt > cur.createdAt) m[rv.mrIid] = rv
  }
  return m
})

// --- Attention badges ---
const PAUSED = new Set(['awaiting_confirmation', 'blocked'])
const reviewsBadge = computed(
  () => repoReviews.value.filter((r) => r.status === 'awaiting_approval' || r.status === 'error').length,
)
const actionsBadge = computed(
  () => routines.runsFor(repoId.value).filter((r) => PAUSED.has(r.status)).length,
)

// --- Header repo switcher (searchable picker → navigates to /flow/:id) ---
function repoAttention(id: string): number {
  const revs = reviews
    .reviewsFor(id)
    .filter((r) => r.status === 'awaiting_approval' || r.status === 'error').length
  const runs = routines.recentRuns.filter((r) => r.repoId === id && PAUSED.has(r.status)).length
  return revs + runs
}
const pickerOpen = ref(false)
const pickerQuery = ref('')
const pickerInput = ref<HTMLInputElement | null>(null)
const filteredRepos = computed(() => {
  const q = pickerQuery.value.trim().toLowerCase()
  const list = q ? repos.items.filter((r) => r.name.toLowerCase().includes(q)) : repos.items
  return [...list].sort((a, b) => repoAttention(b.id) - repoAttention(a.id))
})
function openPicker() {
  pickerOpen.value = true
  pickerQuery.value = ''
  nextTick(() => pickerInput.value?.focus())
}
function pickRepo(id: string) {
  pickerOpen.value = false
  if (id !== repoId.value) router.push(`/flow/${id}`)
}
function onPickerEnter() {
  const first = filteredRepos.value[0]
  if (first) pickRepo(first.id)
}

// --- Settings modal ---
const settingsOpen = ref(false)

// --- Reviews actions (busy-tracked, per-row) ---
const busyIds = ref<string[]>([])
async function withBusy(id: string, fn: () => Promise<unknown>, ok: string) {
  busyIds.value = [...busyIds.value, id]
  try {
    await fn()
    toast.success(ok)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}
const approveReview = (id: string) =>
  withBusy(id, () => reviews.approve(id), 'Review approved — running now')
const retryReview = (id: string) =>
  withBusy(id, () => reviews.retry(id), 'Retrying — a fresh review is running')
const archiveReview = (id: string) => withBusy(id, () => reviews.archive(id), 'Review archived')
async function discardReview(id: string, mrIid: number) {
  const ok = await confirm({
    title: 'Discard review',
    message: `Discard the held review for !${mrIid}? This cannot be undone.`,
    danger: true,
    confirmText: 'Discard',
  })
  if (!ok) return
  await withBusy(id, () => reviews.remove(id), 'Review discarded')
}

// --- Launch a review on an open MR ---
const creatingIid = ref<number | null>(null)
async function startReview(iid: number, mode: string, providerId: string, model: string) {
  let watchNow = true
  if (repoReviews.value.some((r) => !isTerminal(r.status))) {
    watchNow = await confirm({
      title: 'Reviews already running',
      message:
        'Other reviews are still running. Launch this one now and watch it (it runs in parallel), or queue it and keep working — it starts when a slot frees up.',
      confirmText: 'Launch and watch',
      cancelText: 'Queue and stay',
    })
  }
  creatingIid.value = iid
  try {
    const rv = await reviews.create(repoId.value, iid, mode, providerId, model)
    if (watchNow) router.push(`/reviews/${rv.id}`)
    else toast.success('Review queued — it will run when a slot is free.')
  } catch (e) {
    reviews.mrsError = errorMessage(e)
  } finally {
    creatingIid.value = null
  }
}

// --- Release trigger modal (dev + main flows) — logic mirrors RoutinesSection ---
const modalOpen = ref(false)
const modalFlow = ref<'dev' | 'main'>('dev')
const modalIid = ref<number | null>(null)
const submitting = ref(false)
const modalMergeRequest = computed(() => mrs.value.find((mr) => mr.iid === modalIid.value) ?? null)

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
        ? await routines.createMainRelease(repoId.value, payload.input)
        : await routines.createRelease(repoId.value, payload.input)
    modalOpen.value = false
    toast.success(
      payload.flow === 'main' ? 'Release to main started' : `Release started for !${run.mrIid}`,
    )
    router.push(`/actions/${run.id}`)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div>
    <!-- Unknown repo id (e.g. a stale deep link): offer the way back. -->
    <EmptyState
      v-if="ready && !repo"
      icon="i-lucide-folder-x"
      title="Repository not found"
      hint="This repository is not tracked, or the link is out of date."
    >
      <template #action>
        <RouterLink to="/flow" class="btn-line text-xs">
          <span class="i-lucide-arrow-left text-sm" aria-hidden="true" />
          Back to repositories
        </RouterLink>
      </template>
    </EmptyState>

    <template v-else-if="repo">
      <!-- Header: identity + repo switcher + settings -->
      <div class="mb-4 flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
        <div class="min-w-0">
          <div class="text-ink truncate text-lg font-semibold tracking-tight">{{ repo.name }}</div>
          <div class="text-muted truncate font-mono text-xs">{{ repo.url }}</div>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <!-- Searchable repo switcher -->
          <div class="relative">
            <button
              type="button"
              class="btn-line text-xs"
              aria-haspopup="listbox"
              :aria-expanded="pickerOpen"
              @click="pickerOpen ? (pickerOpen = false) : openPicker()"
            >
              <span class="i-lucide-folder-git-2 text-sm" aria-hidden="true" />
              Switch repo
              <span class="i-lucide-chevron-down text-sm" aria-hidden="true" />
            </button>
            <div
              v-if="pickerOpen"
              class="border-line bg-surface absolute right-0 z-30 mt-2 flex max-h-80 w-72 flex-col overflow-hidden border shadow-lg shadow-black/40"
            >
              <div class="border-line/60 flex items-center gap-2 border-b px-3">
                <span class="i-lucide-search text-muted shrink-0 text-sm" aria-hidden="true" />
                <input
                  ref="pickerInput"
                  v-model="pickerQuery"
                  type="text"
                  class="text-ink placeholder:text-muted/60 w-full bg-transparent py-2.5 text-sm outline-none"
                  placeholder="Search repositories…"
                  autocomplete="off"
                  @keydown.enter.prevent="onPickerEnter"
                  @keydown.escape="pickerOpen = false"
                />
              </div>
              <ul class="min-h-0 flex-1 overflow-y-auto py-1" role="listbox">
                <li
                  v-for="r in filteredRepos"
                  :key="r.id"
                  role="option"
                  :aria-selected="r.id === repoId"
                  class="flex cursor-pointer items-center gap-2 border-l-2 px-3 py-2 text-sm"
                  :class="
                    r.id === repoId
                      ? 'border-accent bg-accent/10 text-ink'
                      : 'text-muted hover:text-ink border-transparent'
                  "
                  @click="pickRepo(r.id)"
                >
                  <span class="min-w-0 flex-1 truncate">{{ r.name }}</span>
                  <span
                    v-if="repoAttention(r.id) > 0"
                    class="bg-warn/15 text-warn inline-flex min-w-4 items-center justify-center px-1 font-mono text-[0.6rem]"
                  >
                    {{ repoAttention(r.id) }}
                  </span>
                </li>
                <li v-if="filteredRepos.length === 0" class="text-muted px-3 py-3 text-xs">
                  No repositories match.
                </li>
              </ul>
            </div>
            <!-- click-away -->
            <div v-if="pickerOpen" class="fixed inset-0 z-20" @click="pickerOpen = false" />
          </div>

          <button
            type="button"
            class="btn-line text-xs"
            aria-label="Repository settings"
            @click="settingsOpen = true"
          >
            <span class="i-lucide-settings text-sm" aria-hidden="true" />
            Settings
          </button>
        </div>
      </div>

      <!-- Sticky tab bar: stays put on scroll (matters on mobile). Bleeds to the
           content edges with a solid page background so rows never show through. -->
      <div
        class="bg-canvas border-line/50 sticky top-0 z-10 -mx-4 mb-6 flex gap-1 border-b px-4 md:-mx-8 md:px-8"
        role="tablist"
        aria-label="Workspace"
      >
        <button
          v-for="t in TABS"
          :key="t.id"
          type="button"
          role="tab"
          :aria-selected="tab === t.id"
          class="-mb-px inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
          :class="tab === t.id ? 'border-accent text-ink' : 'text-muted hover:text-ink border-transparent'"
          @click="tab = t.id"
        >
          <span :class="t.icon" class="text-sm" aria-hidden="true" />
          {{ t.label }}
          <span
            v-if="(t.id === 'reviews' && reviewsBadge > 0) || (t.id === 'actions' && actionsBadge > 0)"
            class="bg-warn/15 text-warn inline-flex min-w-4 items-center justify-center px-1 font-mono text-[0.6rem]"
          >
            {{ t.id === 'reviews' ? reviewsBadge : actionsBadge }}
          </span>
        </button>
      </div>

      <!-- MRs -->
      <div v-show="tab === 'mrs'">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h2 class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Open merge requests
          </h2>
          <button type="button" class="btn-line text-xs" @click="openMainRelease">
            <span class="i-lucide-rocket text-sm" aria-hidden="true" />
            Release to main
          </button>
        </div>
        <MergeRequestList
          :items="mrs"
          :loading="mrsLoading"
          :error="reviews.mrsError"
          :busy-iid="creatingIid"
          :providers="providers.items"
          :default-provider-id="defaultProviderId"
          :review-by-mr="reviewByMr"
          show-release
          @review="startReview"
          @release="openDevRelease"
        />
      </div>

      <!-- Reviews -->
      <div v-show="tab === 'reviews'">
        <h2 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Reviews
        </h2>
        <ReviewList
          :items="repoReviews"
          :loading="reviewsLoading"
          :error="reviews.listError"
          :busy-ids="busyIds"
          @archive="archiveReview"
          @approve="approveReview"
          @discard="discardReview"
          @retry="retryReview"
        />
      </div>

      <!-- Actions -->
      <div v-show="tab === 'actions'">
        <RoutinesSection :repo-id="repoId" :merge-requests="mrs" :show-triggers="false" />
      </div>

      <RepoSettingsModal :open="settingsOpen" :repo="repo" @close="settingsOpen = false" />
      <ReleaseModal
        :open="modalOpen"
        :flow="modalFlow"
        :repo-id="repoId"
        :merge-request="modalMergeRequest"
        :submitting="submitting"
        @submit="submit"
        @close="closeModal"
      />
    </template>
  </div>
</template>
