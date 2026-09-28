<script setup lang="ts">
/**
 * MRs tab — organism, the Flow workspace's default tab. Lists a repo's open
 * merge requests (`GET /repos/{id}/merge-requests`, `@pinia/colada` query
 * keyed by repo id), each row optionally showing its most recent review
 * status when one already exists — read from the repo's reviews list under
 * the exact same query key (`repoReviewsQueryKey`) `RepoReviewsSection`
 * uses, so visiting/switching both tabs for the same repo only ever
 * fetches reviews once (colada's cache dedupes by key).
 *
 * Each row's `#actions` slot defaults to a "Review" button that opens
 * `ReviewLaunchDialog` for that MR (slice 2) — still a real scoped slot, so
 * a future slice (Release/New MR) can still override it per call site. The
 * `#header-actions` slot stays empty (New MR lands in a later slice).
 *
 * `repo` is looked up from `useReposStore().repos` — the exact same global
 * list the Flow page (`src/pages/flow/[repoId].vue`) already reads, so this
 * causes no extra fetch. It's only needed for the dialog (repo's own
 * provider/model defaults); `undefined` while the list hasn't settled yet
 * just means "Review" briefly has nothing to open against.
 */
import { computed, ref } from 'vue'
import { useQuery } from '@pinia/colada'
import { Alert, Button, Icon, Skeleton, Text } from '@shared/ui/design-system'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { listMergeRequests } from '@modules/repos/api'
import type { MergeRequest } from '@modules/repos/types'
import { useReposStore } from '@modules/repos/store'
import { listRepoReviews, repoReviewsQueryKey } from '@modules/reviews/api'
import { isReviewActive } from '@modules/reviews'
import { ReviewStatusChip } from '@modules/reviews'
import { latestReviewByMr } from '../mergeRequests'
import ReviewLaunchDialog from './ReviewLaunchDialog.vue'

const props = defineProps<{ repoId: string }>()

defineSlots<{
  'header-actions'?: () => unknown
  actions?: (props: { mr: MergeRequest }) => unknown
}>()

const reposStore = useReposStore()
const repo = computed(() => reposStore.repos.find((r) => r.id === props.repoId))

const {
  data: mrsData,
  isLoading,
  error,
  refetch,
} = useQuery({
  key: () => ['flow-merge-requests', props.repoId] as const,
  query: () => listMergeRequests(props.repoId),
})
const mergeRequests = computed(() => mrsData.value ?? [])

// Best-effort/optional — see the file doc. Shares its query key with
// `RepoReviewsSection`, so this never causes a second network round trip
// once either tab has already warmed the cache for this repo.
const { data: reviewsData } = useQuery({
  key: () => repoReviewsQueryKey(props.repoId),
  query: () => listRepoReviews(props.repoId),
})
const reviewStatusByMr = computed(() => latestReviewByMr(reviewsData.value ?? []))

// The latest review of the MR the dialog is open for, when it's still pending/running.
const activeReviewForDialog = computed(() => {
  if (!reviewDialogMr.value) return null
  const latest = reviewStatusByMr.value.get(reviewDialogMr.value.iid)
  return latest && isReviewActive(latest.status) ? { id: latest.id, status: latest.status } : null
})

const reviewDialogOpen = ref(false)
const reviewDialogMr = ref<MergeRequest | null>(null)

function openReviewDialog(mr: MergeRequest) {
  reviewDialogMr.value = mr
  reviewDialogOpen.value = true
}
</script>

<template>
  <section class="flex flex-col gap-3">
    <div class="flex items-center justify-end gap-2">
      <slot name="header-actions" />
    </div>

    <div v-if="isLoading" class="flex flex-col gap-2" data-testid="flow-mrs-loading-skeleton">
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <Skeleton class="h-4 w-56" />
      </div>
    </div>

    <Alert v-else-if="error" status="danger">
      <p>{{ resolveErrorMessage(error, 'Failed to load merge requests') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="mergeRequests.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="git-pull-request" size="lg" class="text-text-muted" />
      <Text muted>No open merge requests.</Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="mr in mergeRequests"
        :key="mr.iid"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
        data-testid="flow-mr-row"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text muted size="sm" mono>!{{ mr.iid }}</Text>
            <a
              v-if="mr.webUrl"
              :href="mr.webUrl"
              target="_blank"
              rel="noopener"
              class="flex min-w-0 items-center gap-1 truncate font-semibold text-text hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <span class="truncate">{{ mr.title }}</span>
              <Icon name="external-link" size="xs" class="shrink-0 text-text-muted" />
            </a>
            <Text v-else class="truncate font-semibold">{{ mr.title }}</Text>
            <ReviewStatusChip
              v-if="reviewStatusByMr.get(mr.iid)"
              :status="reviewStatusByMr.get(mr.iid)!.status"
            />
          </div>
          <Text muted size="sm" class="truncate font-mono">{{ mr.sourceBranch }} → {{ mr.targetBranch }}</Text>
          <Text muted size="xs">{{ mr.author }}</Text>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <slot name="actions" :mr="mr">
            <Button type="button" variant="outline" size="sm" @click.stop="openReviewDialog(mr)">
              Review
            </Button>
          </slot>
        </div>
      </li>
    </ul>

    <ReviewLaunchDialog
      v-if="repo"
      v-model:open="reviewDialogOpen"
      :repo="repo"
      :merge-request="reviewDialogMr"
      :active-review="activeReviewForDialog"
    />
  </section>
</template>
