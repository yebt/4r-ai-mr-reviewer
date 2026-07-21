<script setup lang="ts">
import { computed, onUnmounted, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn, useLocalStorage } from '@vueuse/core'
import { useIsPhone } from '@shared/composables/useIsPhone'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Alert from '@shared/components/ui/Alert.vue'
import Modal from '@shared/components/ui/Modal.vue'
import { useReviewsStore } from '@modules/reviews/store'
import { useReposStore } from '@modules/repos/store'
import { useProfilesStore } from '@modules/profiles/store'
import { isTerminal } from '@modules/reviews/format'
import { useReviewNotification } from '@modules/reviews/useReviewNotification'
import { ORIGINAL, buildFindingBody, buildFindingMarkdown } from '@modules/reviews/humanize-overrides'
import type { HumanizeFindingText } from '@shared/api/types'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import SummaryCard from '@modules/reviews/components/SummaryCard.vue'
import FindingCard from '@modules/reviews/components/FindingCard.vue'
import FindingCardTriage from '@modules/reviews/components/FindingCardTriage.vue'
import FindingsToolbar from '@modules/reviews/components/FindingsToolbar.vue'
import { useFindingFilters } from '@modules/reviews/useFindingFilters'

const route = useRoute()
const router = useRouter()
const store = useReviewsStore()
const repos = useReposStore()
const profiles = useProfilesStore()

// Only profiles whose style guide finished distilling can rewrite text.
const readyProfiles = computed(() => profiles.items.filter((p) => p.styleGuideStatus === 'ready'))
// Globally selected humanization profile — drives every card's Humanize button
// and "Humanize all". Defaults to the first ready profile once they load.
const profileId = ref('')
watch(readyProfiles, (list) => {
  if (list.length && !list.some((p) => p.id === profileId.value)) profileId.value = list[0]?.id ?? ''
})

const humanizingAll = ref(false)

// Humanize is unusable when there are no ready profiles OR none is selected.
// Drives the phone warning Alert and the phone toggle's warn affordance.
const humanizeUnavailable = computed(() => !readyProfiles.value.length || !profileId.value)

// Fires humanize for the summary and every finding concurrently with the
// selected profile. Each job is independent (per-card spinners show progress);
// per-card failures toast without aborting the rest.
async function humanizeAll() {
  const rv = review.value
  if (!rv || !profileId.value || humanizingAll.value) return
  humanizingAll.value = true
  try {
    const jobs = [
      store.humanizeSummary(rv.id, profileId.value),
      ...rv.findings.map((f) => store.humanizeFinding(rv.id, profileId.value, f.index)),
    ]
    const results = await Promise.allSettled(jobs)
    if (results.some((r) => r.status === 'rejected')) toast.error('Some cards failed to humanize.')
  } finally {
    humanizingAll.value = false
  }
}

const reviewId = computed(() => (route.params as { id: string }).id)
const review = computed(() => store.current)

const crumbs = computed(() => {
  const items: { label: string; to?: string }[] = [{ label: 'Repositories', to: '/repos' }]
  const rv = review.value
  if (rv) {
    const repo = repos.items.find((r) => r.id === rv.repoId)
    items.push({ label: repo?.name ?? 'Repository', to: `/repos/${rv.repoId}` })
    items.push({ label: `Review !${rv.mrIid}` })
  }
  return items
})

watchEffect(() => setBreadcrumbs(crumbs.value))

const selected = ref<number[]>([])
const publishing = ref(false)
const publishError = ref<string | null>(null)

// The summary note posts on the first publish by default and is re-selectable
// afterwards. Reset the default whenever the review's posted state changes (e.g.
// after a publish refresh flips summaryPublished): checked when not yet posted.
const includeSummary = ref(true)
watch(
  () => review.value?.summaryPublished,
  (posted) => {
    includeSummary.value = !posted
  },
  { immediate: true },
)

const unpublished = computed(() => review.value?.findings.filter((f) => !f.published) ?? [])

// Findings view mode, persisted so the choice survives navigation/reload. The
// triage view is the default; "classic" keeps the original flat list intact.
const findingsView = useLocalStorage<'classic' | 'triage'>('reviews:findingsView', 'triage')

// Triage state (filters/sort/counts/visible) over the review's findings.
const triage = useFindingFilters(() => review.value?.findings ?? [])

// Phone-only progressive disclosure. On desktop (`!isPhone`) the humanize bar and
// filter toolbar always render, so these refs only affect the < sm layout.
const isPhone = useIsPhone()
const humanizeOpen = ref(false)
const filtersOpen = ref(false)

// Count of active filter dimensions for the phone Filters toggle badge.
const activeFilterCount = computed(
  () =>
    triage.filters.dimensions.size +
    triage.filters.severities.size +
    (triage.filters.blockingOnly ? 1 : 0) +
    (triage.filters.status !== 'all' ? 1 : 0),
)

// Visible findings that can still be published (used by "Select all visible").
const selectableVisible = computed(() => triage.visible.value.filter((f) => !f.published))

// Add every currently-visible unpublished finding to the selection (union, so a
// prior selection outside the current filter is preserved).
function selectAllVisible() {
  for (const f of selectableVisible.value) {
    if (!selected.value.includes(f.index)) selected.value.push(f.index)
  }
}

function clearSelection() {
  selected.value = []
}

// Copy every selected finding to the clipboard as one Markdown block, respecting
// each card's active tab (Original or a humanize run) — so you can carry several
// findings at once. Findings are emitted in their natural index order.
async function copySelected() {
  const rv = review.value
  if (!rv || selected.value.length === 0) return
  const blocks: string[] = []
  for (const finding of rv.findings) {
    if (!selected.value.includes(finding.index)) continue
    const tab = store.selectedFindingTab(rv.id, finding.index)
    const original = { issue: finding.issue, why: finding.why, fix: finding.fix }
    const shown = tab === ORIGINAL ? original : (store.findingTabs(rv.id, finding.index)[tab] ?? original)
    blocks.push(buildFindingMarkdown(finding, shown))
  }
  try {
    await navigator.clipboard.writeText(blocks.join('\n\n---\n\n'))
    toast.success(`Copied ${blocks.length} finding${blocks.length === 1 ? '' : 's'} as Markdown`)
  } catch {
    toast.error('Copy failed — clipboard needs a secure context (https/localhost)')
  }
}

// Multi-pass progress: maps the backend phase to a labelled step (of 4).
const phaseMap: Record<string, { label: string; step: number }> = {
  risk: { label: 'Risk', step: 1 },
  readability: { label: 'Readability', step: 2 },
  reliability: { label: 'Reliability', step: 3 },
  resilience: { label: 'Resilience', step: 4 },
}
const phase = computed(() => (review.value?.phase ? (phaseMap[review.value.phase] ?? null) : null))

// Notify when a review the user is watching finishes. sawInProgress guards
// against notifying for a review that was already terminal when opened; it is
// reset when navigating to another review (see the reviewId watch below).
const {
  supported: notifSupported,
  permission: notifPermission,
  enabled: notifEnabled,
  enable: enableNotifications,
  notify: notifyDone,
} = useReviewNotification()
let sawInProgress = false

function notificationBody(status: string): string {
  if (status === 'done') {
    const n = review.value?.findings.length ?? 0
    return `Review finished — ${n} finding${n === 1 ? '' : 's'}.`
  }
  return status === 'error' ? 'Review failed.' : 'Review cancelled.'
}

watch(
  () => review.value?.status,
  (status) => {
    if (!status) return
    if (status === 'pending' || status === 'running') {
      sawInProgress = true
      return
    }
    if (!sawInProgress || !isTerminal(status)) return
    sawInProgress = false
    const title = `Review !${review.value?.mrIid ?? ''} ${status}`
    const body = notificationBody(status)
    // Desktop notification when the tab is hidden; a toast when it is focused.
    if (!notifyDone(title, body)) {
      if (status === 'error') toast.error(body)
      else toast.success(body)
    }
  },
)

const { pause, resume } = useIntervalFn(
  async () => {
    await store.refresh(reviewId.value)
    if (review.value && isTerminal(review.value.status)) pause()
  },
  2500,
  { immediate: false },
)

watch(
  reviewId,
  async (id) => {
    pause()
    sawInProgress = false
    selected.value = []
    if (repos.items.length === 0) repos.fetchAll()
    if (profiles.items.length === 0) profiles.fetchAll()
    await store.load(id)
    if (review.value && !isTerminal(review.value.status)) resume()
  },
  { immediate: true },
)
onUnmounted(pause)

function toggle(index: number) {
  const at = selected.value.indexOf(index)
  if (at >= 0) selected.value.splice(at, 1)
  else selected.value.push(index)
}

type PublishPayload = {
  all?: boolean
  indices?: number[]
  includeSummary?: boolean
  summaryOverride?: string
  findingOverrides?: HumanizeFindingText[]
}

// When a bulk publish includes the summary note, honor the summary version the
// user picked in SummaryCard (Original vs a humanized tab). Bulk publish only
// carried includeSummary, so it always posted the generated summary; this
// forwards the selected humanized tab's text as summaryOverride — matching
// SummaryCard's own per-card publish.
function withSummaryOverride(payload: PublishPayload): PublishPayload {
  if (!payload.includeSummary) return payload
  const tab = store.selectedSummaryTab(reviewId.value)
  if (tab === ORIGINAL) return payload
  const text = store.summaryTabs(reviewId.value)[tab]?.summary
  return text ? { ...payload, summaryOverride: text } : payload
}

// When a bulk publish targets findings, honor the humanized version the user
// picked per card (Original vs a V1/V2 tab). Bulk publish carried only indices,
// so selected findings posted the generated (robot) text; this forwards each
// finding's selected humanized tab as a findingOverride — matching the per-card
// Publish button. For an "all" publish the target set is the unpublished findings
// (what the backend's resolveIndices posts).
function withFindingOverrides(payload: PublishPayload): PublishPayload {
  const rv = review.value
  if (!rv) return payload
  const targets = payload.all
    ? rv.findings.filter((f) => !f.published).map((f) => f.index)
    : (payload.indices ?? [])
  if (targets.length === 0) return payload

  const overrides: HumanizeFindingText[] = []
  for (const index of targets) {
    const tab = store.selectedFindingTab(rv.id, index)
    if (tab === ORIGINAL) continue
    const humanized = store.findingTabs(rv.id, index)[tab]
    const finding = rv.findings.find((f) => f.index === index)
    if (humanized && finding) {
      overrides.push({ index, text: buildFindingBody(finding, humanized) })
    }
  }
  return overrides.length ? { ...payload, findingOverrides: overrides } : payload
}

async function publish(payload: PublishPayload) {
  publishing.value = true
  publishError.value = null
  try {
    await store.publish(reviewId.value, withFindingOverrides(withSummaryOverride(payload)))
    selected.value = []
    toast.success('Findings published')
  } catch (e) {
    publishError.value = errorMessage(e)
  } finally {
    publishing.value = false
  }
}

// Phone-only publish confirmation. On phone the "Include summary note" checkbox
// is hidden from the controls and instead surfaced inside a confirm modal, so a
// publish first opens the modal (askPublish) and only fires on confirm. Desktop
// keeps calling publish() directly with the inline includeSummary value.
const publishConfirmOpen = ref(false)
const pendingPublish = ref<{ all?: boolean; indices?: number[] } | null>(null)
const confirmIncludeSummary = ref(true)

function askPublish(payload: { all?: boolean; indices?: number[] }) {
  pendingPublish.value = payload
  confirmIncludeSummary.value = includeSummary.value
  publishConfirmOpen.value = true
}

async function confirmPublish() {
  if (!pendingPublish.value) return
  await publish({ ...pendingPublish.value, includeSummary: confirmIncludeSummary.value })
  publishConfirmOpen.value = false
  pendingPublish.value = null
}

// "Comment all" is shared between phone and desktop: phone routes through the
// confirm modal, desktop publishes immediately as before.
function onCommentAll() {
  if (isPhone.value) askPublish({ all: true })
  else publish({ all: true, includeSummary: includeSummary.value })
}

async function retry() {
  try {
    const rv = await store.retry(reviewId.value)
    router.push(`/reviews/${rv.id}`)
  } catch (e) {
    store.currentError = errorMessage(e)
  }
}

const cancelling = ref(false)

async function cancel() {
  cancelling.value = true
  try {
    await store.cancel(reviewId.value)
    toast.success('Cancelling review…')
    // Keep polling until the backend flips it to the cancelled terminal state.
    if (review.value && !isTerminal(review.value.status)) resume()
  } catch (e) {
    store.currentError = errorMessage(e)
  } finally {
    cancelling.value = false
  }
}

const archiving = ref(false)

async function toggleArchive() {
  const wasArchived = review.value?.archived
  archiving.value = true
  try {
    if (wasArchived) await store.unarchive(reviewId.value)
    else await store.archive(reviewId.value)
    toast.success(wasArchived ? 'Review unarchived' : 'Review archived')
    await store.refresh(reviewId.value)
  } catch (e) {
    store.currentError = errorMessage(e)
  } finally {
    archiving.value = false
  }
}

const deleting = ref(false)

async function remove() {
  const ok = await confirm({
    title: 'Delete review',
    message: 'Delete this review and all its findings? This cannot be undone.',
    danger: true,
  })
  if (!ok) return

  deleting.value = true
  const repoId = review.value?.repoId
  try {
    await store.remove(reviewId.value)
    toast.success('Review deleted')
    router.push(repoId ? `/repos/${repoId}` : '/repos')
  } catch (e) {
    store.currentError = errorMessage(e)
    deleting.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader :title="review ? `Merge request !${review.mrIid}` : 'Review'">
      <template #actions>
        <ReviewStatusChip v-if="review" :status="review.status" />
        <button
          v-if="review"
          class="btn-ghost text-xs"
          :disabled="archiving"
          @click="toggleArchive"
        >
          <span
            v-if="archiving"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span
            v-else
            :class="review.archived ? 'i-lucide-archive-restore' : 'i-lucide-archive'"
            class="text-sm"
            aria-hidden="true"
          />
          {{ review.archived ? 'Unarchive' : 'Archive' }}
        </button>
        <button
          v-if="review"
          class="btn-ghost text-danger text-xs"
          :disabled="deleting"
          @click="remove"
        >
          <span
            v-if="deleting"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          Delete
        </button>
      </template>
    </PageHeader>

    <p v-if="store.currentLoading && !review" class="text-muted text-sm">Loading…</p>
    <p v-else-if="store.currentError" class="text-danger text-sm">{{ store.currentError }}</p>

    <template v-else-if="review">
      <div v-if="review.status === 'pending' || review.status === 'running'">
        <div class="text-muted flex items-center gap-2 text-sm">
          <span class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
          <template v-if="phase">
            Reviewing {{ phase.label }}
            <span class="text-muted/70 font-mono">({{ phase.step }}/4)</span>…
          </template>
          <template v-else>Review {{ review.status }}… updates automatically.</template>
        </div>
        <div v-if="phase" class="bg-line/40 mt-3 h-1 w-full max-w-xs">
          <div
            class="bg-accent h-full transition-all"
            :style="{ width: `${(phase.step / 4) * 100}%` }"
          />
        </div>
        <div class="mt-4 flex flex-wrap items-center gap-2">
          <button class="btn-line text-xs" :disabled="cancelling" @click="cancel">
            <span
              v-if="cancelling"
              class="i-lucide-loader-circle animate-spin text-sm"
              aria-hidden="true"
            />
            <span v-else class="i-lucide-ban text-sm" aria-hidden="true" />
            Cancel review
          </button>
          <button
            v-if="notifSupported && !(notifEnabled && notifPermission === 'granted')"
            class="btn-ghost text-xs"
            :disabled="notifPermission === 'denied'"
            :title="
              notifPermission === 'denied' ? 'Notifications are blocked in your browser' : undefined
            "
            @click="enableNotifications"
          >
            <span class="i-lucide-bell text-sm" aria-hidden="true" />
            Notify me when done
          </button>
          <span
            v-else-if="notifSupported"
            class="text-muted inline-flex items-center gap-1 text-xs"
          >
            <span class="i-lucide-bell-ring text-ok text-sm" aria-hidden="true" />
            Will notify when done
          </span>
        </div>
      </div>

      <div v-else-if="review.status === 'cancelled'" class="flex flex-col items-start gap-3">
        <p class="text-muted text-sm">This review was cancelled.</p>
        <button class="btn-line" @click="retry">
          <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
          Retry (clones the review)
        </button>
      </div>

      <div v-else-if="review.status === 'error'" class="flex flex-col items-start gap-3">
        <p class="text-danger text-sm">{{ review.error || 'Review failed.' }}</p>
        <button class="btn-line" @click="retry">
          <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
          Retry (clones the review)
        </button>
      </div>

      <template v-else>
        <!-- Global humanize bar: pick a ready profile, then Humanize all fires
             the summary + every finding concurrently with per-card spinners. -->
        <div
          class="border-line/50 mb-6 flex flex-col gap-3 border-b pb-4 sm:flex-row sm:flex-wrap sm:items-end sm:gap-4"
        >
          <!-- Phone-only toggle: collapses the humanize settings behind a button.
               Gains a warn affordance (triangle icon + warn text) when humanize
               is not usable so the unavailable state reads at a glance. -->
          <button
            type="button"
            class="btn-line w-full justify-between text-xs sm:hidden"
            :class="humanizeUnavailable ? 'text-warn' : ''"
            :aria-expanded="humanizeOpen"
            @click="humanizeOpen = !humanizeOpen"
          >
            <span class="flex items-center gap-2">
              <span
                :class="humanizeUnavailable ? 'i-lucide-triangle-alert' : 'i-lucide-feather'"
                class="text-sm"
                aria-hidden="true"
              />
              Humanize
            </span>
            <span
              class="i-lucide-chevron-down text-sm transition-transform"
              :class="humanizeOpen ? 'rotate-180' : ''"
              aria-hidden="true"
            />
          </button>
          <!-- Phone-only warning: makes the unusable humanize state obvious even
               while the settings stay collapsed. Desktop keeps its inline copy. -->
          <Alert v-if="humanizeUnavailable" variant="warn" class="sm:hidden">
            <template v-if="!readyProfiles.length">
              No ready humanization profiles.
              <RouterLink to="/profiles" class="text-accent hover:underline">
                Manage profiles
              </RouterLink>
            </template>
            <template v-else>Select a humanization profile to humanize.</template>
          </Alert>
          <template v-if="humanizeOpen || !isPhone">
            <template v-if="readyProfiles.length">
              <label class="block w-full sm:w-auto">
                <span class="field-label">Humanize profile</span>
                <select v-model="profileId" class="field-underline sm:min-w-48">
                  <option v-for="p in readyProfiles" :key="p.id" :value="p.id">{{ p.name }}</option>
                </select>
              </label>
              <button
                class="btn-line w-full text-xs sm:w-auto"
                :disabled="!profileId || humanizingAll"
                @click="humanizeAll"
              >
                <span
                  v-if="humanizingAll"
                  class="i-lucide-loader-circle animate-spin text-sm"
                  aria-hidden="true"
                />
                <span v-else class="i-lucide-feather text-sm" aria-hidden="true" />
                Humanize all
              </button>
            </template>
            <p v-else class="text-muted text-sm">
              No ready humanization profiles.
              <RouterLink to="/profiles" class="text-accent hover:underline">
                Manage profiles
              </RouterLink>
            </p>
          </template>
        </div>

        <SummaryCard :review="review" :profile-id="profileId" />

        <section class="mt-6" :class="selected.length ? 'pb-24 sm:pb-0' : ''">
          <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h2 class="section-title flex items-center gap-2 hidden md:inline ">
              <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
              <span class="hidden sm:inline">Findings</span>
            </h2>
            <div v-if="review.findings.length" class="flex flex-wrap items-center gap-3">
              <div
                class="flex w-full items-center gap-1 sm:w-auto"
                role="group"
                aria-label="Findings view"
              >
                <button
                  v-for="v in (['triage', 'classic'] as const)"
                  :key="v"
                  type="button"
                  class="flex-1 border px-2 py-1.5 text-xs capitalize transition-colors sm:flex-none sm:py-0.5"
                  :class="
                    findingsView === v
                      ? 'border-line text-ink bg-accent/10 sm:border-accent'
                      : 'border-line text-muted hover:text-ink'
                  "
                  :aria-pressed="findingsView === v"
                  @click="findingsView = v"
                >
                  {{ v }}
                </button>
              </div>
              <!-- Include-summary checkbox: desktop only. On phone the choice moves
                   into the publish-confirm modal (askPublish). -->
              <label
                class="text-muted hidden cursor-pointer items-center gap-1.5 text-xs sm:flex"
              >
                <input v-model="includeSummary" type="checkbox" class="accent-accent" />
                Include summary note
              </label>
              <div class="flex w-full items-center gap-2 sm:w-auto">
                <!-- Copy + Publish selected: desktop only — the phone sticky bar covers them. -->
                <button
                  class="btn-ghost hidden flex-1 text-xs sm:inline-flex sm:flex-none"
                  :disabled="selected.length === 0"
                  @click="copySelected"
                >
                  <span class="i-lucide-copy text-sm" aria-hidden="true" />
                  Copy selected ({{ selected.length }})
                </button>
                <button
                  class="btn-ghost hidden flex-1 text-xs sm:inline-flex sm:flex-none"
                  :disabled="publishing || selected.length === 0"
                  @click="publish({ indices: selected, includeSummary })"
                >
                  Publish selected ({{ selected.length }})
                </button>
                <button
                  class="btn-accent flex-1 text-xs sm:flex-none"
                  :disabled="publishing || unpublished.length === 0"
                  @click="onCommentAll"
                >
                  <span
                    v-if="publishing"
                    class="i-lucide-loader-circle animate-spin"
                    aria-hidden="true"
                  />
                  Comment all
                </button>
              </div>
            </div>
          </div>

          <template v-if="review.summaryPublished">
            <!-- Phone: compact alert box. Desktop: unchanged plain text. -->
            <Alert variant="info" class="mb-3 sm:hidden">
              Summary already posted — re-check to post again.
            </Alert>
            <p class="text-muted/70 mb-3 hidden text-xs sm:block">
              Summary already posted — re-check to post again.
            </p>
          </template>
          <template v-if="publishError">
            <Alert variant="danger" class="mb-3 sm:hidden">{{ publishError }}</Alert>
            <p class="text-danger mb-3 hidden text-sm sm:block">{{ publishError }}</p>
          </template>

          <p v-if="review.findings.length === 0" class="text-muted text-sm">
            No findings — clean review.
          </p>

          <!-- Classic: the original flat list, unchanged. -->
          <div v-else-if="findingsView === 'classic'" class="border-line/50 border-t">
            <FindingCard
              v-for="f in review.findings"
              :key="f.index"
              :finding="f"
              :review-id="review.id"
              :profile-id="profileId"
              :selectable="!f.published"
              :selected="selected.includes(f.index)"
              @toggle="toggle"
            />
          </div>

          <!-- Triage: filter/sort toolbar + the visible slice of the same cards. -->
          <div v-else>
            <!-- Phone-only controls: Filters toggle (with active-count badge and
                 the visible/total caption folded in) plus the selection actions.
                 The verbose "Showing X of N" row below is desktop-only. -->
            <div class="mb-3 flex flex-wrap items-center gap-2 text-xs sm:hidden">
              <button
                type="button"
                class="btn-line text-xs"
                :aria-expanded="filtersOpen"
                aria-controls="triage-filters"
                @click="filtersOpen = !filtersOpen"
              >
                <span class="i-lucide-sliders-horizontal text-sm" aria-hidden="true" />
                Filters
                <span
                  v-if="activeFilterCount"
                  class="border-accent/40 bg-accent/15 text-accent border px-1 font-mono"
                >
                  {{ activeFilterCount }}
                </span>
                <span class="text-muted/70">
                  {{ triage.visible.value.length }}/{{ review.findings.length }}
                </span>
              </button>
              <button
                type="button"
                class="btn-ghost text-xs"
                :disabled="selectableVisible.length === 0"
                @click="selectAllVisible"
              >
                Select all ({{ selectableVisible.length }})
              </button>
              <button
                v-if="selected.length"
                type="button"
                class="btn-ghost text-xs"
                @click="clearSelection"
              >
                Clear
              </button>
            </div>

            <!-- Desktop: inline toolbar, rendered exactly as before. -->
            <FindingsToolbar
              v-if="!isPhone"
              id="triage-filters"
              :counts="triage.counts.value"
              :filters="triage.filters"
              :sort="triage.sort.value"
              :dimensions="triage.dimensions"
              :severities="triage.severities"
              :active="triage.active.value"
              @toggle-dimension="triage.toggleDimension"
              @toggle-severity="triage.toggleSeverity"
              @toggle-blocking-only="triage.toggleBlockingOnly"
              @set-status="triage.setStatus"
              @set-sort="triage.setSort"
              @reset="triage.reset"
            />

            <!-- Phone: the same toolbar lives in a bottom-sheet modal, opened by
                 the Filters button. Bound to the identical triage state/handlers. -->
            <Modal v-if="isPhone" :open="filtersOpen" title="Filters" @close="filtersOpen = false">
              <FindingsToolbar
                id="triage-filters"
                :counts="triage.counts.value"
                :filters="triage.filters"
                :sort="triage.sort.value"
                :dimensions="triage.dimensions"
                :severities="triage.severities"
                :active="triage.active.value"
                @toggle-dimension="triage.toggleDimension"
                @toggle-severity="triage.toggleSeverity"
                @toggle-blocking-only="triage.toggleBlockingOnly"
                @set-status="triage.setStatus"
                @set-sort="triage.setSort"
                @reset="triage.reset"
              />
            </Modal>

            <div class="text-muted mb-3 hidden flex-wrap items-center gap-3 text-xs sm:flex">
              <span>
                Showing {{ triage.visible.value.length }} of {{ review.findings.length }}
              </span>
              <button
                type="button"
                class="btn-ghost text-xs"
                :disabled="selectableVisible.length === 0"
                @click="selectAllVisible"
              >
                Select all visible ({{ selectableVisible.length }})
              </button>
              <button
                v-if="selected.length"
                type="button"
                class="btn-ghost text-xs"
                @click="clearSelection"
              >
                Clear selection
              </button>
            </div>

            <template v-if="triage.visible.value.length === 0">
              <!-- Phone: compact alert box. Desktop: unchanged plain text. -->
              <Alert variant="info" class="sm:hidden">No findings match the active filters.</Alert>
              <p class="text-muted hidden text-sm sm:block">
                No findings match the active filters.
              </p>
            </template>
            <div v-else class="flex flex-col gap-3">
              <FindingCardTriage
                v-for="f in triage.visible.value"
                :key="f.index"
                :finding="f"
                :review-id="review.id"
                :profile-id="profileId"
                :selectable="!f.published"
                :selected="selected.includes(f.index)"
                @toggle="toggle"
              />
            </div>
          </div>
        </section>

        <!-- Phone-only sticky selection bar: elevated bottom action strip shown
             only while findings are selected. Desktop keeps the inline controls. -->
        <div
          v-if="selected.length"
          class="border-line bg-surface fixed inset-x-0 bottom-0 z-40 flex items-center gap-2 border-t p-3 sm:hidden"
          role="region"
          aria-label="Selection actions"
        >
          <button
            class="btn-accent flex-1 text-xs"
            :disabled="publishing"
            @click="askPublish({ indices: selected })"
          >
            <span
              v-if="publishing"
              class="i-lucide-loader-circle animate-spin text-sm"
              aria-hidden="true"
            />
            Publish selected ({{ selected.length }})
          </button>
          <button class="btn-line text-xs" aria-label="Copy selected as Markdown" @click="copySelected">
            <span class="i-lucide-copy text-sm" aria-hidden="true" />
          </button>
          <button class="btn-line text-xs" aria-label="Clear selection" @click="clearSelection">
            Clear
          </button>
        </div>

        <!-- Phone-only publish confirmation: surfaces the include-summary choice
             (hidden from the phone controls) and defers the actual publish until
             confirmed. Desktop publishes inline without this modal. -->
        <Modal
          v-if="isPhone"
          :open="publishConfirmOpen"
          title="Publish"
          @close="publishConfirmOpen = false"
        >
          <div class="flex flex-col gap-4">
            <p class="text-muted text-sm">
              {{
                pendingPublish?.all
                  ? 'Comment all findings'
                  : `Publish ${pendingPublish?.indices?.length ?? 0} selected findings`
              }}
            </p>
            <label class="text-ink flex cursor-pointer items-center gap-2 text-sm">
              <input v-model="confirmIncludeSummary" type="checkbox" class="accent-accent" />
              Include summary note
            </label>
            <div class="flex gap-2">
              <button
                class="btn-accent w-full text-xs"
                :disabled="publishing"
                @click="confirmPublish"
              >
                <span
                  v-if="publishing"
                  class="i-lucide-loader-circle animate-spin text-sm"
                  aria-hidden="true"
                />
                Publish
              </button>
              <button class="btn-line w-full text-xs" @click="publishConfirmOpen = false">
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      </template>
    </template>
  </div>
</template>
