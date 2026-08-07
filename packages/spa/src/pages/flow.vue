<script setup lang="ts">
definePage({ meta: { title: 'Flow' } })
import { computed, onMounted, ref, watch } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { api, errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import type { Preflight, Review, RoutineRun } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import MergeRequestList from '@modules/reviews/components/MergeRequestList.vue'
import ReviewList from '@modules/reviews/components/ReviewList.vue'
import PreflightReport from '@modules/repos/components/PreflightReport.vue'
import RoutinesSection from '@modules/routines/components/RoutinesSection.vue'
import { isTerminal } from '@modules/reviews/format'
import { runTitle } from '@modules/routines/format'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useProvidersStore } from '@modules/providers/store'
import { useRoutinesStore } from '@modules/routines/store'

// The Flow workspace: an attention-first cockpit. The repo strip shows which
// repos need you at a glance (a badge counts awaiting-approval/failed reviews and
// paused routines); the active repo leads with a "Needs you" band that turns those
// same items into one-click actions, then the working area below. The active repo
// persists across visits. Reviews for every repo are loaded once so the badges are
// meaningful before you click.
const router = useRouter()
const repos = useReposStore()
const reviews = useReviewsStore()
const providers = useProvidersStore()
const routines = useRoutinesStore()

const selectedRepoId = useLocalStorage('flow.repoId', '')
const repo = computed(() => repos.items.find((r) => r.id === selectedRepoId.value) ?? null)
const loading = ref(true)

const work = ref<'reviews' | 'release' | 'settings'>('reviews')
const WORK_TABS = [
  { id: 'reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { id: 'release', label: 'Release', icon: 'i-lucide-git-merge' },
  { id: 'settings', label: 'Settings', icon: 'i-lucide-settings' },
] as const

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  providers.fetchAll()
  if (selectedRepoId.value && !repo.value) selectedRepoId.value = ''
  // Load reviews for every repo + recent routines so the repo strip can badge
  // which repos need attention before the user picks one.
  await Promise.all([
    ...repos.items.map((r) => reviews.fetchReviews(r.id)),
    routines.listRecent(50),
  ])
  loading.value = false
  if (repo.value) void reviews.fetchMergeRequests(repo.value.id)
})

watch(selectedRepoId, (id) => {
  if (id) {
    void reviews.fetchMergeRequests(id)
    preflight.value = null
    preflightError.value = null
    work.value = 'reviews'
  }
})

// --- Per-repo attention (drives both the strip badges and the Needs-you band) ---
const PAUSED = new Set(['awaiting_confirmation', 'blocked'])
function repoAttention(id: string): number {
  const revs = reviews
    .reviewsFor(id)
    .filter((r) => r.status === 'awaiting_approval' || r.status === 'error').length
  const runs = routines.recentRuns.filter((r) => r.repoId === id && PAUSED.has(r.status)).length
  return revs + runs
}

const repoId = computed(() => repo.value?.id ?? '')
const repoReviews = computed(() => (repoId.value ? reviews.reviewsFor(repoId.value) : []))
const awaiting = computed(() => repoReviews.value.filter((r) => r.status === 'awaiting_approval'))
const failed = computed(() => repoReviews.value.filter((r) => r.status === 'error'))
const pausedRuns = computed<RoutineRun[]>(() =>
  routines.recentRuns.filter((r) => r.repoId === repoId.value && PAUSED.has(r.status)),
)
const attentionCount = computed(
  () => awaiting.value.length + failed.value.length + pausedRuns.value.length,
)

// --- Reviews working data ---
const mrs = computed(() => (repoId.value ? reviews.mergeRequestsFor(repoId.value) : []))
const mrsLoading = computed(() => reviews.mrsLoading && mrs.value.length === 0)
const reviewsLoading = computed(() => reviews.listLoading && repoReviews.value.length === 0)
const defaultProviderId = computed(
  () => repo.value?.providerId || providers.items.find((p) => p.isDefault)?.id || '',
)
const providerName = computed(() => {
  const id = repo.value?.providerId
  if (!id) return 'default provider'
  return providers.items.find((p) => p.id === id)?.name ?? 'default provider'
})
const reviewByMr = computed<Record<number, Review>>(() => {
  const m: Record<number, Review> = {}
  for (const rv of repoReviews.value) {
    const cur = m[rv.mrIid]
    if (!cur || rv.createdAt > cur.createdAt) m[rv.mrIid] = rv
  }
  return m
})

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
  })
  if (!ok) return
  await withBusy(id, () => reviews.remove(id), 'Review discarded')
}

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

// --- Settings working data ---
const preflight = ref<Preflight | null>(null)
const preflightLoading = ref(false)
const preflightError = ref<string | null>(null)
async function runPreflight() {
  if (!repoId.value) return
  preflightLoading.value = true
  preflightError.value = null
  try {
    preflight.value = await api.preflightRepo(repoId.value)
  } catch (e) {
    preflightError.value = errorMessage(e)
  } finally {
    preflightLoading.value = false
  }
}
const webhookBusy = ref(false)
async function toggleWebhook() {
  if (!repo.value) return
  webhookBusy.value = true
  try {
    await repos.setWebhook(repo.value.id, !repo.value.webhookEnabled)
    toast.success(repo.value.webhookEnabled ? 'Webhook enabled' : 'Webhook disabled')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="Flow" label="Workspace" />

    <!-- No repos at all -->
    <EmptyState
      v-if="repos.items.length === 0 && !loading"
      icon="i-lucide-folder-git-2"
      title="No repositories tracked"
      hint="Track a GitLab repository to start reviewing its merge requests and cutting releases."
    >
      <template #action>
        <RouterLink to="/repos" class="btn-accent text-xs">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Track a repository
        </RouterLink>
      </template>
    </EmptyState>

    <template v-else>
      <!-- Repo switcher strip: which repo needs you, at a glance. A warn dot +
           count marks repos with awaiting-approval/failed reviews or paused
           routines, so switching is informed, not blind. -->
      <div
        class="border-line/50 -mx-1 mb-6 flex gap-2 overflow-x-auto border-b px-1 pb-3"
        role="tablist"
        aria-label="Repositories"
      >
        <button
          v-for="r in repos.items"
          :key="r.id"
          type="button"
          role="tab"
          :aria-selected="r.id === selectedRepoId"
          class="flex shrink-0 items-center gap-2 border-b-2 px-3 py-1.5 text-sm transition-colors"
          :class="
            r.id === selectedRepoId
              ? 'border-accent text-ink'
              : 'text-muted hover:text-ink border-transparent'
          "
          @click="selectedRepoId = r.id"
        >
          {{ r.name }}
          <span
            v-if="repoAttention(r.id) > 0"
            class="bg-warn/15 text-warn inline-flex min-w-4 items-center justify-center px-1 font-mono text-[0.6rem]"
            :title="`${repoAttention(r.id)} item(s) need you`"
          >
            {{ repoAttention(r.id) }}
          </span>
        </button>
      </div>

      <!-- Nothing picked yet -->
      <EmptyState
        v-if="!repo"
        icon="i-lucide-mouse-pointer-click"
        title="Pick a repository"
        hint="Choose a repo above to work its reviews, releases and settings — all in one place."
      />

      <template v-else>
        <!-- Context line -->
        <div
          class="mb-6 flex flex-wrap items-center justify-between gap-x-6 gap-y-2"
        >
          <div class="min-w-0">
            <div class="text-ink truncate text-lg font-semibold tracking-tight">{{ repo.name }}</div>
            <div class="text-muted truncate font-mono text-xs">{{ repo.url }}</div>
          </div>
          <div class="label-mono flex flex-wrap items-center gap-x-4 gap-y-1">
            <span>provider {{ providerName }}</span>
            <span v-if="repo.model">model {{ repo.model }}</span>
          </div>
        </div>

        <!-- Needs-you band: the whole point of the workspace. Awaiting-approval
             reviews, failed reviews and paused routines, actionable inline. -->
        <section
          v-if="attentionCount > 0"
          class="border-warn/40 bg-warn/5 mb-8 border-l-2 px-4 py-3"
          aria-labelledby="needs-you"
        >
          <h2 id="needs-you" class="label-mono text-warn mb-3">Needs you · {{ attentionCount }}</h2>
          <ul class="flex flex-col gap-2.5">
            <li
              v-for="rv in awaiting"
              :key="rv.id"
              class="flex flex-wrap items-center justify-between gap-2"
            >
              <RouterLink :to="`/reviews/${rv.id}`" class="text-ink min-w-0 truncate text-sm hover:underline">
                Review <span class="font-mono">!{{ rv.mrIid }}</span> — awaiting your approval
              </RouterLink>
              <div class="flex shrink-0 items-center gap-2">
                <button class="btn-line text-xs" :disabled="busyIds.includes(rv.id)" @click="approveReview(rv.id)">
                  <span class="i-lucide-play text-sm" aria-hidden="true" /> Approve
                </button>
                <button class="btn-ghost text-danger text-xs" :disabled="busyIds.includes(rv.id)" @click="discardReview(rv.id, rv.mrIid)">
                  Discard
                </button>
              </div>
            </li>
            <li
              v-for="rv in failed"
              :key="rv.id"
              class="flex flex-wrap items-center justify-between gap-2"
            >
              <RouterLink :to="`/reviews/${rv.id}`" class="text-ink min-w-0 truncate text-sm hover:underline">
                Review <span class="font-mono">!{{ rv.mrIid }}</span> — <span class="text-danger">failed</span>
              </RouterLink>
              <button class="btn-line text-xs" :disabled="busyIds.includes(rv.id)" @click="retryReview(rv.id)">
                <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" /> Retry
              </button>
            </li>
            <li
              v-for="run in pausedRuns"
              :key="run.id"
              class="flex flex-wrap items-center justify-between gap-2"
            >
              <span class="text-ink min-w-0 truncate text-sm">
                Release <span class="font-mono">{{ runTitle(run) }}</span> — paused
              </span>
              <RouterLink :to="`/actions/${run.id}`" class="btn-line text-xs">
                Open <span class="i-lucide-arrow-right text-sm" aria-hidden="true" />
              </RouterLink>
            </li>
          </ul>
        </section>
        <p v-else class="text-muted mb-8 flex items-center gap-2 text-sm">
          <span class="i-lucide-circle-check-big text-ok text-base" aria-hidden="true" />
          Nothing needs you on {{ repo.name }} right now.
        </p>

        <!-- Working-area switch -->
        <div class="border-line/50 mb-6 flex gap-1 border-b" role="tablist" aria-label="Working area">
          <button
            v-for="t in WORK_TABS"
            :key="t.id"
            type="button"
            role="tab"
            :aria-selected="work === t.id"
            class="-mb-px inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
            :class="work === t.id ? 'border-accent text-ink' : 'text-muted hover:text-ink border-transparent'"
            @click="work = t.id"
          >
            <span :class="t.icon" class="text-sm" aria-hidden="true" />
            {{ t.label }}
          </button>
        </div>

        <!-- Reviews: launch on open MRs + the repo's reviews -->
        <div v-show="work === 'reviews'">
          <section class="mb-10">
            <h2 class="section-title mb-3 flex items-center gap-2">
              <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
              Open merge requests
            </h2>
            <MergeRequestList
              :items="mrs"
              :loading="mrsLoading"
              :error="reviews.mrsError"
              :busy-iid="creatingIid"
              :providers="providers.items"
              :default-provider-id="defaultProviderId"
              :review-by-mr="reviewByMr"
              @review="startReview"
            />
          </section>
          <section>
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
            />
          </section>
        </div>

        <!-- Release: the self-contained routines section (with the tag preview) -->
        <div v-show="work === 'release'">
          <RoutinesSection :repo-id="repo.id" :merge-requests="mrs" />
        </div>

        <!-- Settings: the per-repo controls you touch most, inline -->
        <div v-show="work === 'settings'" class="flex flex-col gap-10">
          <section>
            <h2 class="section-title mb-3 flex items-center gap-2">
              <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
              AI provider
            </h2>
            <div class="row justify-between">
              <div class="min-w-0">
                <div class="text-ink text-sm">{{ providerName }}</div>
                <div class="label-mono mt-0.5">{{ repo.model || 'provider default model' }}</div>
              </div>
              <RouterLink :to="`/repos/${repo.id}`" class="btn-line text-xs">
                <span class="i-lucide-pencil text-sm" aria-hidden="true" /> Change
              </RouterLink>
            </div>
          </section>
          <section>
            <h2 class="section-title mb-3 flex items-center gap-2">
              <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
              Webhook
            </h2>
            <div class="row justify-between">
              <div class="min-w-0">
                <div class="text-ink text-sm">
                  Auto-review on MR open/update —
                  <span :class="repo.webhookEnabled ? 'text-ok' : 'text-muted'">
                    {{ repo.webhookEnabled ? 'enabled' : 'disabled' }}
                  </span>
                </div>
                <div class="label-mono mt-0.5">Manage the secret in the repo's Webhook tab.</div>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button class="btn-line text-xs" :disabled="webhookBusy" @click="toggleWebhook">
                  <span v-if="webhookBusy" class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
                  {{ repo.webhookEnabled ? 'Disable' : 'Enable' }}
                </button>
                <RouterLink :to="`/repos/${repo.id}`" class="btn-ghost text-xs">Details</RouterLink>
              </div>
            </div>
          </section>
          <section>
            <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
              <h2 class="section-title flex items-center gap-2">
                <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
                API scope
              </h2>
              <button class="btn-line text-xs" :disabled="preflightLoading" @click="runPreflight">
                <span
                  :class="preflightLoading ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-shield-check'"
                  class="text-sm"
                  aria-hidden="true"
                />
                {{ preflightLoading ? 'Testing…' : 'Test API scope' }}
              </button>
            </div>
            <p v-if="preflightError" class="text-danger py-1 text-sm" role="alert">{{ preflightError }}</p>
            <PreflightReport v-else-if="preflight" :report="preflight" />
            <p v-else class="text-muted text-sm">
              Check which routine and review actions this repo's token and access level permit.
            </p>
          </section>
        </div>
      </template>
    </template>
  </div>
</template>
