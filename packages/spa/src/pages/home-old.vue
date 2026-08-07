<script setup lang="ts">
definePage({ meta: { title: 'Old overview' } })
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
  const jobs: Promise<unknown>[] = []
  if (repos.items.length === 0) jobs.push(repos.fetchAll())
  if (accounts.items.length === 0) jobs.push(accounts.fetchAll())
  if (providers.items.length === 0) jobs.push(providers.fetchAll())
  await Promise.all(jobs)
  await Promise.all(repos.items.map((r) => reviews.fetchReviews(r.id)))
  reviewsLoading.value = false
  void routines.listRecent()
})

const reviewCount = computed(() => reviews.allReviews.length)
const repoCount = computed(() => repos.items.length)
const accountCount = computed(() => accounts.items.length)
const providerCount = computed(() => providers.items.length)
const setupComplete = computed(() => reviewCount.value > 0)
const inProgress = computed(() => activity.count)
const awaitingCount = computed(
  () => reviews.allReviews.filter((r) => r.status === 'awaiting_approval').length,
)

const recentReviews = computed(() => reviews.allReviews.slice(0, 7))
const recentRuns = computed(() => routines.recentRuns.slice(0, 5))
const runsLoading = computed(() => routines.listLoading && routines.recentRuns.length === 0)

const repoName = (id: string) => repos.items.find((r) => r.id === id)?.name ?? 'repo'
const reviewsForRepo = (id: string) => reviews.reviewsFor(id).length

const recentRepos = computed(() => repos.items.slice(0, 6))
const reposLoading = computed(() => repos.loading && repos.items.length === 0)

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
const scoreTrend = computed(() =>
  done.value
    .slice(0, 12)
    .map((r) => r.score)
    .reverse(),
)

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

const statTile = 'border border-line/60 bg-surface/40 p-4 transition-colors'
</script>

<template>
  <div>
    <PageHeader title="Overview" label="Legacy version" />

    <!-- Kept for side-by-side comparison against the rebuilt overview at "/". -->
    <RouterLink
      to="/"
      class="text-muted hover:text-ink mb-6 inline-flex items-center gap-1 text-xs transition-colors"
    >
      <span class="i-lucide-arrow-left" aria-hidden="true" />
      Back to the current overview
    </RouterLink>

    <SetupChecklist
      v-if="!reviewsLoading && !setupComplete"
      :accounts="accountCount"
      :providers="providerCount"
      :repos="repoCount"
      :reviews="reviewCount"
      class="mb-8"
    />

    <RouterLink
      v-if="awaitingCount > 0"
      to="/reviews"
      class="group border-warn/40 bg-warn/5 hover:border-warn mb-4 flex items-center justify-between gap-3 border px-4 py-3 transition-colors"
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

    <div class="grid grid-cols-3 gap-3">
      <RouterLink to="/reviews" :class="[statTile, 'hover:border-ink flex flex-col justify-between']">
        <div class="label-mono">Reviews</div>
        <div class="text-ink mt-3 font-mono text-2xl font-semibold sm:text-3xl">
          {{ reviewCount }}
        </div>
      </RouterLink>
      <RouterLink to="/repos" :class="[statTile, 'hover:border-ink flex flex-col justify-between']">
        <div class="label-mono">Repositories</div>
        <div class="text-ink mt-3 font-mono text-2xl font-semibold sm:text-3xl">
          {{ repoCount }}
        </div>
      </RouterLink>
      <div :class="[statTile, 'flex flex-col justify-between']">
        <div class="label-mono">In progress</div>
        <div class="mt-3 flex items-center gap-2">
          <span
            v-if="inProgress > 0"
            class="i-lucide-loader-circle text-accent animate-spin text-lg"
            aria-hidden="true"
          />
          <span
            class="font-mono text-2xl font-semibold sm:text-3xl"
            :class="inProgress > 0 ? 'text-accent' : 'text-ink'"
          >
            {{ inProgress }}
          </span>
        </div>
      </div>
    </div>

    <div class="mt-8 grid grid-cols-1 gap-x-8 gap-y-8 lg:grid-cols-2">
      <section aria-labelledby="recent-reviews-heading-old">
        <div class="mb-1 flex items-end justify-between gap-4">
          <h2 id="recent-reviews-heading-old" class="section-title flex items-center gap-2">
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
        />
        <ul v-else class="border-line/50 border-t">
          <li v-for="rv in recentReviews" :key="rv.id" class="row justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                <RouterLink
                  :to="`/reviews/${rv.id}`"
                  class="text-ink hover:text-ink min-w-0 truncate text-sm"
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
              </template>
              <div v-else-if="rv.status === 'error'" class="text-danger text-xs">failed</div>
            </div>
          </li>
        </ul>
      </section>

      <section aria-labelledby="recent-actions-heading-old">
        <div class="mb-1 flex items-end justify-between gap-4">
          <h2 id="recent-actions-heading-old" class="section-title flex items-center gap-2">
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

    <section class="mt-8" aria-labelledby="repos-heading-old">
      <div class="mb-3 flex items-end justify-between gap-4">
        <h2 id="repos-heading-old" class="section-title flex items-center gap-2">
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

      <div v-if="reposLoading" class="grid grid-cols-2 gap-3 sm:grid-cols-3" aria-hidden="true">
        <div v-for="n in 3" :key="n" :class="statTile">
          <Skeleton w="w-28" h="h-4" />
          <Skeleton w="w-16" h="h-3" class="mt-3" />
        </div>
      </div>
      <EmptyState
        v-else-if="recentRepos.length === 0"
        icon="i-lucide-folder-git-2"
        title="No repositories tracked"
        hint="Track a GitLab repository to start reviewing its merge requests."
      />
      <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <RouterLink
          v-for="r in recentRepos"
          :key="r.id"
          :to="`/repos/${r.id}`"
          :class="[statTile, 'group hover:border-ink flex flex-col justify-between']"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="text-ink group-hover:text-ink min-w-0 truncate text-sm">{{ r.name }}</span>
            <span
              class="i-lucide-arrow-up-right text-muted group-hover:text-ink shrink-0"
              aria-hidden="true"
            />
          </div>
          <div class="label-mono mt-2">{{ reviewsForRepo(r.id) }} reviews</div>
        </RouterLink>
      </div>
    </section>

    <section v-if="hasData" class="mt-8" aria-labelledby="stats-heading-old">
      <h2 id="stats-heading-old" class="section-title mb-3 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Stats
      </h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div :class="[statTile, 'flex flex-col']">
          <div class="label-mono">Average score</div>
          <div class="mt-2">
            <div class="text-ink font-mono text-3xl font-semibold">
              {{ avgScore }}<span class="text-muted text-base">/100</span>
            </div>
            <ScoreMeter :value="avgScore" class="mt-3" />
          </div>
        </div>

        <div :class="[statTile, 'flex flex-col']">
          <div class="label-mono mb-3">Recommendations</div>
          <RecommendationBar
            :approve="recCounts.approve"
            :request-changes="recCounts.requestChanges"
            :comment="recCounts.comment"
          />
        </div>

        <div :class="[statTile, 'flex flex-col md:col-span-2']">
          <div class="label-mono mb-3">Score trend</div>
          <Sparkline v-if="scoreTrend.length" :values="scoreTrend" />
        </div>
      </div>
    </section>

    <section class="mt-8" aria-labelledby="shortcuts-heading-old">
      <h2 id="shortcuts-heading-old" class="section-title mb-3 flex items-center gap-2">
        <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
        Quick access
      </h2>
      <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
        <RouterLink
          v-for="s in shortcuts"
          :key="s.to"
          :to="s.to"
          :class="[statTile, 'group hover:border-ink flex items-center justify-between']"
        >
          <div class="flex items-center gap-3">
            <span :class="s.icon" class="text-muted group-hover:text-ink text-lg" aria-hidden="true" />
            <span class="text-ink text-sm">{{ s.label }}</span>
          </div>
          <span class="i-lucide-arrow-up-right text-muted group-hover:text-ink" aria-hidden="true" />
        </RouterLink>
      </div>
    </section>
  </div>
</template>
