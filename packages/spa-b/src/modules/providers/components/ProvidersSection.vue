<script setup lang="ts">
/**
 * Providers settings section — organism. List + row actions (Test, Set
 * default, Edit, Delete-with-confirm) and an Add-provider Dialog. This is
 * the reference pattern every future settings section should follow:
 * `components/<Section>.vue` (list) + `components/<Entity>Form.vue` (create/
 * edit, opened in a Reka Dialog), backed by a feature-scoped Pinia store.
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
import ProviderForm from './ProviderForm.vue'
import { useProvidersStore } from '../store'
import type { Provider, TestProviderResult } from '../types'

const store = useProvidersStore()

const kindLabels: Record<Provider['kind'], string> = {
  'openai-compat': 'OpenAI-compatible',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  openrouter: 'OpenRouter',
}

/**
 * `baseUrl`/`model` can legitimately be empty (some provider kinds don't use
 * them), so never join in an empty segment — that leaves a bare trailing
 * "·" separator with nothing after it.
 */
function subtitle(provider: Provider): string {
  const parts = [kindLabels[provider.kind]]
  if (provider.model) parts.push(provider.model)
  return parts.join(' · ')
}

const dialogOpen = ref(false)
const editingProvider = ref<Provider | null>(null)

function openCreateDialog() {
  editingProvider.value = null
  dialogOpen.value = true
}

function openEditDialog(provider: Provider) {
  editingProvider.value = provider
  dialogOpen.value = true
}

function handleSaved() {
  dialogOpen.value = false
  editingProvider.value = null
}

function handleCancel() {
  dialogOpen.value = false
  editingProvider.value = null
}

const settingDefaultId = ref<string | null>(null)
async function handleSetDefault(provider: Provider) {
  if (provider.isDefault) return
  settingDefaultId.value = provider.id
  try {
    // Rollback and the error toast are handled by the store's mutation
    // hooks; this catch only prevents an unhandled rejection here.
    await store.setDefaultProvider(provider.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    settingDefaultId.value = null
  }
}

const deletingId = ref<string | null>(null)
async function handleDelete(provider: Provider) {
  deletingId.value = provider.id
  try {
    await store.removeProvider(provider.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    deletingId.value = null
  }
}

const testingId = ref<string | null>(null)
const testResults = ref<Record<string, TestProviderResult>>({})
async function handleTest(provider: Provider) {
  testingId.value = provider.id
  try {
    const result = await store.testConnection({
      id: provider.id,
      kind: provider.kind,
      baseUrl: provider.baseUrl,
      model: provider.model,
      // Blank apiKey + id: the server keeps using the stored key (same
      // keep-on-edit semantics as PATCH) when testing an already-saved provider.
      apiKey: '',
    })
    testResults.value = { ...testResults.value, [provider.id]: result }
  } catch (err) {
    testResults.value = {
      ...testResults.value,
      [provider.id]: { ok: false, error: err instanceof Error ? err.message : 'Test failed' },
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
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Providers</Text>
        <Text muted size="sm">Configure the AI providers available to this workspace.</Text>
      </div>
      <Button @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add provider
      </Button>
    </div>

    <div
      v-if="store.providersState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="providers-loading-skeleton"
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
      v-else-if="store.providersState.status === 'error'"
      role="alert"
      class="flex flex-col items-start gap-2 rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      <p>{{ store.error?.message ?? 'Failed to load providers' }}</p>
      <Button variant="outline" size="sm" @click="store.refetch()">Retry</Button>
    </div>

    <div
      v-else-if="store.providers.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="plug-zap" size="lg" class="text-text-muted" />
      <Text muted>No providers configured yet.</Text>
      <Button variant="outline" size="sm" @click="openCreateDialog">Add your first provider</Button>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="provider in store.providers"
        :key="provider.id"
        class="flex flex-col gap-3 rounded-lg border border-line bg-bg-panel p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="font-medium">{{ provider.name }}</Text>
            <Badge v-if="provider.isDefault" status="info">Default</Badge>
          </div>
          <Text muted size="sm">{{ subtitle(provider) }}</Text>
          <Text
            v-if="testResults[provider.id]"
            size="xs"
            :class="testResults[provider.id]!.ok ? 'text-success-text' : 'text-danger-text'"
          >
            {{
              testResults[provider.id]!.ok
                ? 'Connection OK'
                : (testResults[provider.id]!.error ?? 'Connection failed')
            }}
          </Text>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button variant="outline" size="sm" :loading="testingId === provider.id" @click="handleTest(provider)">
            Test
          </Button>
          <Button
            v-if="!provider.isDefault"
            variant="ghost"
            size="sm"
            :loading="settingDefaultId === provider.id"
            @click="handleSetDefault(provider)"
          >
            Set default
          </Button>
          <Button variant="ghost" size="sm" @click="openEditDialog(provider)">Edit</Button>

          <ConfirmDialog
            :title='`Delete "${provider.name}"?`'
            description="This removes the provider and its stored API key. This cannot be undone."
            confirm-label="Delete"
            danger
            :pending="deletingId === provider.id"
            @confirm="handleDelete(provider)"
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
            {{ editingProvider ? 'Edit provider' : 'Add provider' }}
          </DialogTitle>
          <DialogDescription class="mt-1 text-sm text-text-muted">
            {{
              editingProvider
                ? "Update this provider's connection settings."
                : 'Connect a new AI provider to this workspace.'
            }}
          </DialogDescription>
          <ProviderForm
            :key="editingProvider?.id ?? 'create'"
            class="mt-4"
            :provider="editingProvider"
            @saved="handleSaved"
            @cancel="handleCancel"
          />
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>
