<script setup lang="ts">
/**
 * Providers settings section — organism. Dense list rows (title — muted
 * metadata — actions) with the secondary/destructive row actions (Set
 * default, Edit, Delete) collapsed into a Reka `DropdownMenu` triggered by a
 * trailing `⋯` icon-button, keeping only the single standout primary action
 * (Test) inline. This is the reference pattern every future settings
 * section should follow: `components/<Section>.vue` (list) +
 * `components/<Entity>Form.vue` (create/edit, opened in a Reka Dialog),
 * backed by a feature-scoped Pinia store.
 *
 * The default provider is always sorted first (`sortDefaultFirst`, stable —
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
import { Alert, Badge, Button, ConfirmDialog, Icon, Skeleton, Text } from '@shared/ui/design-system'
import ProviderForm from './ProviderForm.vue'
import { sortDefaultFirst, useProvidersStore } from '../store'
import type { Provider, TestProviderResult } from '../types'

const store = useProvidersStore()

const sortedProviders = computed(() => sortDefaultFirst(store.providers))

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

// Briefly highlighted after a row becomes the default, so the sort-to-top
// reads as an obviously reactive move rather than a silent reshuffle.
const highlightId = ref<string | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | undefined
onBeforeUnmount(() => clearTimeout(highlightTimer))

const settingDefaultId = ref<string | null>(null)
async function handleSetDefault(provider: Provider) {
  if (provider.isDefault) return
  settingDefaultId.value = provider.id
  try {
    // Rollback and the error toast are handled by the store's mutation
    // hooks; this catch only prevents an unhandled rejection here.
    await store.setDefaultProvider(provider.id)
    highlightId.value = provider.id
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
      <div class="flex min-w-0 flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Providers</Text>
        <Text muted size="sm" class="truncate">Configure the AI providers available to this workspace.</Text>
      </div>
      <Button class="whitespace-nowrap" @click="openCreateDialog">
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
        class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-8 w-20" />
      </div>
    </div>

    <Alert v-else-if="store.providersState.status === 'error'" status="danger">
      <p>{{ store.error?.message ?? 'Failed to load providers' }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

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
        v-for="provider in sortedProviders"
        :key="provider.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors duration-700"
        :class="highlightId === provider.id ? 'bg-accent-subtle-bg ring-1 ring-accent' : ''"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">{{ provider.name }}</Text>
            <Badge v-if="provider.isDefault" status="info">Default</Badge>
          </div>
          <Text muted size="sm" class="truncate">{{ subtitle(provider) }}</Text>
          <Alert
            v-if="testResults[provider.id]"
            :status="testResults[provider.id]!.ok ? 'success' : 'danger'"
            class="mt-1"
          >
            {{
              testResults[provider.id]!.ok
                ? 'Connection OK'
                : (testResults[provider.id]!.error ?? 'Connection failed')
            }}
          </Alert>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <Button variant="outline" size="sm" :loading="testingId === provider.id" @click="handleTest(provider)">
            Test
          </Button>

          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${provider.name}`">
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
                  v-if="!provider.isDefault"
                  :disabled="settingDefaultId === provider.id"
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-bg-hover"
                  @select="handleSetDefault(provider)"
                >
                  Set default
                </DropdownMenuItem>
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="openEditDialog(provider)"
                >
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Delete "${provider.name}"?`'
                  description="This removes the provider and its stored API key. This cannot be undone."
                  confirm-label="Delete"
                  danger
                  :pending="deletingId === provider.id"
                  @confirm="handleDelete(provider)"
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
