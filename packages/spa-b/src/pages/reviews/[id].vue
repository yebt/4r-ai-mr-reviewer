<script setup lang="ts">
/**
 * Review detail page (`/reviews/:id`, file-based route: `reviews/[id].vue`,
 * sibling of `reviews/index.vue` which is the list). Read-only: renders the
 * full `Review` from `GET /reviews/{id}` — summary, findings grouped by
 * dimension, and a collapsible reasoning trail, plus publish-to-MR and
 * humanize actions (see below).
 *
 * Data comes from `modules/reviews/detail.ts`'s `useReviewDetail`
 * composable, which polls every 2.5s while the review is non-terminal
 * (`isReviewActive`) and the tab is visible — see that file's doc comment.
 * `repoName` is resolved read-only from `useReposStore().repos` (deep
 * import into `modules/repos`, per this milestone's scope) so the header
 * reads "repo-name · !123" the same way the list row does, falling back to
 * the raw `repoId` if the repo isn't in the store's cache.
 *
 * The row actions (Retry/Archive/Approve/Discard) reuse `useReviewsStore`'s
 * mutations rather than duplicating them — a success there only invalidates
 * the *list* query key (`['reviews', ...]`), so this page also calls its own
 * `refetch()` afterwards to pick up the new status. Discard removes the
 * review, so it navigates back to the list instead.
 *
 * Publish (slice 1): once `status === 'done'`, each unpublished finding and
 * the summary get a Publish control (a `Badge` once already published),
 * plus a header "Publish all". Every publish is confirm-gated — it posts a
 * real, irreversible comment to the live GitLab MR. A per-action pending id
 * (`pendingPublishTarget`) tracks which control is loading; on success the
 * detail composable's `refetch()` re-fetches the review so `published` /
 * `summaryPublished` flip reactively (never hand-mutated locally).
 *
 * Humanize (slice 2): also gated on `status === 'done'`, `useReviewHumanize`
 * owns profile selection + past/new humanize runs + active-tab state; the
 * summary card and each finding card render its `activeSummaryText`/
 * `activeFindingParts` (Original or a humanized tab) instead of the raw
 * `Review`/`Finding` text, with a `HumanizeTabs` strip once runs exist. The
 * three publish handlers below merge in the composable's override builders
 * (`summaryOverride`/`findingOverride`/`findingOverridesFor`) so a non-
 * Original active tab replaces the generated MR comment body wholesale —
 * see `modules/reviews/humanize.ts`'s doc comment for the exact contract.
 */
import { computed, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Alert,
  Badge,
  Button,
  ConfirmDialog,
  Field,
  Select,
  Skeleton,
  Text,
} from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useReposStore } from '@modules/repos/store'
import { useReviewDetail } from '@modules/reviews/detail'
import { FINDING_DIMENSIONS, FINDING_SEVERITY_BADGE, groupFindingsByDimension } from '@modules/reviews/findings'
import { RECOMMENDATION_LABELS } from '@modules/reviews/labels'
import { hasUnpublished } from '@modules/reviews/publish'
import { useReviewsStore } from '@modules/reviews/store'
import { useReviewHumanize } from '@modules/reviews/useReviewHumanize'
import type { Finding } from '@modules/reviews/types'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import HumanizeTabs from '@modules/reviews/components/HumanizeTabs.vue'

const route = useRoute('/reviews/[id]')
const router = useRouter()

const reviewId = computed(() => route.params.id)

const { review, state, error, refetch, pausePolling } = useReviewDetail(reviewId)
onUnmounted(pausePolling)

const reposStore = useReposStore()
const repoName = computed(() => {
  const current = review.value
  if (!current) return ''
  return reposStore.repos.find((repo) => repo.id === current.repoId)?.name ?? current.repoId
})

const groupedFindings = computed(() => (review.value ? groupFindingsByDimension(review.value.findings) : null))

const {
  readyProfiles,
  hasReadyProfile,
  profileId,
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
  isHumanizingSummary,
  isHumanizingFinding,
  isHumanizingAll,
  summaryOverride,
  findingOverride,
  findingOverridesFor,
} = useReviewHumanize(reviewId, review)

const profileSelectItems = computed<SelectItemOption[]>(() =>
  readyProfiles.value.map((profile) => ({ label: profile.name, value: profile.id })),
)

const reviewsStore = useReviewsStore()
type PendingAction = 'retry' | 'archive' | 'unarchive' | 'approve' | 'discard'
const pendingAction = ref<PendingAction | null>(null)

async function runAction(action: PendingAction, run: () => Promise<unknown>) {
  pendingAction.value = action
  try {
    await run()
    await refetch()
  } catch {
    // no-op — the store already toasted the error
  } finally {
    pendingAction.value = null
  }
}

function handleRetry() {
  if (review.value) void runAction('retry', () => reviewsStore.retry(review.value!.id))
}

function handleToggleArchive() {
  if (!review.value) return
  const action: PendingAction = review.value.archived ? 'unarchive' : 'archive'
  const mutate = review.value.archived ? reviewsStore.unarchive : reviewsStore.archive
  void runAction(action, () => mutate(review.value!.id))
}

function handleApprove() {
  if (review.value) void runAction('approve', () => reviewsStore.approve(review.value!.id))
}

async function handleDiscard() {
  if (!review.value) return
  pendingAction.value = 'discard'
  try {
    await reviewsStore.discard(review.value.id)
    router.push('/reviews')
  } catch {
    // no-op — the store already toasted the error
  } finally {
    pendingAction.value = null
  }
}

// Publish — a separate pending tracker from `pendingAction` (row actions)
// so a finding/summary/all publish only lights up its own control.
// 'all' | 'summary' | `finding:<index>` identifies which control is loading.
const pendingPublishTarget = ref<string | null>(null)

async function runPublish(target: string, selection: Parameters<typeof reviewsStore.publish>[0]['selection']) {
  if (!review.value) return
  pendingPublishTarget.value = target
  try {
    await reviewsStore.publish({ id: review.value.id, selection })
    await refetch()
  } catch {
    // no-op — the store already toasted the error
  } finally {
    pendingPublishTarget.value = null
  }
}

function handlePublishFinding(finding: Finding) {
  const override = findingOverride(finding)
  void runPublish(`finding:${finding.index}`, {
    indices: [finding.index],
    includeSummary: false,
    findingOverrides: override ? [override] : undefined,
  })
}

function handlePublishSummary() {
  void runPublish('summary', { indices: [], includeSummary: true, summaryOverride: summaryOverride() })
}

function handlePublishAll() {
  if (!review.value) return
  void runPublish('all', {
    all: true,
    findingOverrides: findingOverridesFor(review.value.findings),
    summaryOverride: summaryOverride(),
  })
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <RouterLink to="/reviews" class="inline-flex w-fit items-center text-sm text-text-muted hover:text-text">
      ← Reviews
    </RouterLink>

    <div v-if="state.status === 'pending'" class="flex flex-col gap-3" data-testid="review-detail-skeleton">
      <Skeleton class="h-6 w-64" />
      <Skeleton class="h-4 w-40" />
      <Skeleton class="h-32 w-full" />
    </div>

    <Alert v-else-if="state.status === 'error'" status="danger">
      <p>{{ error?.message ?? 'Failed to load review' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">Retry</Button>
    </Alert>

    <div v-else-if="review" class="flex flex-col gap-6">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="flex min-w-0 flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text as="h1" size="xl" class="truncate font-semibold tracking-tight">
              {{ repoName }} · !{{ review.mrIid }}
            </Text>
            <ReviewStatusChip :status="review.status" />
            <Badge v-if="review.archived" status="neutral">Archived</Badge>
          </div>
          <Text muted size="sm">
            {{ review.contextMode }}<span v-if="review.model"> · {{ review.model }}</span>
          </Text>
          <div v-if="review.status === 'done'" class="mt-0.5 flex flex-wrap items-center gap-2">
            <Badge status="neutral">{{ RECOMMENDATION_LABELS[review.recommendation] }}</Badge>
            <Text muted size="sm">Score {{ review.score }}</Text>
            <Text muted size="sm">{{ review.inputTokens }} in / {{ review.outputTokens }} out tokens</Text>
          </div>
        </div>

        <div class="flex shrink-0 flex-wrap items-center gap-2">
          <Button
            v-if="review.status === 'error' || review.status === 'cancelled'"
            variant="outline"
            size="sm"
            :loading="pendingAction === 'retry'"
            @click="handleRetry"
          >
            Retry
          </Button>
          <Button
            variant="outline"
            size="sm"
            :loading="pendingAction === 'archive' || pendingAction === 'unarchive'"
            @click="handleToggleArchive"
          >
            {{ review.archived ? 'Unarchive' : 'Archive' }}
          </Button>
          <Button
            v-if="review.status === 'awaiting_approval'"
            variant="outline"
            size="sm"
            :loading="pendingAction === 'approve'"
            @click="handleApprove"
          >
            Approve
          </Button>
          <ConfirmDialog
            v-if="review.status === 'done'"
            title="Publish all unpublished findings and the summary to the MR?"
            description="Posts comments to the live merge request. This can't be undone."
            confirm-label="Publish all"
            :danger="false"
            :pending="pendingPublishTarget === 'all'"
            @confirm="handlePublishAll"
          >
            <template #trigger>
              <Button variant="outline" size="sm" :disabled="!hasUnpublished(review)">Publish all</Button>
            </template>
          </ConfirmDialog>
          <ConfirmDialog
            :title='`Discard review !${review.mrIid}?`'
            description="This permanently removes the review. This cannot be undone."
            confirm-label="Discard"
            danger
            :pending="pendingAction === 'discard'"
            @confirm="handleDiscard"
          >
            <template #trigger>
              <Button variant="ghost" size="sm" class="text-danger-text hover:bg-danger-bg">Discard</Button>
            </template>
          </ConfirmDialog>
        </div>
      </div>

      <Alert v-if="review.error" status="danger" title="Review error">
        {{ review.error }}
      </Alert>

      <div
        v-if="review.status === 'done'"
        class="flex flex-wrap items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <Field v-if="hasReadyProfile" label="Voice profile" class="w-56" v-slot="{ id }">
          <Select :id="id" v-model="profileId" :items="profileSelectItems" />
        </Field>
        <Text v-else muted size="sm">
          <RouterLink to="/settings" class="font-medium underline underline-offset-2">Add a ready profile</RouterLink>
          to humanize findings and the summary.
        </Text>
        <Button
          variant="outline"
          size="sm"
          :disabled="!hasReadyProfile"
          :loading="isHumanizingAll"
          @click="humanizeAll()"
        >
          Humanize all
        </Button>
      </div>

      <div v-if="review.summary" class="rounded-lg border border-line-subtle bg-bg-panel p-4">
        <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
          <Text as="h2" size="lg" class="font-semibold">Summary</Text>
          <div class="flex flex-wrap items-center gap-2">
            <Button
              v-if="review.status === 'done'"
              variant="ghost"
              size="sm"
              :disabled="!hasReadyProfile"
              :loading="isHumanizingSummary"
              @click="humanizeSummary()"
            >
              Humanize
            </Button>
            <Badge v-if="review.status === 'done' && review.summaryPublished" status="success">Published</Badge>
            <ConfirmDialog
              v-else-if="review.status === 'done'"
              title="Publish the summary to the MR?"
              description="Posts a comment to the live merge request. This can't be undone."
              confirm-label="Publish"
              :danger="false"
              :pending="pendingPublishTarget === 'summary'"
              @confirm="handlePublishSummary"
            >
              <template #trigger>
                <Button variant="outline" size="sm">Publish</Button>
              </template>
            </ConfirmDialog>
          </div>
        </div>
        <HumanizeTabs
          v-if="review.status === 'done' && summaryTabs > 0"
          :tab-count="summaryTabs"
          :active="summaryTab"
          class="mb-2"
          @select="setSummaryTab"
        />
        <Text class="whitespace-pre-wrap">{{ review.status === 'done' ? activeSummaryText(review) : review.summary }}</Text>
      </div>

      <div class="flex flex-col gap-4">
        <Text as="h2" size="lg" class="font-semibold">Findings</Text>

        <Text v-if="review.findings.length === 0" muted size="sm">No findings.</Text>

        <template v-else>
          <div v-for="dimension in FINDING_DIMENSIONS" :key="dimension">
            <div v-if="groupedFindings![dimension].length > 0" class="flex flex-col gap-2">
              <Text as="h3" size="md" class="font-medium capitalize">{{ dimension }}</Text>
              <ul class="flex flex-col gap-2">
                <li
                  v-for="finding in groupedFindings![dimension]"
                  :key="finding.index"
                  class="flex flex-col gap-1 rounded-lg border border-line-subtle bg-bg-panel p-3"
                >
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <Badge :status="FINDING_SEVERITY_BADGE[finding.severity]">{{ finding.severity }}</Badge>
                      <Badge v-if="finding.blocking" status="danger">Blocking</Badge>
                      <Text mono size="sm" muted>{{ finding.file }}:{{ finding.line }}</Text>
                    </div>
                    <div class="flex flex-wrap items-center gap-2">
                      <Button
                        v-if="review.status === 'done'"
                        variant="ghost"
                        size="sm"
                        :disabled="!hasReadyProfile"
                        :loading="isHumanizingFinding(finding.index)"
                        @click="humanizeFinding(finding.index)"
                      >
                        Humanize
                      </Button>
                      <Badge v-if="review.status === 'done' && finding.published" status="success">Published</Badge>
                      <ConfirmDialog
                        v-else-if="review.status === 'done'"
                        title="Publish this finding to the MR?"
                        description="Posts a comment to the live merge request. This can't be undone."
                        confirm-label="Publish"
                        :danger="false"
                        :pending="pendingPublishTarget === `finding:${finding.index}`"
                        @confirm="handlePublishFinding(finding)"
                      >
                        <template #trigger>
                          <Button variant="outline" size="sm">Publish</Button>
                        </template>
                      </ConfirmDialog>
                    </div>
                  </div>
                  <HumanizeTabs
                    v-if="review.status === 'done' && findingTabs(finding.index) > 0"
                    :tab-count="findingTabs(finding.index)"
                    :active="findingTab(finding.index)"
                    @select="(tab) => setFindingTab(finding.index, tab)"
                  />
                  <Text class="font-medium">{{
                    review.status === 'done' ? activeFindingParts(finding).issue : finding.issue
                  }}</Text>
                  <Text muted size="sm">{{ review.status === 'done' ? activeFindingParts(finding).why : finding.why }}</Text>
                  <Text muted size="sm">
                    Fix: {{ review.status === 'done' ? activeFindingParts(finding).fix : finding.fix }}
                  </Text>
                </li>
              </ul>
            </div>
          </div>
        </template>
      </div>

      <details v-if="review.reasonings.length > 0" class="rounded-lg border border-line-subtle bg-bg-panel">
        <summary class="cursor-pointer select-none px-4 py-3 text-sm font-medium text-text">Reasoning</summary>
        <div class="flex flex-col gap-3 border-t border-line-subtle px-4 py-3">
          <div v-for="(reasoning, i) in review.reasonings" :key="i" class="flex flex-col gap-1">
            <Text size="sm" class="font-medium">{{ reasoning.phase }}</Text>
            <Text muted size="sm" class="whitespace-pre-wrap">{{ reasoning.content }}</Text>
          </div>
        </div>
      </details>
    </div>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Review" } }
</route>
