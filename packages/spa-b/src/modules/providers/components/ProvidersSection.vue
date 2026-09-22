<script setup lang="ts">
/**
 * Providers settings section — organism. List + row actions (Test, Set
 * default, Edit, Delete-with-confirm) and an Add-provider Dialog. This is
 * the reference pattern every future settings section should follow:
 * `components/<Section>.vue` (list) + `components/<Entity>Form.vue` (create/
 * edit, opened in a Reka Dialog), backed by a feature-scoped Pinia store.
 */
import { onMounted, ref } from 'vue'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  AlertDialogTrigger,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { Badge, Button, Icon, Spinner, Text } from '@shared/ui/design-system'
import ProviderForm from './ProviderForm.vue'
import { useProvidersStore } from '../store'
import type { Provider, TestProviderResult } from '../types'

const store = useProvidersStore()

onMounted(() => {
  store.fetchProviders()
})

const kindLabels: Record<Provider['kind'], string> = {
  'openai-compat': 'OpenAI-compatible',
  anthropic: 'Anthropic',
  gemini: 'Gemini',
  openrouter: 'OpenRouter',
}

const dialogOpen = ref(false)
const editingProvider = ref<Provider | null>(null)
const actionError = ref<string | null>(null)

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
  actionError.value = null
  settingDefaultId.value = provider.id
  try {
    await store.setDefaultProvider(provider.id)
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : 'Failed to set default provider'
  } finally {
    settingDefaultId.value = null
  }
}

const deletingId = ref<string | null>(null)
async function handleDelete(provider: Provider) {
  actionError.value = null
  deletingId.value = provider.id
  try {
    await store.removeProvider(provider.id)
  } catch (err) {
    actionError.value = err instanceof Error ? err.message : 'Failed to delete provider'
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
        <template #leading><Icon name="lucide:plus" size="sm" /></template>
        Add provider
      </Button>
    </div>

    <p
      v-if="actionError"
      role="alert"
      class="rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      {{ actionError }}
    </p>

    <div
      v-if="store.loading && store.providers.length === 0"
      class="flex items-center gap-2 p-6 text-text-muted"
    >
      <Spinner size="sm" /> Loading providers…
    </div>

    <p
      v-else-if="store.error"
      role="alert"
      class="rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      {{ store.error }}
    </p>

    <div
      v-else-if="store.providers.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="lucide:plug-zap" size="lg" class="text-text-muted" />
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
          <Text muted size="sm">{{ kindLabels[provider.kind] }} · {{ provider.model }}</Text>
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

          <AlertDialogRoot>
            <AlertDialogTrigger as-child>
              <Button variant="ghost" size="sm" class="text-danger-text hover:bg-danger-bg">Delete</Button>
            </AlertDialogTrigger>
            <AlertDialogPortal>
              <AlertDialogOverlay class="fixed inset-0 z-20 bg-gray-12/40" />
              <AlertDialogContent
                class="fixed left-1/2 top-1/2 z-30 w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
              >
                <AlertDialogTitle class="text-md font-semibold">Delete "{{ provider.name }}"?</AlertDialogTitle>
                <AlertDialogDescription class="mt-1 text-sm text-text-muted">
                  This removes the provider and its stored API key. This cannot be undone.
                </AlertDialogDescription>
                <div class="mt-5 flex justify-end gap-2">
                  <AlertDialogCancel
                    class="rounded-md border border-line px-3 py-1.5 text-sm text-text transition-colors hover:bg-bg-hover"
                  >
                    Cancel
                  </AlertDialogCancel>
                  <AlertDialogAction
                    class="rounded-md bg-danger-solid px-3 py-1.5 text-sm font-medium text-white transition-colors hover:opacity-90"
                    @click="handleDelete(provider)"
                  >
                    Delete
                  </AlertDialogAction>
                </div>
              </AlertDialogContent>
            </AlertDialogPortal>
          </AlertDialogRoot>
        </div>
      </li>
    </ul>

    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay class="fixed inset-0 z-20 bg-gray-12/40" />
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
