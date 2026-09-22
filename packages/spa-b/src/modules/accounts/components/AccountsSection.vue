<script setup lang="ts">
/**
 * Accounts settings section — organism. List + row actions (Edit,
 * Delete-with-confirm) and an Add-account Dialog. Mirrors the Providers
 * module's reference pattern: `components/<Section>.vue` (list) +
 * `components/<Entity>Form.vue` (create/edit, opened in a Reka Dialog),
 * backed by a feature-scoped Pinia store.
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
} from 'reka-ui'
import { Button, ConfirmDialog, Icon, Skeleton, Text } from '@shared/ui/design-system'
import AccountForm from './AccountForm.vue'
import { useAccountsStore } from '../store'
import type { Account } from '../types'

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
      <div class="flex flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Accounts</Text>
        <Text muted size="sm">Connect the GitLab accounts available to this workspace.</Text>
      </div>
      <Button @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add account
      </Button>
    </div>

    <div
      v-if="store.accountsState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="accounts-loading-skeleton"
    >
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-8 w-20" />
      </div>
    </div>

    <div
      v-else-if="store.accountsState.status === 'error'"
      role="alert"
      class="flex flex-col items-start gap-2 rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      <p>{{ store.error?.message ?? 'Failed to load accounts' }}</p>
      <Button variant="outline" size="sm" @click="store.refetch()">Retry</Button>
    </div>

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
        class="flex flex-col gap-3 rounded-lg border border-line bg-bg-panel p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex flex-col gap-1">
          <Text class="font-medium">{{ account.name }}</Text>
          <Text muted size="sm">{{ account.baseUrl }}</Text>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button variant="ghost" size="sm" @click="openEditDialog(account)">Edit</Button>

          <ConfirmDialog
            :title='`Delete "${account.name}"?`'
            description="This removes the account and its stored token. This cannot be undone."
            confirm-label="Delete"
            danger
            :pending="deletingId === account.id"
            @confirm="handleDelete(account)"
          >
            <template #trigger>
              <Button variant="ghost" size="sm">Delete</Button>
            </template>
          </ConfirmDialog>
        </div>
      </li>
    </ul>

    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay class="overlay z-20" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
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
