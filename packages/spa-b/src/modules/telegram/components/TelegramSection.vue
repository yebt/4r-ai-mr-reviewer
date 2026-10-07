<script setup lang="ts">
/**
 * Telegram settings section — organism. Dense list rows (title — muted
 * metadata — actions) with the secondary/destructive row actions (Set
 * default, Edit, Delete) collapsed into a Reka `DropdownMenu` triggered by a
 * trailing `⋯` icon-button, keeping only the single standout primary action
 * (Test) inline. Mirrors `modules/providers/components/ProvidersSection.vue`:
 * `components/<Section>.vue` (list) + `components/<Entity>Form.vue`
 * (create/edit, opened in a Reka Dialog), backed by a feature-scoped Pinia
 * store.
 *
 * The default target is always sorted first (`sortDefaultFirst`, stable —
 * only the default row moves) and briefly highlighted when it changes, so
 * "Set default" reads as an obvious, single-item reorder.
 *
 * Data fetching, caching, and mutation side effects (optimistic updates,
 * rollback, success/error toasts) all live in `../store.ts` (@pinia/colada).
 * This component only renders the three query states — loading skeleton,
 * error-with-retry, list — and wires row actions to the store's mutations.
 */
import { computed, onBeforeUnmount, ref } from 'vue'
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
import { Alert, Badge, Button, ConfirmDialog, Fab, Icon, Skeleton, Text } from '@shared/ui/design-system'
import TelegramForm from './TelegramForm.vue'
import { sortDefaultFirst, useTelegramStore } from '../store'
import type { TelegramTarget, TestTelegramTargetResult } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const store = useTelegramStore()

const sortedTargets = computed(() => sortDefaultFirst(store.targets))

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

// Briefly highlighted after a row becomes the default, so the sort-to-top
// reads as an obviously reactive move rather than a silent reshuffle.
const highlightId = ref<string | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | undefined
onBeforeUnmount(() => clearTimeout(highlightTimer))

const settingDefaultId = ref<string | null>(null)
async function handleSetDefault(target: TelegramTarget) {
  if (target.isDefault) return
  settingDefaultId.value = target.id
  try {
    // Rollback and the error toast are handled by the store's mutation
    // hooks; this catch only prevents an unhandled rejection here.
    await store.setDefaultTarget(target.id)
    highlightId.value = target.id
    clearTimeout(highlightTimer)
    highlightTimer = setTimeout(() => {
      highlightId.value = null
    }, 1200)
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
    // A resolved request means the test message was delivered: the backend
    // returns 200 {status:'sent'} on success and throws (404/502) on failure.
    await store.testTarget(target.id)
    testResults.value = { ...testResults.value, [target.id]: { ok: true } }
  } catch (err) {
    testResults.value = {
      ...testResults.value,
      [target.id]: { ok: false, error: resolveErrorMessage(err, 'Test failed') },
    }
  } finally {
    testingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex min-w-0 flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Telegram</Text>
        <Text muted size="sm" class="truncate">Configure the Telegram chats notifications are sent to.</Text>
      </div>
      <Button class="max-md:hidden whitespace-nowrap" @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add target
      </Button>
    </div>

    <Fab label="Add Telegram target" @click="openCreateDialog" />

    <div
      v-if="store.targetsState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="telegram-loading-skeleton"
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

    <Alert v-else-if="store.targetsState.status === 'error'" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load Telegram targets') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

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
        v-for="target in sortedTargets"
        :key="target.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors duration-700"
        :class="highlightId === target.id ? 'bg-accent-subtle-bg ring-1 ring-accent' : ''"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">{{ target.name }}</Text>
            <Badge v-if="target.isDefault" status="info">Default</Badge>
            <Badge v-if="target.isBot">Bot</Badge>
          </div>
          <Text muted size="sm" class="truncate">{{ subtitle(target) }}</Text>
          <Alert
            v-if="testResults[target.id]"
            :status="testResults[target.id]!.ok ? 'success' : 'danger'"
            class="mt-1"
          >
            {{
              testResults[target.id]!.ok
                ? 'Test message sent'
                : (testResults[target.id]!.error ?? 'Test failed')
            }}
          </Alert>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <Button variant="outline" size="sm" :loading="testingId === target.id" @click="handleTest(target)">
            Test
          </Button>

          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${target.name}`">
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
                  v-if="!target.isDefault"
                  :disabled="settingDefaultId === target.id"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                  @select="handleSetDefault(target)"
                >
                  Set default
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="openEditDialog(target)"
                >
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Delete "${target.name}"?`'
                  description="This removes the Telegram target and its stored bot token. This cannot be undone."
                  confirm-label="Delete"
                  danger
                  :pending="deletingId === target.id"
                  @confirm="handleDelete(target)"
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
