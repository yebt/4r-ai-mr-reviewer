<script setup lang="ts">
/**
 * Accounts settings section — organism. Dense list rows (title — muted
 * metadata — actions). Accounts have no single standout primary action, so
 * both row actions (Edit, Delete) collapse into a Reka `DropdownMenu`
 * triggered by a trailing `⋯` icon-button — mirrors the Providers/Telegram
 * sections' overflow-menu pattern, just without an inline primary action.
 * `components/<Section>.vue` (list) + `components/<Entity>Form.vue`
 * (create/edit, opened in a Reka Dialog), backed by a feature-scoped Pinia
 * store.
 *
 * Data fetching, caching, and mutation side effects (optimistic updates,
 * rollback, success/error toasts) all live in `../store.ts` (@pinia/colada).
 * This component only renders the three query states — loading skeleton,
 * error-with-retry, list — and wires row actions to the store's mutations.
 */
import { ref } from 'vue'
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
import AccountForm from './AccountForm.vue'
import { useAccountsStore } from '../store'
import type { Account } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const store = useAccountsStore()

const dialogOpen = ref(false)
const editingAccount = ref<Account | null>(null)

function openCreateDialog() {
  editingAccount.value = null
  dialogOpen.value = true
}

function openEditDialog(account: Account) {
  editingAccount.value = account
  dialogOpen.value = true
}

function handleSaved() {
  dialogOpen.value = false
  editingAccount.value = null
}

function handleCancel() {
  dialogOpen.value = false
  editingAccount.value = null
}

const deletingId = ref<string | null>(null)
async function handleDelete(account: Account) {
  deletingId.value = account.id
  try {
    await store.removeAccount(account.id)
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
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Accounts</Text>
        <Text muted size="sm" class="truncate">Connect the GitLab accounts available to this workspace.</Text>
      </div>
      <Button class="max-md:hidden whitespace-nowrap" @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add account
      </Button>
    </div>

    <Fab label="Add account" @click="openCreateDialog" />

    <div
      v-if="store.accountsState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="accounts-loading-skeleton"
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
        <Skeleton class="h-8 w-20" />
      </div>
    </div>

    <Alert v-else-if="store.accountsState.status === 'error'" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load accounts') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="store.accounts.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="plug-zap" size="lg" class="text-text-muted" />
      <Text muted>No accounts configured yet.</Text>
      <Button variant="outline" size="sm" @click="openCreateDialog">Add your first account</Button>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="account in store.accounts"
        :key="account.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <Text class="truncate font-medium">{{ account.name }}</Text>
          <Text muted size="sm" class="truncate">{{ account.baseUrl }}</Text>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${account.name}`">
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
                  @select="openEditDialog(account)"
                >
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Delete "${account.name}"?`'
                  description="This removes the account and its stored token. This cannot be undone."
                  confirm-label="Delete"
                  danger
                  :pending="deletingId === account.id"
                  @confirm="handleDelete(account)"
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

    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay class="overlay z-20" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-30 max-h-[85svh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
        >
          <DialogTitle class="text-md font-semibold">
            {{ editingAccount ? 'Edit account' : 'Add account' }}
          </DialogTitle>
          <DialogDescription class="mt-1 text-sm text-text-muted">
            {{
              editingAccount
                ? "Update this account's connection settings."
                : 'Connect a new GitLab account to this workspace.'
            }}
          </DialogDescription>
          <AccountForm
            :key="editingAccount?.id ?? 'create'"
            class="mt-4"
            :account="editingAccount"
            @saved="handleSaved"
            @cancel="handleCancel"
          />
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>
