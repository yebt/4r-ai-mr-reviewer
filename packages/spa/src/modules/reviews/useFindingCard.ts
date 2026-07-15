import { computed, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Finding, FindingHumanized } from '@shared/api/types'
import { useReviewsStore } from '@modules/reviews/store'
import { ORIGINAL, buildFindingBody } from '@modules/reviews/humanize-overrides'

// Shared logic for the classic (FindingCard) and triage (FindingCardTriage)
// finding cards: humanize tabs, the parts shown for the active tab, and the
// per-card humanize/publish actions. Each card keeps only its own template and
// severity-border styling. Pass the component's reactive props by reference so
// the computeds stay reactive; `store` is returned for template tab-switching.
export function useFindingCard(props: { finding: Finding; reviewId: string; profileId: string }) {
  const store = useReviewsStore()

  const tabs = computed(() => store.findingTabs(props.reviewId, props.finding.index))
  const selectedTab = computed(() => store.selectedFindingTab(props.reviewId, props.finding.index))
  const humanizing = computed(() => store.findingHumanizing(props.reviewId, props.finding.index))
  // In-session marker: which tab was published for this finding (null = none).
  const publishedTabIdx = computed(() =>
    store.publishedFindingTab(props.reviewId, props.finding.index),
  )

  // The finding parts shown for the active tab: Original is the generated finding;
  // a humanize tab is that run's rewritten issue/why/fix.
  const shown = computed<Pick<FindingHumanized, 'issue' | 'why' | 'fix'>>(() => {
    if (selectedTab.value === ORIGINAL) {
      return { issue: props.finding.issue, why: props.finding.why, fix: props.finding.fix }
    }
    const tab = tabs.value[selectedTab.value]
    return { issue: tab?.issue ?? '', why: tab?.why ?? '', fix: tab?.fix ?? '' }
  })

  const publishing = ref(false)

  async function humanize() {
    if (!props.profileId || humanizing.value) return
    try {
      await store.humanizeFinding(props.reviewId, props.profileId, props.finding.index)
    } catch (e) {
      toast.error(errorMessage(e))
    }
  }

  async function publish() {
    if (publishing.value) return
    publishing.value = true
    try {
      const tab = selectedTab.value
      const humanized = tab === ORIGINAL ? null : tabs.value[tab]
      // Original posts the generated body (no override forwarded). A humanize tab
      // forwards the assembled body, with the restored dimension/severity tag.
      await store.publish(props.reviewId, {
        indices: [props.finding.index],
        includeSummary: false,
        ...(humanized
          ? {
              findingOverrides: [
                { index: props.finding.index, text: buildFindingBody(props.finding, humanized) },
              ],
            }
          : {}),
      })
      store.markPublished(props.reviewId, props.finding.index, tab)
      toast.success('Finding published')
    } catch (e) {
      toast.error(errorMessage(e))
    } finally {
      publishing.value = false
    }
  }

  return { store, tabs, selectedTab, humanizing, publishedTabIdx, shown, publishing, humanize, publish }
}
