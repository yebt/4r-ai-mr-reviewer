<script setup lang="ts">
/**
 * Flow picker (`/flow`) — the entry point into the per-repo Flow workspace
 * (`./[repoId].vue`). A searchable list of every connected repo
 * (`useReposStore`), each row linking to `/flow/:repoId` and wearing an
 * "attention" count badge — reviews `awaiting_approval`/`error` plus runs
 * `awaiting_confirmation`/`blocked`, computed by
 * `@modules/flow#attentionCountsByRepo` over the runs/reviews stores'
 * already-loaded global recent lists (see that helper's doc for the
 * resulting "recent items only" limitation). Attention-heavy repos float to
 * the top, mirroring the old app's picker
 * (`packages/spa/src/pages/flow/index.vue`). The "Flow" nav entry
 * (`src/core/nav.ts`) already routes here.
 */
import { computed, ref } from 'vue'
import { useFilter } from 'reka-ui'
import { Badge, Heading, Icon, Skeleton, Text } from '@shared/ui/design-system'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useRunsStore } from '@modules/runs/store'
import { attentionCountsByRepo } from '@modules/flow'

const reposStore = useReposStore()
const reviewsStore = useReviewsStore()
const runsStore = useRunsStore()

const attentionByRepo = computed(() => attentionCountsByRepo(reviewsStore.reviews, runsStore.runs))

// Locale-aware, case-insensitive filter — same primitive nav.ts's command
// palette filter uses (`useFilter` from reka-ui), so search behaves
// consistently across the app.
const query = ref('')
const { contains } = useFilter({ sensitivity: 'base' })

const filteredRepos = computed(() => {
  const term = query.value.trim()
  const base = term ? reposStore.repos.filter((repo) => contains(repo.name, term)) : reposStore.repos
  // Repos that need attention float to the top; stable sort keeps every
  // other row in its existing (server) order.
  return [...base].sort(
    (a, b) => (attentionByRepo.value.get(b.id) ?? 0) - (attentionByRepo.value.get(a.id) ?? 0),
  )
})
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <Heading :level="1" size="2xl">Flow</Heading>
      <Text muted>Pick a repository to review its merge requests, releases, and runs.</Text>
    </div>

    <div v-if="reposStore.reposState.status === 'pending'" class="flex flex-col gap-2" data-testid="flow-repos-loading-skeleton">
      <div
        v-for="n in 4"
        :key="n"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <Skeleton class="h-4 w-48" />
      </div>
    </div>

    <div
      v-else-if="reposStore.repos.length === 0"
      class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="git-branch" size="lg" class="text-text-muted" />
      <Text muted>No repositories tracked yet.</Text>
      <RouterLink
        to="/settings/repos"
        class="inline-flex h-8 items-center gap-2 rounded-md bg-accent px-3 text-sm font-medium text-text-on-accent transition-colors hover:bg-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon name="plus" size="sm" />
        Track a repository
      </RouterLink>
    </div>

    <template v-else>
      <label class="relative flex items-center">
        <Icon name="search" size="sm" class="pointer-events-none absolute left-2.5 text-text-muted" />
        <input
          v-model="query"
          type="text"
          class="h-9 w-full rounded-md border border-line bg-bg-panel pl-8 pr-3 text-sm text-text outline-none placeholder:text-text-placeholder focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          placeholder="Search repositories…"
          autocomplete="off"
          aria-label="Search repositories"
        />
      </label>

      <ul class="flex flex-col gap-2">
        <li v-for="repo in filteredRepos" :key="repo.id">
          <RouterLink
            :to="`/flow/${repo.id}`"
            class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          >
            <Icon name="git-branch" size="sm" class="shrink-0 text-text-muted" />
            <div class="flex min-w-0 flex-1 flex-col gap-0.5">
              <Text class="truncate font-medium">{{ repo.name }}</Text>
              <Text muted size="xs" class="truncate font-mono">{{ repo.url }}</Text>
            </div>
            <Badge
              v-if="(attentionByRepo.get(repo.id) ?? 0) > 0"
              status="warning"
              class="shrink-0"
              :aria-label="`${attentionByRepo.get(repo.id)} items need attention`"
            >
              {{ attentionByRepo.get(repo.id) }}
            </Badge>
          </RouterLink>
        </li>
        <li
          v-if="filteredRepos.length === 0"
          class="rounded-lg border border-dashed border-line p-6 text-center"
        >
          <Text muted size="sm">No repositories match "{{ query }}".</Text>
        </li>
      </ul>
    </template>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Flow" } }
</route>
