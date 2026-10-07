<script setup lang="ts">
/**
 * Webhook config dialog for a single repo, opened from `RepositoriesSection`.
 * Unlike `RepoForm`/`ProviderForm` (form content embedded in the parent's
 * own `DialogRoot`), this owns its `DialogRoot` directly since it's a
 * standalone view rather than a create/edit form — controlled via the
 * `open`/`repo` props and `update:open`.
 *
 * The enable/require-confirmation `Switch`es save immediately on toggle
 * (no separate "Save" step) via `store.setWebhook`, matching the backend's
 * `PATCH /repos/{id}/webhook` shape (`{ enabled, requireConfirmation }`).
 * "Rotate secret" is guarded by a `ConfirmDialog` since it invalidates the
 * previously configured secret on the provider side.
 */
import { computed, ref, watch } from 'vue'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Alert, Button, ConfirmDialog, Icon, Switch, Text } from '@shared/ui/design-system'
import { useReposStore } from '../store'
import type { Repo } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const props = defineProps<{
  open: boolean
  repo: Repo | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const store = useReposStore()

const secretVisible = ref(false)
const copiedField = ref<'url' | 'secret' | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined
const savingEnabled = ref(false)
const savingConfirmation = ref(false)
const rotating = ref(false)
const actionError = ref<string | null>(null)

// Reset local, dialog-scoped UI state whenever the dialog closes or targets
// a different repo, so reopening never leaks a previous repo's "revealed"
// secret or stale error into the next one.
// Watch each value as its own source: a getter returning a fresh array would
// re-fire whenever the repo/MR list refetches (e.g. on window focus) and hands
// down a new object with the same id, wiping the dialog's in-progress state.
watch(
  [() => props.open, () => props.repo?.id],
  () => {
    secretVisible.value = false
    copiedField.value = null
    actionError.value = null
    clearTimeout(copiedTimer)
  },
)

const payloadUrl = computed(() => (props.repo ? `${location.origin}${props.repo.webhookPath}` : ''))

async function copy(field: 'url' | 'secret', value: string) {
  try {
    await navigator.clipboard.writeText(value)
    copiedField.value = field
    clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      if (copiedField.value === field) copiedField.value = null
    }, 1500)
  } catch {
    // Clipboard API unavailable/denied (e.g. insecure context) — the value
    // is still visible as selectable text, so this is a silent no-op.
  }
}

async function handleToggleEnabled(enabled: boolean) {
  if (!props.repo) return
  savingEnabled.value = true
  actionError.value = null
  try {
    await store.setWebhook(props.repo.id, {
      enabled,
      requireConfirmation: props.repo.webhookRequireConfirmation,
    })
  } catch (err) {
    actionError.value = resolveErrorMessage(err, 'Failed to update webhook')
  } finally {
    savingEnabled.value = false
  }
}

async function handleToggleConfirmation(requireConfirmation: boolean) {
  if (!props.repo) return
  savingConfirmation.value = true
  actionError.value = null
  try {
    await store.setWebhook(props.repo.id, {
      enabled: props.repo.webhookEnabled,
      requireConfirmation,
    })
  } catch (err) {
    actionError.value = resolveErrorMessage(err, 'Failed to update webhook')
  } finally {
    savingConfirmation.value = false
  }
}

async function handleRotate() {
  if (!props.repo) return
  rotating.value = true
  actionError.value = null
  try {
    await store.rotateWebhookSecret(props.repo.id)
    secretVisible.value = true
  } catch (err) {
    actionError.value = resolveErrorMessage(err, 'Failed to rotate webhook secret')
  } finally {
    rotating.value = false
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold">Webhook</DialogTitle>
        <DialogDescription class="mt-1 text-sm text-text-muted">
          {{ repo ? `Receive review triggers from "${repo.name}" automatically.` : 'Configure the repository webhook.' }}
        </DialogDescription>

        <div v-if="repo" class="mt-4 flex flex-col gap-4">
          <div class="flex items-center justify-between gap-3">
            <div class="flex flex-col gap-0.5">
              <span class="text-sm font-medium text-text">Enable webhook</span>
              <Text size="sm" muted>Trigger reviews automatically from provider push events.</Text>
            </div>
            <Switch
              :model-value="repo.webhookEnabled"
              :disabled="savingEnabled"
              aria-label="Enable webhook"
              @update:model-value="(value) => handleToggleEnabled(value === true)"
            />
          </div>

          <div class="flex items-center justify-between gap-3">
            <div class="flex flex-col gap-0.5">
              <span class="text-sm font-medium text-text">Require confirmation</span>
              <Text size="sm" muted>Wait for manual confirmation before running a review.</Text>
            </div>
            <Switch
              :model-value="repo.webhookRequireConfirmation"
              :disabled="!repo.webhookEnabled || savingConfirmation"
              aria-label="Require confirmation"
              @update:model-value="(value) => handleToggleConfirmation(value === true)"
            />
          </div>

          <template v-if="repo.webhookEnabled">
            <div class="flex flex-col gap-1.5">
              <span class="text-sm font-medium text-text">Payload URL</span>
              <div class="flex items-center gap-2">
                <code class="min-w-0 flex-1 truncate rounded-md border border-line bg-bg-panel px-2.5 py-1.5 text-xs text-text">
                  {{ payloadUrl }}
                </code>
                <Button type="button" variant="outline" size="sm" @click="copy('url', payloadUrl)">
                  {{ copiedField === 'url' ? 'Copied' : 'Copy' }}
                </Button>
              </div>
            </div>

            <div class="flex flex-col gap-1.5">
              <span class="text-sm font-medium text-text">Secret</span>
              <div class="flex items-center gap-2">
                <code class="min-w-0 flex-1 truncate rounded-md border border-line bg-bg-panel px-2.5 py-1.5 text-xs text-text">
                  {{ secretVisible ? repo.webhookSecret || '—' : '••••••••••••••••' }}
                </code>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  :aria-label="secretVisible ? 'Hide secret' : 'Show secret'"
                  :aria-pressed="secretVisible"
                  @click="secretVisible = !secretVisible"
                >
                  <Icon :name="secretVisible ? 'eye-off' : 'eye'" size="sm" />
                </Button>
                <Button type="button" variant="outline" size="sm" @click="copy('secret', repo.webhookSecret)">
                  {{ copiedField === 'secret' ? 'Copied' : 'Copy' }}
                </Button>
              </div>
            </div>

            <ConfirmDialog
              title="Rotate webhook secret?"
              description="The current secret stops working immediately — update the webhook configuration on the provider side with the new one."
              confirm-label="Rotate"
              :pending="rotating"
              @confirm="handleRotate"
            >
              <template #trigger>
                <Button type="button" variant="outline" size="sm" class="self-start">Rotate secret</Button>
              </template>
            </ConfirmDialog>

            <Alert status="info">
              Configure this payload URL and secret on the provider's webhook settings to enable automatic reviews.
            </Alert>
          </template>

          <Alert v-if="actionError" status="danger">{{ actionError }}</Alert>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
