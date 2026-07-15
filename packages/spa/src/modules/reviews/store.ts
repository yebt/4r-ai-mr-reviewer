import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type {
  FindingHumanized,
  HumanizeFindingText,
  MergeRequest,
  Review,
  SummaryHumanized,
} from '@shared/api/types'
import { ORIGINAL } from '@modules/reviews/humanize-overrides'

// Per-review humanize state. Accumulating tabs of structured humanized output
// plus the selected tab per card, kept in the store (keyed by reviewId). Every
// run is also persisted server-side, so `load` rehydrates these tabs from the
// server (the source of truth) and they survive a full page reload.
interface HumanizedState {
  summary: SummaryHumanized[] // tabs 0..n; Original is implicit (ORIGINAL)
  findings: Record<number, FindingHumanized[]> // per finding index → tabs
}
interface SelectedState {
  summary: number // ORIGINAL (-1) or a tab index
  findings: Record<number, number> // per finding index → ORIGINAL or tab index
}
interface HumanizingState {
  summary: boolean
  findings: Record<number, boolean>
}
// Which tab index was actually published, per card. This is an IN-SESSION marker
// only (it does not survive a page reload) that complements the backend
// `published`/`summaryPublished` flags: the flags drive the persistent "already
// on the MR" card highlight, while this tells the user which specific version
// (Original / V1 / V2…) they sent. `summary === null` and an absent findings key
// both mean "nothing published yet this session".
interface PublishedTabState {
  summary: number | null // ORIGINAL (-1) or a tab index; null = not published
  findings: Record<number, number> // per finding index → published tab index
}

export const useReviewsStore = defineStore('reviews', () => {
  // Caches keyed by repo id, so revisiting a repo shows its previous MRs/reviews
  // immediately while we revalidate in the background (no loading flash).
  const mrsByRepo = ref<Record<string, MergeRequest[]>>({})
  const mrsLoading = ref(false)
  const mrsError = ref<string | null>(null)

  const reviewsByRepo = ref<Record<string, Review[]>>({})
  const listLoading = ref(false)
  const listError = ref<string | null>(null)

  // Archived reviews live in a separate cache slot so a "Show archived" toggle
  // can render them without clobbering the active list.
  const archivedByRepo = ref<Record<string, Review[]>>({})
  const archivedLoading = ref(false)
  const archivedError = ref<string | null>(null)

  // Review-detail cache keyed by review id.
  const reviewsById = ref<Record<string, Review>>({})
  const current = ref<Review | null>(null)
  const currentLoading = ref(false)
  const currentError = ref<string | null>(null)

  // In-session humanize caches, keyed by reviewId. See the interfaces above.
  const humanized = ref<Record<string, HumanizedState>>({})
  const selected = ref<Record<string, SelectedState>>({})
  const humanizing = ref<Record<string, HumanizingState>>({})
  // In-session marker of the published tab per card. See PublishedTabState.
  const publishedTab = ref<Record<string, PublishedTabState>>({})

  // Lazy initializers — nested records are created on first touch so callers
  // never read undefined (noUncheckedIndexedAccess is on).
  function ensureHumanized(reviewId: string): HumanizedState {
    const existing = humanized.value[reviewId]
    if (existing) return existing
    const created: HumanizedState = { summary: [], findings: {} }
    humanized.value[reviewId] = created
    return created
  }
  function ensureSelected(reviewId: string): SelectedState {
    const existing = selected.value[reviewId]
    if (existing) return existing
    const created: SelectedState = { summary: ORIGINAL, findings: {} }
    selected.value[reviewId] = created
    return created
  }
  function ensureHumanizing(reviewId: string): HumanizingState {
    const existing = humanizing.value[reviewId]
    if (existing) return existing
    const created: HumanizingState = { summary: false, findings: {} }
    humanizing.value[reviewId] = created
    return created
  }
  function ensurePublishedTab(reviewId: string): PublishedTabState {
    const existing = publishedTab.value[reviewId]
    if (existing) return existing
    const created: PublishedTabState = { summary: null, findings: {} }
    publishedTab.value[reviewId] = created
    return created
  }

  // Read-only helpers with safe defaults for templates.
  function summaryTabs(reviewId: string): SummaryHumanized[] {
    return humanized.value[reviewId]?.summary ?? []
  }
  function findingTabs(reviewId: string, index: number): FindingHumanized[] {
    return humanized.value[reviewId]?.findings[index] ?? []
  }
  function selectedSummaryTab(reviewId: string): number {
    return selected.value[reviewId]?.summary ?? ORIGINAL
  }
  function selectedFindingTab(reviewId: string, index: number): number {
    return selected.value[reviewId]?.findings[index] ?? ORIGINAL
  }
  function summaryHumanizing(reviewId: string): boolean {
    return humanizing.value[reviewId]?.summary ?? false
  }
  function findingHumanizing(reviewId: string, index: number): boolean {
    return humanizing.value[reviewId]?.findings[index] ?? false
  }

  // Read the in-session published-tab marker; null means nothing published yet.
  function publishedSummaryTab(reviewId: string): number | null {
    return publishedTab.value[reviewId]?.summary ?? null
  }
  function publishedFindingTab(reviewId: string, index: number): number | null {
    return publishedTab.value[reviewId]?.findings[index] ?? null
  }

  // markPublished records which tab (Original = ORIGINAL, else a tab index) was
  // just published for the summary (target 'summary') or a finding (target =
  // finding index). Call only after a successful publish.
  function markPublished(reviewId: string, target: 'summary' | number, tab: number) {
    const entry = ensurePublishedTab(reviewId)
    if (target === 'summary') entry.summary = tab
    else entry.findings[target] = tab
  }

  function selectSummaryTab(reviewId: string, tab: number) {
    ensureSelected(reviewId).summary = tab
  }
  function selectFindingTab(reviewId: string, index: number, tab: number) {
    ensureSelected(reviewId).findings[index] = tab
  }

  // humanizeSummary appends a structured summary rewrite as a new tab and
  // auto-selects it. Concurrency-safe and independent of every other card: it
  // only touches its own reviewId/summary slot. Rethrows so callers can toast.
  async function humanizeSummary(reviewId: string, profileId: string) {
    ensureHumanizing(reviewId).summary = true
    try {
      const result = await api.humanizeSummary(reviewId, profileId)
      const entry = ensureHumanized(reviewId)
      entry.summary.push(result)
      ensureSelected(reviewId).summary = entry.summary.length - 1
    } finally {
      ensureHumanizing(reviewId).summary = false
    }
  }

  // humanizeFinding appends a structured rewrite for one finding as a new tab
  // and auto-selects it. Independent per finding index, so many can be in flight
  // at once. Rethrows so callers can toast.
  async function humanizeFinding(reviewId: string, profileId: string, index: number) {
    ensureHumanizing(reviewId).findings[index] = true
    try {
      const result = await api.humanizeFinding(reviewId, profileId, index)
      const entry = ensureHumanized(reviewId)
      const tabs = entry.findings[index] ?? []
      tabs.push(result)
      entry.findings[index] = tabs
      ensureSelected(reviewId).findings[index] = tabs.length - 1
    } finally {
      ensureHumanizing(reviewId).findings[index] = false
    }
  }

  function mergeRequestsFor(repoId: string): MergeRequest[] {
    return mrsByRepo.value[repoId] ?? []
  }

  function reviewsFor(repoId: string): Review[] {
    return reviewsByRepo.value[repoId] ?? []
  }

  function archivedReviewsFor(repoId: string): Review[] {
    return archivedByRepo.value[repoId] ?? []
  }

  // Flattened view of every cached review, newest first — for the global list.
  const allReviews = computed(() =>
    Object.values(reviewsByRepo.value)
      .flat()
      .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1)),
  )

  async function fetchMergeRequests(repoId: string) {
    mrsLoading.value = true
    mrsError.value = null
    try {
      const data = await api.listMergeRequests(repoId)
      mrsByRepo.value = { ...mrsByRepo.value, [repoId]: data }
    } catch (e) {
      mrsError.value = errorMessage(e)
    } finally {
      mrsLoading.value = false
    }
  }

  async function fetchReviews(repoId: string) {
    listLoading.value = true
    listError.value = null
    try {
      const data = await api.listRepoReviews(repoId)
      reviewsByRepo.value = { ...reviewsByRepo.value, [repoId]: data }
    } catch (e) {
      listError.value = errorMessage(e)
    } finally {
      listLoading.value = false
    }
  }

  async function fetchArchivedReviews(repoId: string) {
    archivedLoading.value = true
    archivedError.value = null
    try {
      const data = await api.listRepoReviews(repoId, true)
      archivedByRepo.value = { ...archivedByRepo.value, [repoId]: data }
    } catch (e) {
      archivedError.value = errorMessage(e)
    } finally {
      archivedLoading.value = false
    }
  }

  function cacheReview(rv: Review) {
    reviewsById.value[rv.id] = rv
    const list = reviewsByRepo.value[rv.repoId]
    if (list) {
      const exists = list.some((r) => r.id === rv.id)
      reviewsByRepo.value = {
        ...reviewsByRepo.value,
        [rv.repoId]: exists ? list.map((r) => (r.id === rv.id ? rv : r)) : [rv, ...list],
      }
    }
  }

  async function create(repoId: string, mrIid: number, mode: string) {
    const created = await api.createReview({ repoId, mrIid, mode })
    cacheReview(created)
    return created
  }

  // hydrateHumanized overwrites the in-session humanize cache for a review with
  // the server's persisted tabs, so a full page reload rehydrates every tab. The
  // server is the source of truth. Selection stays defaulted to Original — we do
  // not auto-select a rehydrated tab.
  async function hydrateHumanized(reviewId: string) {
    // A humanize call in flight owns the in-session tabs — its result is not
    // persisted yet, so overwriting from the server here would drop it and let
    // the pending push land on a stale array. Skip until it settles.
    const busy = humanizing.value[reviewId]
    if (busy && (busy.summary || Object.values(busy.findings).some(Boolean))) return

    const data = await api.getHumanizations(reviewId)
    humanized.value[reviewId] = {
      summary: data.summary ?? [],
      findings: data.findings ?? {},
    }
  }

  async function load(id: string) {
    const cached = reviewsById.value[id]
    if (cached) current.value = cached
    // Only show the spinner when we have nothing to show yet.
    currentLoading.value = !cached
    currentError.value = null
    try {
      const rv = await api.getReview(id)
      current.value = rv
      cacheReview(rv)
    } catch (e) {
      currentError.value = errorMessage(e)
    } finally {
      currentLoading.value = false
    }
    // Rehydrate persisted humanize tabs from the server. Best-effort: a failure
    // here must not blank the review that already loaded above.
    try {
      await hydrateHumanized(id)
    } catch {
      // Keep whatever in-session tabs we already have.
    }
  }

  // refresh silently updates the current review (used while polling).
  async function refresh(id: string) {
    const rv = await api.getReview(id)
    current.value = rv
    cacheReview(rv)
  }

  async function retry(id: string) {
    return api.retryReview(id)
  }

  // cancel requests cooperative cancellation of a pending/running review, then
  // refreshes so the UI observes the flip to the cancelled terminal state.
  async function cancel(id: string) {
    await api.cancelReview(id)
    await refresh(id)
  }

  // remove hard-deletes a review and purges it from every cache.
  async function remove(id: string) {
    await api.deleteReview(id)
    const { [id]: _removed, ...rest } = reviewsById.value
    reviewsById.value = rest
    reviewsByRepo.value = Object.fromEntries(
      Object.entries(reviewsByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    if (current.value?.id === id) current.value = null
  }

  // archive soft-hides a review: it drops out of the active repo list and its
  // archived flag flips across the detail caches.
  async function archive(id: string) {
    await api.archiveReview(id)
    setArchivedFlag(id, true)
    // Remove from every active repo list; drop the whole archived cache slot so
    // it is refetched fresh when the toggle is next opened.
    reviewsByRepo.value = Object.fromEntries(
      Object.entries(reviewsByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    const repoId = reviewsById.value[id]?.repoId ?? current.value?.repoId
    if (repoId) {
      const { [repoId]: _dropped, ...rest } = archivedByRepo.value
      archivedByRepo.value = rest
    }
  }

  // unarchive restores a review to the active list.
  async function unarchive(id: string) {
    await api.unarchiveReview(id)
    setArchivedFlag(id, false)
    archivedByRepo.value = Object.fromEntries(
      Object.entries(archivedByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    const repoId = reviewsById.value[id]?.repoId ?? current.value?.repoId
    if (repoId) {
      const { [repoId]: _dropped, ...rest } = reviewsByRepo.value
      reviewsByRepo.value = rest
    }
  }

  // setArchivedFlag keeps the detail caches (by id + current) in sync.
  function setArchivedFlag(id: string, archived: boolean) {
    const cached = reviewsById.value[id]
    if (cached) reviewsById.value[id] = { ...cached, archived }
    if (current.value?.id === id) current.value = { ...current.value, archived }
  }

  async function publish(
    id: string,
    selection: {
      all?: boolean
      indices?: number[]
      includeSummary?: boolean
      // Optional per-publish overrides (humanized summary / finding bodies).
      summaryOverride?: string
      findingOverrides?: HumanizeFindingText[]
    },
  ) {
    await api.publishReview(id, selection)
    await refresh(id)
  }

  return {
    mrsByRepo,
    mrsLoading,
    mrsError,
    reviewsByRepo,
    listLoading,
    listError,
    archivedByRepo,
    archivedLoading,
    archivedError,
    reviewsById,
    current,
    currentLoading,
    currentError,
    mergeRequestsFor,
    reviewsFor,
    archivedReviewsFor,
    allReviews,
    fetchMergeRequests,
    fetchReviews,
    fetchArchivedReviews,
    create,
    load,
    refresh,
    retry,
    cancel,
    remove,
    archive,
    unarchive,
    publish,
    summaryTabs,
    findingTabs,
    selectedSummaryTab,
    selectedFindingTab,
    summaryHumanizing,
    findingHumanizing,
    selectSummaryTab,
    selectFindingTab,
    publishedSummaryTab,
    publishedFindingTab,
    markPublished,
    humanizeSummary,
    humanizeFinding,
  }
})
