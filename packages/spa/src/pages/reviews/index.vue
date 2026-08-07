<script setup lang="ts">
definePage({ meta: { title: 'Reviews' } })
import { computed, onMounted, ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import Skeleton from '@shared/components/ui/Skeleton.vue'
import ScoreMeter from '@shared/components/charts/ScoreMeter.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import {
  formatDateTime,
  isTerminal,
  recommendationClass,
  recommendationLabel,
  shortId,
} from '@modules/reviews/format'
import { toast } from '@shared/composables/useToast'
import { confirm } from '@shared/composables/useConfirm'
import { errorMessage } from '@shared/api/client'

const repos = useReposStore()
const reviews = useReviewsStore()
const loading = ref(false)
// Ids with an archive/unarchive request in flight, so concurrent clicks on
// different rows each keep their own button disabled independently.
const busyIds = ref<string[]>([])

const showArchived = ref(false)
const archivedLoaded = ref(false)

async function archiveReview(id: string) {
  busyIds.value = [...busyIds.value, id]
  try {
    await reviews.archive(id)
    if (showArchived.value) await fetchAllArchived()
    toast.success('Review archived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}

async function unarchiveReview(id: string, repoId: string) {
  busyIds.value = [...busyIds.value, id]
  try {
    await reviews.unarchive(id)
    // unarchive drops the repo's active cache slot; repopulate it so the review
    // reappears in the active list.
    await reviews.fetchReviews(repoId)
    toast.success('Review unarchived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}

async function approveReview(id: string) {
  busyIds.value = [...busyIds.value, id]
  try {
    await reviews.approve(id)
    toast.success('Review approved — running now')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}

async function retryReview(id: string) {
  busyIds.value = [...busyIds.value, id]
  try {
    await reviews.retry(id)
    toast.success('Retrying — a fresh review is running')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}

async function discardReview(id: string, mrIid: number) {
  const ok = await confirm({
    title: 'Discard review',
    message: `Discard the held review for !${mrIid}? This cannot be undone.`,
    danger: true,
  })
  if (!ok) return
  busyIds.value = [...busyIds.value, id]
  try {
    await reviews.remove(id)
    toast.success('Review discarded')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyIds.value = busyIds.value.filter((x) => x !== id)
  }
}

async function fetchAllArchived() {
  await Promise.all(repos.items.map((r) => reviews.fetchArchivedReviews(r.id)))
  archivedLoaded.value = true
}

async function toggleArchived() {
  showArchived.value = !showArchived.value
  if (showArchived.value && !archivedLoaded.value) await fetchAllArchived()
}

const repoName = (id: string) => repos.items.find((r) => r.id === id)?.name ?? 'repo'

async function load() {
  loading.value = reviews.allReviews.length === 0
  if (repos.items.length === 0) await repos.fetchAll()
  await Promise.all(repos.items.map((r) => reviews.fetchReviews(r.id)))
  loading.value = false
}
onMounted(load)

const items = computed(() => reviews.allReviews)
const archived = computed(() => reviews.allArchived)

// --- Status filter + attention sort ---
// A self-hoster opens this list to answer "what needs me?" — so it must be
// filterable by status and float the attention-needing reviews to the top.
type StatusFilter = 'all' | 'awaiting_approval' | 'running' | 'done' | 'error'
const statusFilter = ref<StatusFilter>('all')
const FILTERS: { key: StatusFilter; label: string }[] = [
  { key: 'all', label: 'All' },
  { key: 'awaiting_approval', label: 'Awaiting' },
  { key: 'running', label: 'Running' },
  { key: 'done', label: 'Done' },
  { key: 'error', label: 'Failed' },
]
function matchesFilter(status: string, f: StatusFilter): boolean {
  if (f === 'all') return true
  if (f === 'running') return status === 'running' || status === 'pending'
  return status === f
}
const filterCount = (f: StatusFilter) =>
  items.value.filter((rv) => matchesFilter(rv.status, f)).length
// Attention first: awaiting_approval (waiting on you), then error, then the
// store's newest-first order.
const attentionRank = (s: string) => (s === 'awaiting_approval' ? 0 : s === 'error' ? 1 : 2)
const visibleItems = computed(() =>
  items.value.filter((rv) => matchesFilter(rv.status, statusFilter.value)).sort(
    (a, b) => attentionRank(a.status) - attentionRank(b.status),
  ),
)

// Bulk archive: every finished (terminal) review that isn't archived yet. Running
// and awaiting-approval reviews are left alone — only "old" ones are swept. Archive
// is reversible (Show archived → Unarchive), so this is a safe declutter.
const archiving = ref(false)
const archivable = computed(() => items.value.filter((rv) => isTerminal(rv.status)))
async function archiveAll() {
  const targets = [...archivable.value]
  if (targets.length === 0 || archiving.value) return
  const ok = await confirm({
    title: 'Archive finished reviews',
    message: `Archive ${targets.length} finished ${targets.length === 1 ? 'review' : 'reviews'}? They move to the archived list — you can restore them anytime.`,
  })
  if (!ok) return
  archiving.value = true
  try {
    for (const rv of targets) await reviews.archive(rv.id)
    toast.success(`Archived ${targets.length} ${targets.length === 1 ? 'review' : 'reviews'}`)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archiving.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="Reviews" />

    <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div class="flex flex-wrap items-center gap-1.5" role="group" aria-label="Filter by status">
        <button
          v-for="f in FILTERS"
          :key="f.key"
          type="button"
          class="border px-2 py-1 font-mono text-[0.66rem] tracking-wider uppercase transition-colors"
          :class="
            statusFilter === f.key
              ? 'border-accent text-ink bg-accent/10'
              : 'border-line text-muted hover:text-ink'
          "
          :aria-pressed="statusFilter === f.key"
          @click="statusFilter = f.key"
        >
          {{ f.label }} <span class="text-muted">{{ filterCount(f.key) }}</span>
        </button>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="archivable.length > 0"
          class="btn-ghost text-xs"
          :disabled="archiving"
          title="Archive every finished review"
          @click="archiveAll"
        >
          <span
            :class="archiving ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive'"
            class="text-sm"
            aria-hidden="true"
          />
          Archive all ({{ archivable.length }})
        </button>
        <button class="btn-ghost text-xs" @click="toggleArchived">
          <span
            :class="showArchived ? 'i-lucide-eye-off' : 'i-lucide-archive'"
            class="text-sm"
            aria-hidden="true"
          />
          {{ showArchived ? 'Hide archived' : 'Show archived' }}
        </button>
      </div>
    </div>

    <!-- Load error surfaced distinctly so a failed fetch never masquerades as an
         empty "No reviews yet" state. -->
    <div
      v-if="reviews.listError && !loading"
      class="border-danger/40 bg-danger/5 mb-3 flex flex-wrap items-center justify-between gap-3 border px-4 py-3"
    >
      <span class="text-danger flex items-center gap-2 text-sm">
        <span class="i-lucide-triangle-alert shrink-0 text-base" aria-hidden="true" />
        Couldn't load reviews — {{ reviews.listError }}
      </span>
      <button class="btn-line text-xs" @click="load">
        <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
        Retry
      </button>
    </div>

    <!-- Skeleton rows while the first load is in flight and nothing is cached. -->
    <ul v-if="loading" class="border-line/50 border-t" aria-hidden="true">
      <li v-for="n in 5" :key="n" class="row justify-between">
        <div class="min-w-0 space-y-2">
          <div class="flex items-center gap-3">
            <Skeleton w="w-48" h="h-4" />
            <Skeleton w="w-16" h="h-3" />
          </div>
          <Skeleton w="w-56" h="h-3" />
        </div>
        <Skeleton w="w-16" h="h-4" />
      </li>
    </ul>
    <EmptyState
      v-else-if="visibleItems.length === 0 && !reviews.listError"
      icon="i-lucide-list-checks"
      :title="items.length === 0 ? 'No reviews yet' : 'No matching reviews'"
      :hint="
        items.length === 0
          ? 'Open a repository and start a review on a merge request.'
          : 'No reviews match this status filter.'
      "
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="rv in visibleItems" :key="rv.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <RouterLink
              :to="`/reviews/${rv.id}`"
              class="text-ink text-sm hover:underline"
              :style="{ viewTransitionName: `review-${rv.id}` }"
            >
              {{ repoName(rv.repoId) }} · !{{ rv.mrIid }}
            </RouterLink>
            <ReviewStatusChip :status="rv.status" />
          </div>
          <div class="label-mono mt-0.5 flex flex-wrap gap-x-2">
            <span class="text-muted">#{{ shortId(rv.id) }}</span>
            <span>{{ rv.contextMode }}</span>
            <span v-if="rv.createdAt">{{ formatDateTime(rv.createdAt) }}</span>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-3">
          <div v-if="rv.status === 'done'" class="text-right">
            <div class="text-sm" :class="recommendationClass[rv.recommendation]">
              {{ recommendationLabel(rv.recommendation) }}
            </div>
            <div class="label-mono mt-0.5">score {{ rv.score }}</div>
            <ScoreMeter :value="rv.score" class="mt-1 ml-auto w-16" />
          </div>
          <button
            v-if="rv.status === 'error'"
            class="btn-line text-xs"
            :disabled="busyIds.includes(rv.id)"
            :aria-label="`Retry review !${rv.mrIid}`"
            title="Retry — clones this review and re-runs it"
            @click="retryReview(rv.id)"
          >
            <span
              :class="
                busyIds.includes(rv.id)
                  ? 'i-lucide-loader-circle animate-spin'
                  : 'i-lucide-refresh-cw'
              "
              class="text-sm"
              aria-hidden="true"
            />
            Retry
          </button>
          <template v-if="rv.status === 'awaiting_approval'">
            <button
              class="btn-line text-xs"
              :disabled="busyIds.includes(rv.id)"
              :aria-label="`Approve review !${rv.mrIid}`"
              title="Approve and run"
              @click="approveReview(rv.id)"
            >
              <span
                :class="
                  busyIds.includes(rv.id) ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-play'
                "
                class="text-sm"
                aria-hidden="true"
              />
              Approve
            </button>
            <button
              class="btn-ghost text-danger text-xs"
              :disabled="busyIds.includes(rv.id)"
              :aria-label="`Discard review !${rv.mrIid}`"
              title="Discard"
              @click="discardReview(rv.id, rv.mrIid)"
            >
              <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
            </button>
          </template>
          <button
            v-else
            class="btn-ghost text-xs"
            :disabled="busyIds.includes(rv.id) || !isTerminal(rv.status)"
            :aria-label="`Archive review !${rv.mrIid}`"
            :title="isTerminal(rv.status) ? 'Archive' : 'Cannot archive a running review'"
            @click="archiveReview(rv.id)"
          >
            <span class="i-lucide-archive text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>

    <template v-if="showArchived">
      <h2 class="section-title text-muted mt-8 mb-3 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Archived
      </h2>
      <p v-if="reviews.archivedLoading && archived.length === 0" class="text-muted py-3 text-sm">
        Loading…
      </p>
      <p v-else-if="archived.length === 0" class="text-muted py-3 text-sm">No archived reviews.</p>
      <ul v-else class="border-line/50 border-t">
        <li v-for="rv in archived" :key="rv.id" class="row justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
              <RouterLink
                :to="`/reviews/${rv.id}`"
                class="text-ink text-sm hover:underline"
                :style="{ viewTransitionName: `review-${rv.id}` }"
              >
                {{ repoName(rv.repoId) }} · !{{ rv.mrIid }}
              </RouterLink>
              <ReviewStatusChip :status="rv.status" />
            </div>
            <div class="label-mono mt-0.5 flex flex-wrap gap-x-2">
              <span class="text-muted">#{{ shortId(rv.id) }}</span>
              <span>{{ rv.contextMode }}</span>
              <span v-if="rv.createdAt">{{ formatDateTime(rv.createdAt) }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-3">
            <div v-if="rv.status === 'done'" class="text-right">
              <div class="text-sm" :class="recommendationClass[rv.recommendation]">
                {{ recommendationLabel(rv.recommendation) }}
              </div>
              <div class="label-mono mt-0.5">score {{ rv.score }}</div>
            </div>
            <button
              class="btn-ghost text-xs"
              :disabled="busyIds.includes(rv.id)"
              :aria-label="`Unarchive review !${rv.mrIid}`"
              title="Unarchive"
              @click="unarchiveReview(rv.id, rv.repoId)"
            >
              <span class="i-lucide-archive-restore text-sm" aria-hidden="true" />
            </button>
          </div>
        </li>
      </ul>
    </template>
  </div>
</template>
