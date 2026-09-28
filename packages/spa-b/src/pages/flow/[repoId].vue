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
 * Review (slice 2), Release/Release-to-main (slice 3) and New MR (slice 4)
 * dialogs are all wired. The header's "Release to main" and "New MR" buttons
 * both live in the MRs tab's `#header-actions` slot, with their dialogs
 * (`ReleaseDialog` `flow="main"`, `NewMergeRequestDialog`) here rather than
 * inside `MergeRequestListSection`, since neither needs an existing MR and
 * both must work with zero open MRs.
 *
 * Slice 5: a "Repository settings" gear next to the switcher opens a Reka
 * `DropdownMenu` with three items, reusing `modules/repos`' existing
 * pieces rather than duplicating their logic — "Provider & model…" opens
 * `RepoForm` (reassign mode) inside a Dialog, exactly like
 * `RepositoriesSection` does on Settings → Repos; "Webhook…" and "Check
 * permissions…" open `WebhookDialog`/`PreflightDialog`, which both own
 * their own `DialogRoot` already. All three are imported by file path
 * (not the module barrel, which only re-exports `RepositoriesSection` +
 * types — see that module's own `index.ts`).
 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuTrigger,
  TabsContent,
  TabsList,
  TabsRoot,
  TabsTrigger,
} from 'reka-ui'
import { Button, Heading, Icon, Select, Skeleton, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import PreflightDialog from '@modules/repos/components/PreflightDialog.vue'
import RepoForm from '@modules/repos/components/RepoForm.vue'
import WebhookDialog from '@modules/repos/components/WebhookDialog.vue'
import { useReposStore } from '@modules/repos/store'
import {
  MergeRequestListSection,
  NewMergeRequestDialog,
  ReleaseDialog,
  RepoActionsSection,
  RepoReviewsSection,
} from '@modules/flow'

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
const mainReleaseDialogOpen = ref(false)
const newMergeRequestDialogOpen = ref(false)

// Repository settings — see the top-of-file note. All three reuse
// `modules/repos`' existing form/dialog pieces for the current `repo`.
const providerFormDialogOpen = ref(false)
const webhookDialogOpen = ref(false)
const preflightDialogOpen = ref(false)

function handleProviderFormSaved() {
  providerFormDialogOpen.value = false
}

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
        <div class="flex w-full shrink-0 items-center gap-2 sm:w-auto">
          <div class="min-w-0 flex-1 sm:w-64 sm:flex-none">
            <Select
              :model-value="repoId"
              :items="switcherOptions"
              aria-label="Switch repository"
              @update:model-value="switchRepo"
            />
          </div>

          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="outline" size="sm" aria-label="Repository settings">
                <Icon name="settings" size="sm" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuPortal>
              <DropdownMenuContent
                align="end"
                :side-offset="4"
                class="z-30 min-w-48 rounded-md border border-line bg-bg-panel-raised p-1 shadow-token-lg"
              >
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="providerFormDialogOpen = true"
                >
                  Provider & model…
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="webhookDialogOpen = true"
                >
                  Webhook…
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="preflightDialogOpen = true"
                >
                  Check permissions…
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
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
          <MergeRequestListSection :repo-id="repoId">
            <template #header-actions>
              <Button type="button" variant="outline" size="sm" @click="newMergeRequestDialogOpen = true">
                New MR
              </Button>
              <Button type="button" variant="outline" size="sm" @click="mainReleaseDialogOpen = true">
                Release to main
              </Button>
            </template>
          </MergeRequestListSection>
        </TabsContent>
        <TabsContent value="reviews">
          <RepoReviewsSection :repo-id="repoId" />
        </TabsContent>
        <TabsContent value="actions">
          <RepoActionsSection :repo-id="repoId" />
        </TabsContent>
      </TabsRoot>

      <ReleaseDialog v-model:open="mainReleaseDialogOpen" flow="main" :repo="repo" />
      <NewMergeRequestDialog v-model:open="newMergeRequestDialogOpen" :repo="repo" />

      <DialogRoot v-model:open="providerFormDialogOpen">
        <DialogPortal>
          <DialogOverlay class="overlay z-20" />
          <DialogContent
            class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
          >
            <DialogTitle class="text-md font-semibold">Reassign repository</DialogTitle>
            <DialogDescription class="mt-1 text-sm text-text-muted">
              Update this repository's account, provider, model, or default voice profile.
            </DialogDescription>
            <RepoForm
              :key="repo.id"
              class="mt-4"
              :repo="repo"
              @saved="handleProviderFormSaved"
              @cancel="providerFormDialogOpen = false"
            />
          </DialogContent>
        </DialogPortal>
      </DialogRoot>

      <WebhookDialog :open="webhookDialogOpen" :repo="repo" @update:open="(value) => (webhookDialogOpen = value)" />
      <PreflightDialog
        :open="preflightDialogOpen"
        :repo="repo"
        @update:open="(value) => (preflightDialogOpen = value)"
      />
    </template>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Flow" } }
</route>
