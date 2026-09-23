<script setup lang="ts">
/**
 * Home dashboard. Renders inside AppShell (see src/core/App.vue). Composes
 * the global runs list (`useRunsStore`) + repos list (`useReposStore`) with
 * the pure selectors in `@modules/runs/dashboard.ts` into a stat row, a
 * "needs attention" queue, a recent-activity feed, and quick nav to the
 * other primary destinations. Replaces the former placeholder welcome card.
 */
import { computed, onUnmounted } from 'vue'
import { useReposStore } from '@modules/repos/store'
import { useRunsStore } from '@modules/runs/store'
import {
  attentionRuns,
  formatDateTime,
  recentRuns,
  runStats,
  runTitle,
  RunStatusChip,
} from '@modules/runs'
import { Alert, Button, Heading, Icon, Skeleton, Text } from '@shared/ui/design-system'

const runsStore = useRunsStore()
const reposStore = useReposStore()

// Mirrors RunsListSection's/the run detail page's onUnmounted pause — the
// store's polling `watch` only reacts to `hasActiveRun`/document visibility,
// it has no idea this page navigated away. Without this, navigating off the
// dashboard while a run is active leaves the 2.5s poll running forever.
onUnmounted(() => {
  runsStore.pausePolling()
})

const stats = computed(() => runStats(runsStore.runs))
const attention = computed(() => attentionRuns(runsStore.runs))
const recent = computed(() => recentRuns(runsStore.runs, 6))

interface StatTile {
  key: string
  label: string
  value: number
  icon: string
  emphasize: boolean
}

const statTiles = computed<StatTile[]>(() => [
  { key: 'repos', label: 'Repositories', value: reposStore.repos.length, icon: 'git-branch', emphasize: false },
  { key: 'active', label: 'Active runs', value: stats.value.active, icon: 'activity', emphasize: false },
  {
    key: 'attention',
    label: 'Needs attention',
    value: stats.value.attention,
    icon: 'triangle-alert',
    emphasize: stats.value.attention > 0,
  },
  { key: 'total', label: 'Total runs', value: stats.value.total, icon: 'layers', emphasize: false },
])

interface QuickNavItem {
  key: string
  to: string
  icon: string
  label: string
  description: string
}

const quickNavItems: QuickNavItem[] = [
  { key: 'reviews', to: '/reviews', icon: 'git-pull-request', label: 'Reviews', description: 'AI review runs across every repo.' },
  { key: 'runs', to: '/runs', icon: 'activity', label: 'Runs', description: 'Routine runs — releases and approve & tag.' },
  { key: 'settings', to: '/settings', icon: 'settings', label: 'Settings', description: 'Repos, providers, accounts, and notifications.' },
]
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <Heading :level="1" size="2xl">Dashboard</Heading>
      <Text muted>An overview of your repositories and routine runs.</Text>
    </div>

    <div v-if="runsStore.runsState.status === 'pending'" class="flex flex-col gap-6" data-testid="dashboard-loading">
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div v-for="n in 4" :key="n" class="flex flex-col gap-2 rounded-lg border border-line-subtle bg-bg-panel p-4">
          <Skeleton class="h-4 w-20" />
          <Skeleton class="h-7 w-12" />
        </div>
      </div>
      <div class="flex flex-col gap-2">
        <div
          v-for="n in 3"
          :key="n"
          class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
        >
          <div class="flex flex-col gap-2">
            <Skeleton class="h-4 w-48" />
            <Skeleton class="h-3 w-32" />
          </div>
        </div>
      </div>
    </div>

    <Alert v-else-if="runsStore.runsState.status === 'error'" status="danger">
      <p>{{ runsStore.error?.message ?? 'Failed to load runs' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="runsStore.refetch()">Retry</Button>
    </Alert>

    <template v-else>
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div
          v-for="tile in statTiles"
          :key="tile.key"
          class="flex flex-col gap-2 rounded-lg border p-4"
          :class="tile.emphasize ? 'border-warning-solid/30 bg-warning-bg' : 'border-line-subtle bg-bg-panel'"
          data-testid="dashboard-stat-tile"
        >
          <div class="flex items-center gap-2">
            <Icon :name="tile.icon" size="sm" :class="tile.emphasize ? 'text-warning-text' : 'text-text-muted'" />
            <Text muted size="sm">{{ tile.label }}</Text>
          </div>
          <Text as="span" size="2xl" class="font-semibold tracking-tight" :class="tile.emphasize ? 'text-warning-text' : 'text-text'">
            {{ tile.value }}
          </Text>
        </div>
      </div>

      <section class="flex flex-col gap-3">
        <Heading :level="2" size="lg">Needs attention</Heading>

        <div
          v-if="attention.length === 0"
          class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-6 text-center"
        >
          <Icon name="circle-check" size="lg" class="text-text-muted" />
          <Text muted size="sm">Nothing needs attention.</Text>
        </div>

        <ul v-else class="flex flex-col gap-2">
          <li
            v-for="run in attention"
            :key="run.id"
            class="rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
            data-testid="attention-run-row"
          >
            <RouterLink
              :to="`/runs/${run.id}`"
              :aria-label="`View run ${runTitle(run)}`"
              class="flex min-w-0 flex-col gap-1 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <Text class="truncate font-semibold">{{ runTitle(run) }}</Text>
              <Text muted size="sm" class="truncate">{{ run.repoName ?? 'Unknown repo' }}</Text>
              <div class="flex flex-wrap items-center gap-2">
                <RunStatusChip :status="run.status" />
                <Text muted size="xs">{{ formatDateTime(run.updatedAt) }}</Text>
              </div>
            </RouterLink>
          </li>
        </ul>
      </section>

      <section class="flex flex-col gap-3">
        <Heading :level="2" size="lg">Recent activity</Heading>

        <div
          v-if="recent.length === 0"
          class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-6 text-center"
        >
          <Icon name="list" size="lg" class="text-text-muted" />
          <Text muted size="sm">No runs yet.</Text>
          <RouterLink to="/runs" class="text-sm text-accent-text underline-offset-2 hover:underline">
            View runs
          </RouterLink>
        </div>

        <ul v-else class="flex flex-col gap-2">
          <li
            v-for="run in recent"
            :key="run.id"
            class="rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
            data-testid="recent-run-row"
          >
            <RouterLink
              :to="`/runs/${run.id}`"
              :aria-label="`View run ${runTitle(run)}`"
              class="flex min-w-0 flex-col gap-1 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            >
              <Text class="truncate font-semibold">{{ runTitle(run) }}</Text>
              <Text muted size="sm" class="truncate">{{ run.repoName ?? 'Unknown repo' }}</Text>
              <div class="flex flex-wrap items-center gap-2">
                <RunStatusChip :status="run.status" />
                <Text muted size="xs">{{ formatDateTime(run.updatedAt) }}</Text>
              </div>
            </RouterLink>
          </li>
        </ul>
      </section>
    </template>

    <section class="flex flex-col gap-3">
      <Heading :level="2" size="lg">Quick links</Heading>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <RouterLink
          v-for="item in quickNavItems"
          :key="item.key"
          :to="item.to"
          class="flex flex-col gap-2 rounded-lg border border-line-subtle bg-bg-panel p-4 transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        >
          <div class="flex items-center gap-2">
            <Icon :name="item.icon" size="sm" class="text-text-muted" />
            <Text class="font-semibold">{{ item.label }}</Text>
          </div>
          <Text muted size="sm">{{ item.description }}</Text>
        </RouterLink>
      </div>
    </section>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Dashboard" } }
</route>
