<script setup lang="ts">
import { computed, onMounted } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import ScoreMeter from '@shared/components/charts/ScoreMeter.vue'
import Sparkline from '@shared/components/charts/Sparkline.vue'
import RecommendationBar from '@shared/components/charts/RecommendationBar.vue'
import { useReposStore } from '@modules/repos/store'
import { useProvidersStore } from '@modules/providers/store'
import { useReviewsStore } from '@modules/reviews/store'

const repos = useReposStore()
const providers = useProvidersStore()
const reviews = useReviewsStore()

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
  void Promise.all(repos.items.map((r) => reviews.fetchReviews(r.id)))
})

const reviewCount = computed(() => reviews.allReviews.length)
const repoCount = computed(() => repos.items.length)
const providerCount = computed(() => providers.items.length)

// Metrics from the list data (no findings needed).
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

const shortcuts = [
  { to: '/repos', label: 'Repositories', icon: 'i-lucide-folder-git-2' },
  { to: '/reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { to: '/accounts', label: 'GitLab accounts', icon: 'i-lucide-users' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu' },
  { to: '/profiles', label: 'Profiles', icon: 'i-lucide-feather' },
  { to: '/skills', label: '4R skills', icon: 'i-lucide-book-open' },
]

const cardBase = 'border border-line/60 bg-surface/40 p-5 transition-colors'
</script>

<template>
  <div>
    <PageHeader title="Overview" />

    <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
      <!-- Hero stat -->
      <RouterLink
        to="/reviews"
        :class="[cardBase, 'group hover:border-ink col-span-2 flex flex-col justify-between']"
      >
        <div class="label-mono">Reviews run</div>
        <div class="text-ink mt-6 font-mono text-5xl font-semibold">{{ reviewCount }}</div>
        <div class="text-accent mt-4 inline-flex items-center gap-1 text-sm">
          View all
          <span
            class="i-lucide-arrow-right transition-transform group-hover:translate-x-0.5"
            aria-hidden="true"
          />
        </div>
      </RouterLink>

      <RouterLink to="/repos" :class="[cardBase, 'hover:border-ink flex flex-col justify-between']">
        <div class="label-mono">Repositories</div>
        <div class="text-ink mt-4 font-mono text-3xl font-semibold">{{ repoCount }}</div>
      </RouterLink>
      <RouterLink
        to="/providers"
        :class="[cardBase, 'hover:border-ink flex flex-col justify-between']"
      >
        <div class="label-mono">Providers</div>
        <div class="text-ink mt-4 font-mono text-3xl font-semibold">{{ providerCount }}</div>
      </RouterLink>

      <!-- Average score -->
      <div :class="[cardBase, 'col-span-2 flex flex-col']">
        <div class="label-mono">Average score</div>
        <div v-if="hasData" class="mt-2">
          <div class="text-ink font-mono text-3xl font-semibold">
            {{ avgScore }}<span class="text-muted text-base">/100</span>
          </div>
          <ScoreMeter :value="avgScore" class="mt-3" />
        </div>
        <div v-else class="text-muted mt-4 text-sm">No completed reviews yet.</div>
      </div>

      <!-- Recommendation split -->
      <div :class="[cardBase, 'col-span-2 flex flex-col']">
        <div class="label-mono mb-3">Recommendations</div>
        <RecommendationBar
          v-if="hasData"
          :approve="recCounts.approve"
          :request-changes="recCounts.requestChanges"
          :comment="recCounts.comment"
        />
        <div v-else class="text-muted text-sm">No completed reviews yet.</div>
      </div>

      <!-- Score trend -->
      <div :class="[cardBase, 'col-span-2 flex flex-col md:col-span-4']">
        <div class="label-mono mb-3">Score trend</div>
        <Sparkline v-if="scoreTrend.length" :values="scoreTrend" />
        <div v-else class="text-muted text-sm">No completed reviews yet.</div>
      </div>

      <!-- Shortcuts -->
      <RouterLink
        v-for="(s, i) in shortcuts"
        :key="s.to"
        :to="s.to"
        :class="[
          cardBase,
          'group hover:border-ink flex items-center justify-between',
          i === shortcuts.length - 1 ? 'col-span-2 md:col-span-4' : '',
        ]"
      >
        <div class="flex items-center gap-3">
          <span
            :class="s.icon"
            class="text-muted group-hover:text-ink text-lg"
            aria-hidden="true"
          />
          <span class="text-ink text-sm">{{ s.label }}</span>
        </div>
        <span
          class="i-lucide-arrow-up-right text-muted group-hover:text-accent"
          aria-hidden="true"
        />
      </RouterLink>
    </div>
  </div>
</template>
