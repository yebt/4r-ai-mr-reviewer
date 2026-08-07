<script setup lang="ts">
definePage({ meta: { title: 'Flow' } })
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api, errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import { useFlowRepo } from '@shared/composables/useFlowRepo'
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

// The Flow workspace: an attention-first per-repo cockpit. A searchable repo
// picker (shared with Cmd/Ctrl+K) loads a repo; a Needs-you band turns the repo's
// awaiting/failed reviews and paused routines into one-click actions; the tabs
// below drive reviews, releases and real inline settings.
const router = useRouter()
const repos = useReposStore()
const reviews = useReviewsStore()
const providers = useProvidersStore()
const routines = useRoutinesStore()

const { repoId: selectedRepoId } = useFlowRepo()
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
    syncSettingsForm()
  }
})

// --- Searchable repo picker (also drivable from Cmd/Ctrl+K) ---
const PAUSED = new Set(['awaiting_confirmation', 'blocked'])
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
  // Repos that need attention float up.
  return [...list].sort((a, b) => repoAttention(b.id) - repoAttention(a.id))
})
function openPicker() {
  pickerOpen.value = true
  pickerQuery.value = ''
  nextTick(() => pickerInput.value?.focus())
}
function pickRepo(id: string) {
  selectedRepoId.value = id
  pickerOpen.value = false
}
function onPickerEnter() {
  const first = filteredRepos.value[0]
  if (first) pickRepo(first.id)
}

// --- Per-repo attention ---
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
// Skip a failed review: archive it so its alert leaves the Needs-you band while
// its history stays in the archived list. Handy once you've dealt with (or
// deliberately ignored) a failure.
const skipFailed = (id: string) => withBusy(id, () => reviews.archive(id), 'Failure skipped — archived')
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

// --- Settings: inline provider/model + full webhook config ---
const settingsForm = ref({ providerId: '', model: '' })
function syncSettingsForm() {
  settingsForm.value = { providerId: repo.value?.providerId ?? '', model: repo.value?.model ?? '' }
}
watch(repo, syncSettingsForm, { immediate: true })
const providerModels = computed(() => {
  const p = providers.items.find((x) => x.id === settingsForm.value.providerId)
  return [...(p?.models ?? [])].sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
})
const providerDirty = computed(
  () =>
    settingsForm.value.providerId !== (repo.value?.providerId ?? '') ||
    settingsForm.value.model.trim() !== (repo.value?.model ?? ''),
)
const savingProvider = ref(false)
async function saveProvider() {
  if (!repo.value || !providerDirty.value) return
  savingProvider.value = true
  try {
    await repos.assign(repo.value.id, {
      providerId: settingsForm.value.providerId,
      model: settingsForm.value.model.trim(),
    })
    toast.success('Provider updated')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    savingProvider.value = false
  }
}

const webhookBusy = ref(false)
const showWebhookSecret = ref(false)
const webhookUrl = computed(() => (repo.value ? window.location.origin + repo.value.webhookPath : ''))
async function setWebhook(enabled: boolean, requireConfirmation: boolean) {
  if (!repo.value || webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.setWebhook(repo.value.id, enabled, requireConfirmation)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}
const toggleWebhook = () =>
  setWebhook(!repo.value?.webhookEnabled, repo.value?.webhookRequireConfirmation ?? false)
const toggleRequireConfirmation = () =>
  setWebhook(true, !repo.value?.webhookRequireConfirmation)
async function rotateWebhook() {
  if (!repo.value || webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.rotateWebhookSecret(repo.value.id)
    toast.success('Token rotated — update it in GitLab')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}
async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(`${label} copied`)
  } catch {
    toast.error('Copy failed — your browser blocked clipboard access')
  }
}

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
</script>

<template>
  <div>
    <PageHeader title="Flow" label="Workspace">
      <template #actions>
        <!-- Searchable repo picker -->
        <div class="relative">
          <button
            type="button"
            class="btn-line text-xs"
            aria-haspopup="listbox"
            :aria-expanded="pickerOpen"
            @click="pickerOpen ? (pickerOpen = false) : openPicker()"
          >
            <span class="i-lucide-folder-git-2 text-sm" aria-hidden="true" />
            {{ repo?.name ?? 'Select a repository' }}
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
                :aria-selected="r.id === selectedRepoId"
                class="flex cursor-pointer items-center gap-2 border-l-2 px-3 py-2 text-sm"
                :class="
                  r.id === selectedRepoId
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
      </template>
    </PageHeader>

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

    <EmptyState
      v-else-if="!repo"
      icon="i-lucide-mouse-pointer-click"
      title="Pick a repository"
      hint="Use the picker above (or Cmd/Ctrl+K) to load a repo and work its reviews, releases and settings here."
    />

    <template v-else>
      <div class="mb-6 flex flex-wrap items-center justify-between gap-x-6 gap-y-2">
        <div class="min-w-0">
          <div class="text-ink truncate text-lg font-semibold tracking-tight">{{ repo.name }}</div>
          <div class="text-muted truncate font-mono text-xs">{{ repo.url }}</div>
        </div>
        <div class="label-mono flex flex-wrap items-center gap-x-4 gap-y-1">
          <span>provider {{ providerName }}</span>
          <span v-if="repo.model">model {{ repo.model }}</span>
        </div>
      </div>

      <!-- Needs-you band -->
      <section
        v-if="attentionCount > 0"
        class="border-warn/40 bg-warn/5 mb-8 border-l-2 px-4 py-3"
        aria-labelledby="needs-you"
      >
        <h2 id="needs-you" class="label-mono text-warn mb-3">Needs you · {{ attentionCount }}</h2>
        <ul class="flex flex-col gap-2.5">
          <li v-for="rv in awaiting" :key="rv.id" class="flex flex-wrap items-center justify-between gap-2">
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
          <li v-for="rv in failed" :key="rv.id" class="flex flex-wrap items-center justify-between gap-2">
            <RouterLink :to="`/reviews/${rv.id}`" class="text-ink min-w-0 truncate text-sm hover:underline">
              Review <span class="font-mono">!{{ rv.mrIid }}</span> — <span class="text-danger">failed</span>
            </RouterLink>
            <div class="flex shrink-0 items-center gap-2">
              <button class="btn-line text-xs" :disabled="busyIds.includes(rv.id)" @click="retryReview(rv.id)">
                <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" /> Retry
              </button>
              <button
                class="btn-ghost text-xs"
                :disabled="busyIds.includes(rv.id)"
                title="Dismiss this alert — archives the failed review, keeping its history"
                @click="skipFailed(rv.id)"
              >
                Skip
              </button>
            </div>
          </li>
          <li v-for="run in pausedRuns" :key="run.id" class="flex flex-wrap items-center justify-between gap-2">
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

      <!-- Reviews -->
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

      <!-- Release -->
      <div v-show="work === 'release'">
        <RoutinesSection :repo-id="repo.id" :merge-requests="mrs" />
      </div>

      <!-- Settings: real inline controls -->
      <div v-show="work === 'settings'" class="flex flex-col gap-10">
        <section>
          <h2 class="section-title mb-3 flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            AI provider
          </h2>
          <div class="flex flex-col gap-4">
            <div>
              <label class="field-label" for="fl-provider">Provider</label>
              <select id="fl-provider" v-model="settingsForm.providerId" class="field-underline">
                <option value="">Use default provider</option>
                <option v-for="p in providers.items" :key="p.id" :value="p.id">
                  {{ p.name }}{{ p.isDefault ? ' (default)' : '' }}
                </option>
              </select>
            </div>
            <div>
              <label class="field-label" for="fl-model">
                Model <span class="text-muted normal-case">— optional, uses the provider's model</span>
              </label>
              <input
                id="fl-model"
                v-model="settingsForm.model"
                class="field-underline"
                list="fl-model-presets"
                placeholder="use provider's model"
                autocomplete="off"
              />
              <datalist id="fl-model-presets">
                <option v-for="m in providerModels" :key="m" :value="m" />
              </datalist>
            </div>
            <div class="flex items-center gap-3">
              <button class="btn-accent text-xs" :disabled="!providerDirty || savingProvider" @click="saveProvider">
                <span v-if="savingProvider" class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
                Save
              </button>
              <button v-if="providerDirty" class="btn-ghost text-xs" @click="syncSettingsForm">Reset</button>
            </div>
          </div>
        </section>

        <section>
          <h2 class="section-title mb-3 flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Webhook
          </h2>
          <div class="flex flex-col gap-4">
            <div class="row justify-between">
              <div class="min-w-0">
                <div class="text-ink text-sm">Auto-review on MR open/update</div>
                <div class="label-mono mt-0.5">
                  {{ repo.webhookEnabled ? 'GitLab events trigger a review' : 'Disabled' }}
                </div>
              </div>
              <button class="btn-line text-xs" :disabled="webhookBusy" @click="toggleWebhook">
                <span v-if="webhookBusy" class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
                <span
                  v-else
                  :class="repo.webhookEnabled ? 'i-lucide-toggle-right text-ok' : 'i-lucide-toggle-left'"
                  class="text-base"
                  aria-hidden="true"
                />
                {{ repo.webhookEnabled ? 'Enabled' : 'Disabled' }}
              </button>
            </div>

            <template v-if="repo.webhookEnabled">
              <div class="row justify-between">
                <div class="min-w-0">
                  <div class="text-ink text-sm">Require confirmation before running</div>
                  <div class="label-mono mt-0.5">
                    Webhook reviews are held for approval instead of running automatically.
                  </div>
                </div>
                <button class="btn-line text-xs" :disabled="webhookBusy" @click="toggleRequireConfirmation">
                  <span
                    :class="repo.webhookRequireConfirmation ? 'i-lucide-toggle-right text-ok' : 'i-lucide-toggle-left'"
                    class="text-base"
                    aria-hidden="true"
                  />
                  {{ repo.webhookRequireConfirmation ? 'On' : 'Off' }}
                </button>
              </div>

              <div>
                <span class="field-label">Webhook URL</span>
                <div class="flex items-center gap-2">
                  <code class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs">
                    {{ webhookUrl }}
                  </code>
                  <button class="btn-ghost text-xs" aria-label="Copy webhook URL" @click="copyText(webhookUrl, 'URL')">
                    <span class="i-lucide-copy text-sm" aria-hidden="true" /> Copy
                  </button>
                </div>
              </div>

              <div>
                <span class="field-label">Secret token</span>
                <div class="flex items-center gap-2">
                  <code class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs">
                    {{ showWebhookSecret ? repo.webhookSecret : '•'.repeat(24) }}
                  </code>
                  <button
                    class="btn-ghost text-xs"
                    :aria-label="showWebhookSecret ? 'Hide secret token' : 'Show secret token'"
                    @click="showWebhookSecret = !showWebhookSecret"
                  >
                    <span :class="showWebhookSecret ? 'i-lucide-eye-off' : 'i-lucide-eye'" class="text-sm" aria-hidden="true" />
                    {{ showWebhookSecret ? 'Hide' : 'Show' }}
                  </button>
                  <button class="btn-ghost text-xs" aria-label="Copy secret token" @click="copyText(repo.webhookSecret, 'Secret token')">
                    <span class="i-lucide-copy text-sm" aria-hidden="true" /> Copy
                  </button>
                  <button class="btn-ghost text-xs" :disabled="webhookBusy" aria-label="Rotate secret token" @click="rotateWebhook">
                    <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" /> Rotate
                  </button>
                </div>
                <p class="text-muted mt-1.5 text-xs">
                  Paste the URL and secret into GitLab → Settings → Webhooks. Rotating invalidates the
                  old token until you update it there.
                </p>
              </div>
            </template>
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
  </div>
</template>
