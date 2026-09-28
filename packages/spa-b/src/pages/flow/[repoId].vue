<script setup lang="ts">
/**
 * Flow workspace (`/flow/:repoId`) — the repo lives in the URL so refresh,
 * back and deep links all land on the same repo (mirrors the old app's
 * `packages/spa/src/pages/flow/[repoId].vue`). Header: repo name + link to
 * its GitLab URL + a repo switcher (the design-system `Select` atom,
 * navigating to the same tab on another repo). Below it, a 3-tab strip
 * (MRs · Reviews · Actions) built on Reka's `Tabs` primitives, synced to
 * `?tab=` (default `mrs`) so refresh/back/deep-link keep the active tab.
 *
 * `reposStore.repos` is the same global list `./index.vue`'s picker reads —
 * no extra fetch here. `ready` gates the "repository not found" state on
 * that list having settled at least once, so a genuinely unknown repo id
 * shows "not found" instead of flashing it while repos are still loading.
 *
 * No action buttons/dialogs yet (Review/Release/New MR land in slices 2-4).
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TabsContent, TabsList, TabsRoot, TabsTrigger } from 'reka-ui'
import { Heading, Icon, Select, Skeleton, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useReposStore } from '@modules/repos/store'
import { MergeRequestListSection, RepoActionsSection, RepoReviewsSection } from '@modules/flow'

const route = useRoute('/flow/[repoId]')
const router = useRouter()
const reposStore = useReposStore()

const repoId = computed(() => String(route.params.repoId))
const repo = computed(() => reposStore.repos.find((r) => r.id === repoId.value) ?? null)
const ready = computed(() => !reposStore.isLoading)

const switcherOptions = computed<SelectItemOption[]>(() =>
  reposStore.repos.map((r) => ({ label: r.name, value: r.id })),
)

function switchRepo(id: string | undefined) {
  if (!id || id === repoId.value) return
  router.push({ path: `/flow/${id}`, query: route.query })
}

type FlowTab = 'mrs' | 'reviews' | 'actions'
const FLOW_TABS: { id: FlowTab; label: string; icon: string }[] = [
  { id: 'mrs', label: 'MRs', icon: 'git-pull-request' },
  { id: 'reviews', label: 'Reviews', icon: 'list' },
  { id: 'actions', label: 'Actions', icon: 'activity' },
]
function isFlowTab(value: unknown): value is FlowTab {
  return value === 'mrs' || value === 'reviews' || value === 'actions'
}
// A writable computed, so `TabsRoot`'s v-model both reads the URL and
// writes back to it — no separate watcher needed (mirrors the old app's
// `?tab=` sync, just expressed as a getter/setter instead of a ref+watch).
const tab = computed<FlowTab>({
  get: () => (isFlowTab(route.query.tab) ? route.query.tab : 'mrs'),
  set: (value) => {
    router.replace({ query: { ...route.query, tab: value } })
  },
})
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-if="!ready" class="flex flex-col gap-4">
      <Skeleton class="h-7 w-64" />
      <Skeleton class="h-4 w-96" />
    </div>

    <div
      v-else-if="!repo"
      class="flex flex-col items-center gap-3 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="git-branch" size="lg" class="text-text-muted" />
      <Text muted>This repository is not tracked, or the link is out of date.</Text>
      <RouterLink
        to="/flow"
        class="inline-flex items-center gap-1.5 text-sm text-text-muted transition-colors hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon name="chevron-down" size="sm" class="rotate-90" />
        Back to Flow
      </RouterLink>
    </div>

    <template v-else>
      <div class="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
        <div class="min-w-0">
          <Heading :level="1" size="xl" class="truncate">{{ repo.name }}</Heading>
          <a
            v-if="repo.url"
            :href="repo.url"
            target="_blank"
            rel="noopener"
            class="block truncate font-mono text-xs text-text-muted hover:text-text hover:underline"
          >
            {{ repo.url }}
          </a>
        </div>
        <div class="w-full shrink-0 sm:w-64">
          <Select
            :model-value="repoId"
            :items="switcherOptions"
            aria-label="Switch repository"
            @update:model-value="switchRepo"
          />
        </div>
      </div>

      <TabsRoot v-model="tab" class="flex flex-col gap-4">
        <TabsList class="flex gap-1 overflow-x-auto border-b border-line-subtle" aria-label="Repository workspace">
          <TabsTrigger
            v-for="t in FLOW_TABS"
            :key="t.id"
            :value="t.id"
            class="-mb-px inline-flex shrink-0 items-center gap-1.5 border-b-2 border-transparent px-3 py-2.5 text-sm font-medium text-text-muted outline-none transition-colors hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring data-[state=active]:border-accent-solid data-[state=active]:text-text"
          >
            <Icon :name="t.icon" size="sm" />
            {{ t.label }}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="mrs">
          <MergeRequestListSection :repo-id="repoId" />
        </TabsContent>
        <TabsContent value="reviews">
          <RepoReviewsSection :repo-id="repoId" />
        </TabsContent>
        <TabsContent value="actions">
          <RepoActionsSection :repo-id="repoId" />
        </TabsContent>
      </TabsRoot>
    </template>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Flow" } }
</route>
