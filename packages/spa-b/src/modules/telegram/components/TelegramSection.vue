<script setup lang="ts">
/**
 * Telegram settings section — organism. List + row actions (Test, Set
 * default, Edit, Delete-with-confirm) and an Add-target Dialog. Mirrors
 * `modules/providers/components/ProvidersSection.vue`: `components/<Section>.vue`
 * (list) + `components/<Entity>Form.vue` (create/edit, opened in a Reka
 * Dialog), backed by a feature-scoped Pinia store.
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
import { Badge, Button, ConfirmDialog, Icon, Skeleton, Text } from '@shared/ui/design-system'
import TelegramForm from './TelegramForm.vue'
import { useTelegramStore } from '../store'
import type { TelegramTarget, TestTelegramTargetResult } from '../types'

const store = useTelegramStore()

function subtitle(target: TelegramTarget): string {
  return target.threadId ? `${target.chatId} · thread ${target.threadId}` : target.chatId
}

const dialogOpen = ref(false)
const editingTarget = ref<TelegramTarget | null>(null)

function openCreateDialog() {
  editingTarget.value = null
  dialogOpen.value = true
}

function openEditDialog(target: TelegramTarget) {
  editingTarget.value = target
  dialogOpen.value = true
}

function handleSaved() {
  dialogOpen.value = false
  editingTarget.value = null
}

function handleCancel() {
  dialogOpen.value = false
  editingTarget.value = null
}

const settingDefaultId = ref<string | null>(null)
async function handleSetDefault(target: TelegramTarget) {
  if (target.isDefault) return
  settingDefaultId.value = target.id
  try {
    // Rollback and the error toast are handled by the store's mutation
    // hooks; this catch only prevents an unhandled rejection here.
    await store.setDefaultTarget(target.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    settingDefaultId.value = null
  }
}

const deletingId = ref<string | null>(null)
async function handleDelete(target: TelegramTarget) {
  deletingId.value = target.id
  try {
    await store.removeTarget(target.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    deletingId.value = null
  }
}

const testingId = ref<string | null>(null)
const testResults = ref<Record<string, TestTelegramTargetResult>>({})
async function handleTest(target: TelegramTarget) {
  testingId.value = target.id
  try {
    const result = await store.testTarget(target.id)
    testResults.value = { ...testResults.value, [target.id]: result }
  } catch (err) {
    testResults.value = {
      ...testResults.value,
      [target.id]: { ok: false, error: err instanceof Error ? err.message : 'Test failed' },
    }
  } finally {
    testingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Telegram</Text>
        <Text muted size="sm">Configure the Telegram chats notifications are sent to.</Text>
      </div>
      <Button @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add target
      </Button>
    </div>

    <div
      v-if="store.targetsState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="telegram-loading-skeleton"
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
      v-else-if="store.targetsState.status === 'error'"
      role="alert"
      class="flex flex-col items-start gap-2 rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      <p>{{ store.error?.message ?? 'Failed to load Telegram targets' }}</p>
      <Button variant="outline" size="sm" @click="store.refetch()">Retry</Button>
    </div>

    <div
      v-else-if="store.targets.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="send" size="lg" class="text-text-muted" />
      <Text muted>No Telegram targets configured yet.</Text>
      <Button variant="outline" size="sm" @click="openCreateDialog">Add your first target</Button>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="target in store.targets"
        :key="target.id"
        class="flex flex-col gap-3 rounded-lg border border-line bg-bg-panel p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="font-medium">{{ target.name }}</Text>
            <Badge v-if="target.isDefault" status="info">Default</Badge>
            <Badge v-if="target.isBot">Bot</Badge>
          </div>
          <Text muted size="sm">{{ subtitle(target) }}</Text>
          <Text
            v-if="testResults[target.id]"
            size="xs"
            :class="testResults[target.id]!.ok ? 'text-success-text' : 'text-danger-text'"
          >
            {{
              testResults[target.id]!.ok
                ? 'Test message sent'
                : (testResults[target.id]!.error ?? 'Test failed')
            }}
          </Text>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" :loading="testingId === target.id" @click="handleTest(target)">
            Test
          </Button>
          <Button
            v-if="!target.isDefault"
            variant="ghost"
            size="sm"
            :loading="settingDefaultId === target.id"
            @click="handleSetDefault(target)"
          >
            Set default
          </Button>
          <Button variant="ghost" size="sm" @click="openEditDialog(target)">Edit</Button>

          <ConfirmDialog
            :title='`Delete "${target.name}"?`'
            description="This removes the Telegram target and its stored bot token. This cannot be undone."
            confirm-label="Delete"
            danger
            :pending="deletingId === target.id"
            @confirm="handleDelete(target)"
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
            {{ editingTarget ? 'Edit Telegram target' : 'Add Telegram target' }}
          </DialogTitle>
          <DialogDescription class="mt-1 text-sm text-text-muted">
            {{
              editingTarget
                ? "Update this Telegram target's settings."
                : 'Connect a new Telegram chat to receive notifications.'
            }}
          </DialogDescription>
          <TelegramForm
            :key="editingTarget?.id ?? 'create'"
            class="mt-4"
            :target="editingTarget"
            @saved="handleSaved"
            @cancel="handleCancel"
          />
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>
