<script setup lang="ts">
import { computed, onUnmounted, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn, useLocalStorage } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import { useReviewsStore } from '@modules/reviews/store'
import { useReposStore } from '@modules/repos/store'
import { useProfilesStore } from '@modules/profiles/store'
import { isTerminal } from '@modules/reviews/format'
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

// Multi-pass progress: maps the backend phase to a labelled step (of 4).
const phaseMap: Record<string, { label: string; step: number }> = {
  risk: { label: 'Risk', step: 1 },
  readability: { label: 'Readability', step: 2 },
  reliability: { label: 'Reliability', step: 3 },
  resilience: { label: 'Resilience', step: 4 },
}
const phase = computed(() => (review.value?.phase ? (phaseMap[review.value.phase] ?? null) : null))

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

async function publish(payload: { all?: boolean; indices?: number[]; includeSummary?: boolean }) {
  publishing.value = true
  publishError.value = null
  try {
    await store.publish(reviewId.value, payload)
    selected.value = []
    toast.success('Findings published')
  } catch (e) {
    publishError.value = errorMessage(e)
  } finally {
    publishing.value = false
  }
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
        <button class="btn-line mt-4 text-xs" :disabled="cancelling" @click="cancel">
          <span
            v-if="cancelling"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-ban text-sm" aria-hidden="true" />
          Cancel review
        </button>
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
        <div class="border-line/50 mb-6 flex flex-wrap items-end gap-4 border-b pb-4">
          <template v-if="readyProfiles.length">
            <label class="block">
              <span class="field-label">Humanize profile</span>
              <select v-model="profileId" class="field-underline min-w-48">
                <option v-for="p in readyProfiles" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </label>
            <button
              class="btn-line text-xs"
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
            <RouterLink to="/profiles" class="text-accent hover:underline">Manage profiles</RouterLink>
          </p>
        </div>

        <SummaryCard :review="review" :profile-id="profileId" />

        <section class="mt-6">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <h2 class="section-title flex items-center gap-2">
              <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
              Findings
            </h2>
            <div v-if="review.findings.length" class="flex flex-wrap items-center gap-3">
              <div class="flex items-center gap-1" role="group" aria-label="Findings view">
                <button
                  v-for="v in (['triage', 'classic'] as const)"
                  :key="v"
                  type="button"
                  class="border px-2 py-0.5 text-xs capitalize transition-colors"
                  :class="
                    findingsView === v
                      ? 'border-accent text-ink bg-accent/10'
                      : 'border-line text-muted hover:text-ink'
                  "
                  :aria-pressed="findingsView === v"
                  @click="findingsView = v"
                >
                  {{ v }}
                </button>
              </div>
              <label class="text-muted flex cursor-pointer items-center gap-1.5 text-xs">
                <input v-model="includeSummary" type="checkbox" class="accent-accent" />
                Include summary note
              </label>
              <div class="flex items-center gap-2">
                <button
                  class="btn-ghost text-xs"
                  :disabled="publishing || selected.length === 0"
                  @click="publish({ indices: selected, includeSummary })"
                >
                  Publish selected ({{ selected.length }})
                </button>
                <button
                  class="btn-accent text-xs"
                  :disabled="publishing || unpublished.length === 0"
                  @click="publish({ all: true, includeSummary })"
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

          <p v-if="review.summaryPublished" class="text-muted/70 mb-3 text-xs">
            Summary already posted — re-check to post again.
          </p>
          <p v-if="publishError" class="text-danger mb-3 text-sm">{{ publishError }}</p>

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
            <FindingsToolbar
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

            <div class="text-muted mb-3 flex flex-wrap items-center gap-3 text-xs">
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

            <p v-if="triage.visible.value.length === 0" class="text-muted text-sm">
              No findings match the active filters.
            </p>
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
      </template>
    </template>
  </div>
</template>
