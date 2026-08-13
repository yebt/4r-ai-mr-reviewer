<script setup lang="ts">
definePage({ meta: { title: 'Flow' } })
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useRoutinesStore } from '@modules/routines/store'

// The Flow entry point: a searchable repo picker (no auto-redirect to a last
// repo — the workspace lives at /flow/:repoId, driven by the URL). Repos that
// need attention — held/failed reviews or a paused release — float to the top
// and wear a count badge, so the picker doubles as a triage list.
const router = useRouter()
const repos = useReposStore()
const reviews = useReviewsStore()
const routines = useRoutinesStore()

const loading = ref(true)

const PAUSED = new Set(['awaiting_confirmation', 'blocked'])
function repoAttention(id: string): number {
  const revs = reviews
    .reviewsFor(id)
    .filter((r) => r.status === 'awaiting_approval' || r.status === 'error').length
  const runs = routines.recentRuns.filter((r) => r.repoId === id && PAUSED.has(r.status)).length
  return revs + runs
}

const query = ref('')
const filteredRepos = computed(() => {
  const q = query.value.trim().toLowerCase()
  const list = q ? repos.items.filter((r) => r.name.toLowerCase().includes(q)) : repos.items
  // Repos that need attention float up.
  return [...list].sort((a, b) => repoAttention(b.id) - repoAttention(a.id))
})

function openRepo(id: string) {
  router.push(`/flow/${id}`)
}
function onEnter() {
  const first = filteredRepos.value[0]
  if (first) openRepo(first.id)
}

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  // Attention badges need each repo's reviews and the recent routine runs.
  await Promise.all([
    ...repos.items.map((r) => reviews.fetchReviews(r.id)),
    routines.listRecent(50),
  ])
  loading.value = false
})
</script>

<template>
  <div>
    <PageHeader title="Flow" label="Workspace" />

    <EmptyState
      v-if="repos.items.length === 0 && !loading"
      icon="i-lucide-folder-git-2"
      title="No repositories tracked"
      hint="Track a GitLab repository to start reviewing its merge requests and cutting releases."
    >
      <template #action>
        <RouterLink to="/repos" class="btn-accent text-xs">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Track a repository
        </RouterLink>
      </template>
    </EmptyState>

    <div v-else class="border-line bg-surface flex flex-col overflow-hidden border">
      <div class="border-line/60 flex items-center gap-2 border-b px-3">
        <span class="i-lucide-search text-muted shrink-0 text-sm" aria-hidden="true" />
        <input
          v-model="query"
          type="text"
          class="text-ink placeholder:text-muted/60 w-full bg-transparent py-2.5 text-sm outline-none"
          placeholder="Search repositories…"
          autocomplete="off"
          aria-label="Search repositories"
          @keydown.enter.prevent="onEnter"
        />
      </div>
      <ul class="min-h-0 flex-1 overflow-y-auto py-1">
        <li v-for="r in filteredRepos" :key="r.id">
          <button
            type="button"
            class="text-muted hover:text-ink flex w-full cursor-pointer items-center gap-2 border-l-2 border-transparent px-3 py-2.5 text-left text-sm"
            @click="openRepo(r.id)"
          >
            <span class="i-lucide-folder-git-2 shrink-0 text-sm opacity-80" aria-hidden="true" />
            <span class="min-w-0 flex-1 truncate">{{ r.name }}</span>
            <span
              v-if="repoAttention(r.id) > 0"
              class="bg-warn/15 text-warn inline-flex min-w-4 items-center justify-center px-1 font-mono text-[0.6rem]"
              :aria-label="`${repoAttention(r.id)} need attention`"
            >
              {{ repoAttention(r.id) }}
            </span>
            <span class="i-lucide-chevron-right shrink-0 text-sm" aria-hidden="true" />
          </button>
        </li>
        <li v-if="filteredRepos.length === 0 && !loading" class="text-muted px-3 py-3 text-xs">
          No repositories match.
        </li>
      </ul>
    </div>
  </div>
</template>
