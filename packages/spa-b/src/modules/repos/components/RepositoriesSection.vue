<script setup lang="ts">
/**
 * Repos settings section — organism, mirrors `ProvidersSection` (dense list
 * rows, secondary/destructive row actions collapsed into a Reka
 * `DropdownMenu`). `components/<Section>.vue` (list) + `components/
 * <Entity>Form.vue` (create/reassign, opened in a Reka Dialog) +
 * `components/WebhookDialog.vue` (a second, standalone dialog for the
 * webhook config), backed by a feature-scoped Pinia store.
 *
 * `useAccountsStore`/`useProvidersStore` are read-only dependencies here,
 * used only to resolve `accountId`/`providerId` into display labels for the
 * metadata line — never mutated from this module.
 */
import { computed, ref } from 'vue'
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
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Button, ConfirmDialog, Fab, Icon, Skeleton, Text } from '@shared/ui/design-system'
// Deep-imported (not through each module's barrel, which only re-exports
// its Section component + types, not the store) — read-only reuse, per the
// task boundary: this module never edits accounts/providers.
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'
import RepoForm from './RepoForm.vue'
import WebhookDialog from './WebhookDialog.vue'
import { useReposStore } from '../store'
import type { Repo } from '../types'

const store = useReposStore()
const accountsStore = useAccountsStore()
const providersStore = useProvidersStore()

function accountLabel(accountId: string): string {
  return accountsStore.accounts.find((account) => account.id === accountId)?.name ?? 'Unknown account'
}

/**
 * `providerId`/`model` of `''` mean "use default" — `model` can still
 * override just the model while keeping the default provider, so it's
 * resolved independently of `providerId`.
 */
function providerLabel(repo: Repo): string {
  if (!repo.providerId) return 'Default provider'
  const provider = providersStore.providers.find((candidate) => candidate.id === repo.providerId)
  const providerName = provider?.name ?? 'Unknown provider'
  const model = repo.model || provider?.model
  return model ? `${providerName} / ${model}` : providerName
}

function subtitle(repo: Repo): string {
  return `${accountLabel(repo.accountId)} · ${providerLabel(repo)}`
}

const formDialogOpen = ref(false)
const editingRepo = ref<Repo | null>(null)

function openCreateDialog() {
  editingRepo.value = null
  formDialogOpen.value = true
}

function openEditDialog(repo: Repo) {
  editingRepo.value = repo
  formDialogOpen.value = true
}

function handleSaved() {
  formDialogOpen.value = false
  editingRepo.value = null
}

function handleFormCancel() {
  formDialogOpen.value = false
  editingRepo.value = null
}

const webhookDialogOpen = ref(false)
const webhookRepoId = ref<string | null>(null)
// Look the repo back up by id (rather than holding a snapshot) so the
// dialog always reflects the latest cache row while it's open (e.g. right
// after enabling the webhook or rotating its secret).
const webhookRepo = computed(() => store.repos.find((repo) => repo.id === webhookRepoId.value) ?? null)

function openWebhookDialog(repo: Repo) {
  webhookRepoId.value = repo.id
  webhookDialogOpen.value = true
}

function handleWebhookDialogOpenChange(open: boolean) {
  webhookDialogOpen.value = open
  if (!open) webhookRepoId.value = null
}

const deletingId = ref<string | null>(null)
async function handleDelete(repo: Repo) {
  deletingId.value = repo.id
  try {
    await store.removeRepo(repo.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex min-w-0 flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Repos</Text>
        <Text muted size="sm" class="truncate">Manage the repositories connected to this workspace.</Text>
      </div>
      <Button class="max-md:hidden whitespace-nowrap" @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add repository
      </Button>
    </div>

    <Fab label="Add repository" @click="openCreateDialog" />

    <div
      v-if="store.reposState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="repos-loading-skeleton"
    >
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-8 w-8" />
      </div>
    </div>

    <Alert v-else-if="store.reposState.status === 'error'" status="danger">
      <p>{{ store.error?.message ?? 'Failed to load repositories' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="store.repos.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="git-branch" size="lg" class="text-text-muted" />
      <Text muted>No repositories connected yet.</Text>
      <Button variant="outline" size="sm" @click="openCreateDialog">Add your first repository</Button>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="repo in store.repos"
        :key="repo.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <Text class="truncate font-medium">{{ repo.name }}</Text>
          <Text muted size="sm" class="truncate">{{ subtitle(repo) }}</Text>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${repo.name}`">
                <Icon name="ellipsis" size="sm" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuPortal>
              <DropdownMenuContent
                align="end"
                :side-offset="4"
                class="z-30 min-w-40 rounded-md border border-line bg-bg-panel-raised p-1 shadow-token-lg"
              >
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="openEditDialog(repo)"
                >
                  Reassign
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="openWebhookDialog(repo)"
                >
                  Webhook
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Delete "${repo.name}"?`'
                  description="Removes the repo and its reviews."
                  confirm-label="Delete"
                  danger
                  :pending="deletingId === repo.id"
                  @confirm="handleDelete(repo)"
                >
                  <template #trigger>
                    <DropdownMenuItem
                      class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[highlighted]:bg-danger-bg"
                      @select.prevent
                    >
                      Delete
                    </DropdownMenuItem>
                  </template>
                </ConfirmDialog>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
        </div>
      </li>
    </ul>

    <DialogRoot v-model:open="formDialogOpen">
      <DialogPortal>
        <DialogOverlay class="overlay z-20" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
        >
          <DialogTitle class="text-md font-semibold">
            {{ editingRepo ? 'Reassign repository' : 'Add repository' }}
          </DialogTitle>
          <DialogDescription class="mt-1 text-sm text-text-muted">
            {{
              editingRepo
                ? "Update this repository's account, provider, model, or default voice profile."
                : 'Connect a new repository to this workspace.'
            }}
          </DialogDescription>
          <RepoForm
            :key="editingRepo?.id ?? 'create'"
            class="mt-4"
            :repo="editingRepo"
            @saved="handleSaved"
            @cancel="handleFormCancel"
          />
        </DialogContent>
      </DialogPortal>
    </DialogRoot>

    <WebhookDialog :open="webhookDialogOpen" :repo="webhookRepo" @update:open="handleWebhookDialogOpenChange" />
  </section>
</template>
