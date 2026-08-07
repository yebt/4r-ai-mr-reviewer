<script setup lang="ts">
definePage({ meta: { title: 'Overview' } })
import { computed, onMounted, ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import SetupChecklist from '@shared/components/ui/SetupChecklist.vue'
import Skeleton from '@shared/components/ui/Skeleton.vue'
import ScoreMeter from '@shared/components/charts/ScoreMeter.vue'
import Sparkline from '@shared/components/charts/Sparkline.vue'
import RecommendationBar from '@shared/components/charts/RecommendationBar.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useRoutinesStore } from '@modules/routines/store'
import { useActivityStore } from '@modules/activity/store'
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import { recommendationClass, recommendationLabel } from '@modules/reviews/format'
import RoutineRunList from '@modules/routines/components/RoutineRunList.vue'

const repos = useReposStore()
const reviews = useReviewsStore()
const routines = useRoutinesStore()
const activity = useActivityStore()
const accounts = useAccountsStore()
const providers = useProvidersStore()

// Stale-while-revalidate: only show a skeleton on the very first load, when
// there is nothing cached to display yet.
const reviewsLoading = ref(false)

onMounted(async () => {
  reviewsLoading.value = reviews.allReviews.length === 0
  // Accounts + providers back the first-run checklist counts, so load them
  // alongside repos before the checklist decides which step to point at.
  const jobs: Promise<unknown>[] = []
  if (repos.items.length === 0) jobs.push(repos.fetchAll())
  if (accounts.items.length === 0) jobs.push(accounts.fetchAll())
  if (providers.items.length === 0) jobs.push(providers.fetchAll())
  await Promise.all(jobs)
  await Promise.all(repos.items.map((r) => reviews.fetchReviews(r.id)))
  reviewsLoading.value = false
  void routines.listRecent()
})

// --- Attention strip ---
const reviewCount = computed(() => reviews.allReviews.length)
const repoCount = computed(() => repos.items.length)
const accountCount = computed(() => accounts.items.length)
const providerCount = computed(() => providers.items.length)
// The first-run checklist leads the page until the user has run their first
// review, at which point they are through the core loop and it retires.
const setupComplete = computed(() => reviewCount.value > 0)
const inProgress = computed(() => activity.count)
const awaitingCount = computed(
  () => reviews.allReviews.filter((r) => r.status === 'awaiting_approval').length,
)

// --- Recent activity ---
const recentReviews = computed(() => reviews.allReviews.slice(0, 7))
const recentRuns = computed(() => routines.recentRuns.slice(0, 5))
const runsLoading = computed(() => routines.listLoading && routines.recentRuns.length === 0)

const repoName = (id: string) => repos.items.find((r) => r.id === id)?.name ?? 'repo'
const reviewsForRepo = (id: string) => reviews.reviewsFor(id).length

const recentRepos = computed(() => repos.items.slice(0, 6))
const reposLoading = computed(() => repos.loading && repos.items.length === 0)

// --- Secondary stats: summaries of completed reviews, shown only once there's data. ---
const done = computed(() => reviews.allReviews.filter((r) => r.status === 'done'))
const hasData = computed(() => done.value.length > 0)
const avgScore = computed(() =>
  hasData.value ? Math.round(done.value.reduce((s, r) => s + r.score, 0) / done.value.length) : 0,
)
const recCounts = computed(() => {
  let approve = 0
  let requestChanges = 0
  let comment = 0
  for (const r of done.value) {
    if (r.recommendation === 'approve') approve++
    else if (r.recommendation === 'request_changes') requestChanges++
    else comment++
  }
  return { approve, requestChanges, comment }
})
// Chronological, most recent 12.
const scoreTrend = computed(() =>
  done.value
    .slice(0, 12)
    .map((r) => r.score)
    .reverse(),
)

// Compact relative-time label for the recent-review rows. Not reactive to the
// clock ticking, which is fine: rows re-render whenever the review data changes.
function timeAgo(iso: string): string {
  if (!iso) return ''
  const then = new Date(iso).getTime()
  if (Number.isNaN(then)) return ''
  const secs = Math.round((Date.now() - then) / 1000)
  if (secs < 60) return 'just now'
  const mins = Math.round(secs / 60)
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.round(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  const days = Math.round(hrs / 24)
  if (days < 7) return `${days}d ago`
  const weeks = Math.round(days / 7)
  if (weeks < 5) return `${weeks}w ago`
  const months = Math.round(days / 30)
  if (months < 12) return `${months}mo ago`
  return `${Math.round(days / 365)}y ago`
}

const shortcuts = [
  { to: '/repos', label: 'Repositories', icon: 'i-lucide-folder-git-2' },
  { to: '/reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { to: '/actions', label: 'Actions', icon: 'i-lucide-git-merge' },
  { to: '/accounts', label: 'GitLab accounts', icon: 'i-lucide-users' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu' },
  { to: '/profiles', label: 'Profiles', icon: 'i-lucide-feather' },
  { to: '/skills', label: '4R skills', icon: 'i-lucide-book-open' },
]
</script>

<template>
  <div>
    <PageHeader title="Overview" />

    <!-- First-run guide: leads the page for a fresh self-hoster and retires
         itself once the first review exists. The one deliberately-boxed element,
         because it is the thing a newcomer must not miss. -->
    <SetupChecklist
      v-if="!reviewsLoading && !setupComplete"
      :accounts="accountCount"
      :providers="providerCount"
      :repos="repoCount"
      :reviews="reviewCount"
      class="mb-8"
    />

    <!-- Held for approval: the one moment the page should shout. Warn voice, not
         lime — it is an alarm, not the live signal. -->
    <RouterLink
      v-if="awaitingCount > 0"
      to="/reviews"
      class="group border-warn/40 bg-warn/5 hover:border-warn mb-6 flex items-center justify-between gap-3 border px-4 py-3 transition-colors"
    >
      <span class="text-warn flex items-center gap-2 text-sm">
        <span class="i-lucide-clock text-base" aria-hidden="true" />
        <span>
          <span class="font-semibold">{{ awaitingCount }}</span>
          {{ awaitingCount === 1 ? 'review' : 'reviews' }} awaiting your approval
        </span>
      </span>
      <span
        class="i-lucide-arrow-right text-warn transition-transform group-hover:translate-x-0.5"
        aria-hidden="true"
      />
    </RouterLink>

    <!-- Stat readout: a borderless instrument line framed by hairlines, not a row
         of bezelled tiles. -->
    <div class="border-line/50 flex flex-wrap items-start gap-x-12 gap-y-4 border-y py-4">
      <RouterLink to="/reviews" class="group">
        <div class="label-mono group-hover:text-ink transition-colors">Reviews</div>
        <div class="text-ink mt-1 font-mono text-3xl font-semibold">{{ reviewCount }}</div>
      </RouterLink>
      <RouterLink to="/repos" class="group">
        <div class="label-mono group-hover:text-ink transition-colors">Repositories</div>
        <div class="text-ink mt-1 font-mono text-3xl font-semibold">{{ repoCount }}</div>
      </RouterLink>
      <div>
        <div class="label-mono">In progress</div>
        <div class="mt-1 flex items-center gap-2">
          <span
            v-if="inProgress > 0"
            class="i-lucide-loader-circle text-accent animate-spin text-lg"
            aria-hidden="true"
          />
          <span
            class="font-mono text-3xl font-semibold"
            :class="inProgress > 0 ? 'text-accent' : 'text-ink'"
          >
            {{ inProgress }}
          </span>
        </div>
      </div>
    </div>

    <!-- Recent activity: reviews (the core artifact) + actions, side by side. -->
    <div class="mt-10 grid grid-cols-1 gap-x-10 gap-y-10 lg:grid-cols-2">
      <section aria-labelledby="recent-reviews-heading">
        <div class="mb-1 flex items-end justify-between gap-4">
          <h2 id="recent-reviews-heading" class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Recent reviews
          </h2>
          <RouterLink
            to="/reviews"
            class="text-muted hover:text-ink inline-flex items-center gap-1 text-xs transition-colors"
          >
            View all
            <span class="i-lucide-arrow-right" aria-hidden="true" />
          </RouterLink>
        </div>

        <ul v-if="reviewsLoading" class="border-line/50 border-t" aria-hidden="true">
          <li v-for="n in 5" :key="n" class="row justify-between">
            <div class="min-w-0 space-y-2">
              <div class="flex items-center gap-3">
                <Skeleton w="w-40" h="h-4" />
                <Skeleton w="w-16" h="h-3" />
              </div>
              <Skeleton w="w-24" h="h-3" />
            </div>
            <Skeleton w="w-16" h="h-4" />
          </li>
        </ul>
        <EmptyState
          v-else-if="recentReviews.length === 0"
          icon="i-lucide-list-checks"
          title="No reviews yet"
          hint="Open a repository and start a review on a merge request."
        >
          <template #action>
            <RouterLink to="/repos" class="btn-accent text-xs">
              {{ repoCount > 0 ? 'Open a repository' : 'Track a repository' }}
              <span class="i-lucide-arrow-right text-sm" aria-hidden="true" />
            </RouterLink>
          </template>
        </EmptyState>
        <ul v-else class="border-line/50 border-t">
          <li v-for="rv in recentReviews" :key="rv.id" class="row justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                <RouterLink
                  :to="`/reviews/${rv.id}`"
                  class="text-ink min-w-0 truncate text-sm hover:underline"
                >
                  {{ repoName(rv.repoId) }} · !{{ rv.mrIid }}
                </RouterLink>
                <ReviewStatusChip :status="rv.status" />
              </div>
              <div v-if="rv.createdAt" class="label-mono mt-0.5">{{ timeAgo(rv.createdAt) }}</div>
            </div>
            <div class="shrink-0 text-right">
              <template v-if="rv.status === 'done'">
                <div class="text-sm" :class="recommendationClass[rv.recommendation]">
                  {{ recommendationLabel(rv.recommendation) }}
                </div>
                <div class="label-mono mt-0.5">score {{ rv.score }}</div>
                <ScoreMeter :value="rv.score" class="mt-1 ml-auto w-16" />
              </template>
              <div v-else-if="rv.status === 'error'" class="text-danger text-xs">failed</div>
            </div>
          </li>
        </ul>
      </section>

      <section aria-labelledby="recent-actions-heading">
        <div class="mb-1 flex items-end justify-between gap-4">
          <h2 id="recent-actions-heading" class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Recent actions
          </h2>
          <RouterLink
            to="/actions"
            class="text-muted hover:text-ink inline-flex items-center gap-1 text-xs transition-colors"
          >
            View all
            <span class="i-lucide-arrow-right" aria-hidden="true" />
          </RouterLink>
        </div>
        <RoutineRunList :items="recentRuns" :loading="runsLoading" :error="routines.listError" />
      </section>
    </div>

    <!-- Repositories: a borderless ledger of hairline rows, not a grid of boxes. -->
    <section class="mt-10" aria-labelledby="repos-heading">
      <div class="mb-1 flex items-end justify-between gap-4">
        <h2 id="repos-heading" class="section-title flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Repositories
        </h2>
        <RouterLink
          to="/repos"
          class="text-muted hover:text-ink inline-flex items-center gap-1 text-xs transition-colors"
        >
          View all
          <span class="i-lucide-arrow-right" aria-hidden="true" />
        </RouterLink>
      </div>

      <ul v-if="reposLoading" class="border-line/50 border-t" aria-hidden="true">
        <li v-for="n in 4" :key="n" class="row justify-between">
          <div class="flex items-center gap-2">
            <Skeleton w="w-4" h="h-4" />
            <Skeleton w="w-32" h="h-4" />
          </div>
          <Skeleton w="w-16" h="h-3" />
        </li>
      </ul>
      <EmptyState
        v-else-if="recentRepos.length === 0"
        icon="i-lucide-folder-git-2"
        title="No repositories tracked"
        hint="Track a GitLab repository to start reviewing its merge requests."
      >
        <template #action>
          <RouterLink to="/repos" class="btn-accent text-xs">
            <span class="i-lucide-plus text-sm" aria-hidden="true" />
            Track a repository
          </RouterLink>
        </template>
      </EmptyState>
      <ul v-else class="border-line/50 border-t">
        <li v-for="r in recentRepos" :key="r.id" class="row justify-between gap-3">
          <RouterLink
            :to="`/repos/${r.id}`"
            class="group text-ink flex min-w-0 items-center gap-2 text-sm"
          >
            <span
              class="i-lucide-folder-git-2 text-muted group-hover:text-ink shrink-0 text-sm transition-colors"
              aria-hidden="true"
            />
            <span class="truncate group-hover:underline">{{ r.name }}</span>
          </RouterLink>
          <span class="label-mono shrink-0">{{ reviewsForRepo(r.id) }} reviews</span>
        </li>
      </ul>
    </section>

    <!-- Secondary stats: borderless blocks, only once there's completed-review
         data to summarise (a fresh install never shows empty placeholders). -->
    <section v-if="hasData" class="mt-10" aria-labelledby="stats-heading">
      <h2 id="stats-heading" class="section-title mb-4 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Stats
      </h2>
      <div class="grid grid-cols-1 gap-x-12 gap-y-8 md:grid-cols-2">
        <div>
          <div class="label-mono">Average score</div>
          <div class="text-ink mt-2 font-mono text-3xl font-semibold">
            {{ avgScore }}<span class="text-muted text-base">/100</span>
          </div>
          <ScoreMeter :value="avgScore" class="mt-3" />
        </div>

        <div>
          <div class="label-mono mb-3">Recommendations</div>
          <RecommendationBar
            :approve="recCounts.approve"
            :request-changes="recCounts.requestChanges"
            :comment="recCounts.comment"
          />
        </div>

        <div class="md:col-span-2">
          <div class="label-mono mb-3">Score trend</div>
          <Sparkline v-if="scoreTrend.length" :values="scoreTrend" />
        </div>
      </div>
    </section>

    <!-- Quick access: borderless links, not boxed shortcuts. The command palette
         covers the same ground faster, so this is a quiet index, not a wall. -->
    <section class="mt-10" aria-labelledby="shortcuts-heading">
      <h2 id="shortcuts-heading" class="section-title mb-4 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Quick access
      </h2>
      <div class="flex flex-wrap gap-x-6 gap-y-3">
        <RouterLink
          v-for="s in shortcuts"
          :key="s.to"
          :to="s.to"
          class="text-muted hover:text-ink inline-flex items-center gap-2 text-sm transition-colors"
        >
          <span :class="s.icon" class="text-base" aria-hidden="true" />
          {{ s.label }}
        </RouterLink>
      </div>
      <p class="text-muted mt-4 text-xs">
        Tip: press
        <kbd class="border-line text-muted border px-1 py-0.5 font-mono text-[0.6rem]">⌘K</kbd>
        /
        <kbd class="border-line text-muted border px-1 py-0.5 font-mono text-[0.6rem]">Ctrl&nbsp;K</kbd>
        to search and jump anywhere.
      </p>
    </section>
  </div>
</template>
