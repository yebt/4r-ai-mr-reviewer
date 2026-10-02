/**
 * Humanize state (slice 2) for `src/pages/reviews/[id].vue` — profile
 * selection, past + newly-created humanize runs, active-tab state per
 * summary/finding, and the override builders that feed the slice-1 publish
 * handlers. Page-local like `detail.ts`'s `useReviewDetail`, which this
 * composable is meant to sit alongside (it takes the review that composable
 * already fetched/polls, rather than re-fetching it).
 *
 * Past runs come from `GET /reviews/{id}/humanizations` (`['humanizations',
 * id]`, loaded once). New runs from `humanizeSummary`/`humanizeFinding` are
 * appended to local `extra*` arrays on success rather than triggering a
 * refetch, so a freshly-created tab appears immediately without a round
 * trip. Each accumulator's index space is shared: `run(n)` for `n <
 * fetched.length` reads the fetched list, otherwise the local `extra` list —
 * so tab index stays one contiguous, run-ordered sequence for both.
 */
import { computed, reactive, ref, watch, type Ref } from 'vue'
import { useQuery } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { useProfilesStore } from '@modules/profiles/store'
import { useReposStore } from '@modules/repos/store'
import { buildFindingBody, ORIGINAL } from './humanize'
import * as reviewsApi from './api'
import type { Finding, FindingHumanized, Review, SummaryHumanized } from './types'

export const HUMANIZATIONS_QUERY_KEY = 'humanizations' as const

export function useReviewHumanize(reviewId: Ref<string>, review: Ref<Review | undefined>) {
  const profilesStore = useProfilesStore()
  const reposStore = useReposStore()
  const toast = useToast()

  const readyProfiles = computed(() => profilesStore.profiles.filter((profile) => profile.styleGuideStatus === 'ready'))
  const hasReadyProfile = computed(() => readyProfiles.value.length > 0)

  // Pre-seed from the review's repo default (when it's a ready profile),
  // else the first ready profile — then sticky: once seeded, the user's own
  // pick is never overwritten by this watcher.
  const profileId = ref('')
  let profileSeeded = false
  watch(
    [readyProfiles, review],
    ([profiles, currentReview]) => {
      if (profileSeeded || profiles.length === 0 || !currentReview) return
      const repo = reposStore.repos.find((r) => r.id === currentReview.repoId)
      const defaultReady = repo?.defaultProfileId ? profiles.find((p) => p.id === repo.defaultProfileId) : undefined
      profileId.value = defaultReady?.id ?? profiles[0]!.id
      profileSeeded = true
    },
    { immediate: true },
  )

  const query = useQuery({
    key: () => [HUMANIZATIONS_QUERY_KEY, reviewId.value],
    query: () => reviewsApi.getHumanizations(reviewId.value),
  })

  // Locally-accumulated runs from this session's mutations, appended (never
  // replacing the fetched list) so new tabs show up without a refetch.
  const extraSummaryRuns = ref<SummaryHumanized[]>([])
  const extraFindingRuns = reactive<Record<number, FindingHumanized[]>>({})

  watch(reviewId, () => {
    extraSummaryRuns.value = []
    for (const key of Object.keys(extraFindingRuns)) delete extraFindingRuns[Number(key)]
  })

  const summaryTabs = computed(() => (query.data.value?.summary.length ?? 0) + extraSummaryRuns.value.length)
  function findingTabs(index: number): number {
    const fetched = query.data.value?.findings[String(index)]?.length ?? 0
    return fetched + (extraFindingRuns[index]?.length ?? 0)
  }

  function summaryRunAt(tab: number): SummaryHumanized | undefined {
    const fetched = query.data.value?.summary ?? []
    if (tab < fetched.length) return fetched[tab]
    return extraSummaryRuns.value[tab - fetched.length]
  }
  function findingRunAt(index: number, tab: number): FindingHumanized | undefined {
    const fetched = query.data.value?.findings[String(index)] ?? []
    if (tab < fetched.length) return fetched[tab]
    return extraFindingRuns[index]?.[tab - fetched.length]
  }

  // Active-tab state: ORIGINAL (generated original) or a run index.
  const summaryTab = ref<number>(ORIGINAL)
  const findingTabState = reactive<Record<number, number>>({})
  function setSummaryTab(tab: number) {
    summaryTab.value = tab
  }
  function findingTab(index: number): number {
    return findingTabState[index] ?? ORIGINAL
  }
  function setFindingTab(index: number, tab: number) {
    findingTabState[index] = tab
  }

  function activeSummaryText(currentReview: Review): string {
    if (summaryTab.value === ORIGINAL) return currentReview.summary
    return summaryRunAt(summaryTab.value)?.summary ?? currentReview.summary
  }
  function activeFindingParts(finding: Finding): { issue: string; why: string; fix: string } {
    const tab = findingTab(finding.index)
    if (tab === ORIGINAL) return { issue: finding.issue, why: finding.why, fix: finding.fix }
    return findingRunAt(finding.index, tab) ?? { issue: finding.issue, why: finding.why, fix: finding.fix }
  }

  const pendingSummary = ref(false)
  const pendingFindingIndices = ref<Set<number>>(new Set())
  const pendingAll = ref(false)

  function isHumanizingFinding(index: number): boolean {
    return pendingFindingIndices.value.has(index)
  }

  async function humanizeSummary() {
    if (!profileId.value) return
    pendingSummary.value = true
    try {
      const result = await reviewsApi.humanizeSummary(reviewId.value, profileId.value)
      extraSummaryRuns.value = [...extraSummaryRuns.value, result]
      summaryTab.value = summaryTabs.value - 1
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to humanize summary'))
    } finally {
      pendingSummary.value = false
    }
  }

  async function humanizeFinding(index: number) {
    if (!profileId.value) return
    pendingFindingIndices.value = new Set(pendingFindingIndices.value).add(index)
    try {
      const result = await reviewsApi.humanizeFinding(reviewId.value, profileId.value, index)
      extraFindingRuns[index] = [...(extraFindingRuns[index] ?? []), result]
      findingTabState[index] = findingTabs(index) - 1
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to humanize finding'))
    } finally {
      const next = new Set(pendingFindingIndices.value)
      next.delete(index)
      pendingFindingIndices.value = next
    }
  }

  async function humanizeAll() {
    if (!profileId.value || !review.value) return
    pendingAll.value = true
    try {
      await Promise.allSettled([humanizeSummary(), ...review.value.findings.map((finding) => humanizeFinding(finding.index))])
    } finally {
      pendingAll.value = false
    }
  }

  // Override builders — the publish integration point. `undefined` means
  // "no override" (the active tab is Original), so callers can pass these
  // straight through to `PublishSelection` unchanged.
  function summaryOverride(): string | undefined {
    if (summaryTab.value === ORIGINAL) return undefined
    return summaryRunAt(summaryTab.value)?.summary
  }
  function findingOverride(finding: Finding): { index: number; text: string } | undefined {
    const tab = findingTab(finding.index)
    if (tab === ORIGINAL) return undefined
    const run = findingRunAt(finding.index, tab)
    if (!run) return undefined
    return { index: finding.index, text: buildFindingBody(finding, run) }
  }
  function findingOverridesFor(findings: Finding[]): { index: number; text: string }[] {
    return findings
      .map((finding) => findingOverride(finding))
      .filter((override): override is { index: number; text: string } => override !== undefined)
  }

  return {
    readyProfiles,
    hasReadyProfile,
    profileId,
    isLoading: query.isLoading,

    summaryTabs,
    findingTabs,
    summaryTab,
    setSummaryTab,
    findingTab,
    setFindingTab,
    activeSummaryText,
    activeFindingParts,

    humanizeSummary,
    humanizeFinding,
    humanizeAll,
    isHumanizingSummary: pendingSummary,
    isHumanizingFinding,
    isHumanizingAll: pendingAll,

    summaryOverride,
    findingOverride,
    findingOverridesFor,
  }
}
