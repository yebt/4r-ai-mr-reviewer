<script setup lang="ts">
/**
 * Create/edit form for a single provider. Used inside a Reka `Dialog` by
 * `ProvidersSection`. Keyed by `provider?.id ?? 'create'` in the parent so
 * switching context (create -> edit another row) always remounts with fresh
 * local state, rather than trying to reset a shared instance in place.
 *
 * Field order and requiredness are kind-dependent:
 * - Base URL is required (with the `*`) only for `openai-compat`. `anthropic`
 *   and `gemini` have fixed default endpoints but can still proxy through a
 *   custom one, so the field stays visible (a power user can override it) but
 *   optional, with a muted hint. `openrouter` has a single fixed endpoint —
 *   the backend always stores an empty `baseUrl` for it — so the field is
 *   hidden entirely rather than shown as a confusing always-empty optional
 *   input.
 * - For `kind === 'openrouter'`, the "Model" (default) field moves after the
 *   Models section and becomes a `<select>` populated from `form.models`
 *   (the models the user has actually added), since the default model must
 *   be one of the configured ones. The other kinds keep "Model" as a
 *   free-text input in its original position, before API key.
 * - For `kind === 'openrouter'`, the "Add a model id" free-text fallback is
 *   replaced by a search input over the full catalog fetched once via
 *   `store.fetchOpenRouterModels`. Results render inline (in normal document
 *   flow, not a floating popover) as a bordered, scrollable list below the
 *   search input, so they push the rest of the form down instead of
 *   overlapping the Temperature field. The catalog is hundreds of models, so
 *   results only render once the query is non-empty and are capped to
 *   `OPENROUTER_RESULTS_LIMIT` — focusing/opening the search must stay
 *   instant regardless of catalog size. Client-side filtering uses Reka's
 *   `useFilter().contains`, the same helper CommandPalette.vue uses.
 *   Clicking a result appends its id to `form.models` (deduped).
 */
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { useFilter } from 'reka-ui'
import { Alert, Badge, Button, Checkbox, Field, Icon, Input, Select, Spinner, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useProvidersStore } from '../store'
import type { Provider, ProviderKind, TestProviderResult } from '../types'

const props = defineProps<{
  provider?: Provider | null
}>()

const emit = defineEmits<{
  saved: [provider: Provider]
  cancel: []
}>()

const store = useProvidersStore()

const isEditing = computed(() => props.provider != null)

const kindOptions: SelectItemOption[] = [
  { label: 'OpenAI-compatible', value: 'openai-compat' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Gemini', value: 'gemini' },
  { label: 'OpenRouter', value: 'openrouter' },
]

const form = reactive({
  name: props.provider?.name ?? '',
  kind: (props.provider?.kind ?? 'openai-compat') as ProviderKind,
  baseUrl: props.provider?.baseUrl ?? '',
  model: props.provider?.model ?? '',
  // Write-only on the server: a saved provider never returns its key, so
  // edit mode always starts blank ("leave blank to keep current").
  apiKey: '',
  temperatureInput: props.provider?.temperature != null ? String(props.provider.temperature) : '',
  models: [...(props.provider?.models ?? [])] as string[],
  makeDefault: false,
})

// Base URL only blocks submit for `openai-compat`: the other kinds have a
// fixed default endpoint and the backend accepts an empty `baseUrl` for them
// (confirmed against the live backend).
const isBaseUrlRequired = computed(() => form.kind === 'openai-compat')

type FormErrors = Partial<Record<'name' | 'baseUrl' | 'model' | 'temperature', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const newModelInput = ref('')
const saving = ref(false)
const testing = ref(false)
const testResult = ref<TestProviderResult | null>(null)
const formError = ref<string | null>(null)

function parseTemperature(input: string): number | null | 'invalid' {
  const trimmed = input.trim()
  if (trimmed === '') return null
  const value = Number(trimmed)
  return Number.isFinite(value) ? value : 'invalid'
}

function addModel() {
  const value = newModelInput.value.trim()
  if (value && !form.models.includes(value)) {
    form.models.push(value)
  }
  newModelInput.value = ''
}

function removeModel(model: string) {
  form.models = form.models.filter((existing) => existing !== model)
}

// --- OpenRouter model search (inline, in-flow results) --------------------

// The full catalog is hundreds of models — rendering it all (e.g. on every
// input focus) visibly lags the UI. Results only render once there's a
// non-empty query, and are capped to a small page so the rendered list
// itself stays cheap regardless of catalog size.
const OPENROUTER_RESULTS_LIMIT = 50

const openRouterQuery = ref('')

const { contains } = useFilter({ sensitivity: 'base' })

const filteredOpenRouterModels = computed(() => {
  const term = openRouterQuery.value.trim()
  if (!term) return []
  const matches = store.openRouterModels.filter(
    (model) => contains(model.name, term) || contains(model.id, term),
  )
  return matches.slice(0, OPENROUTER_RESULTS_LIMIT)
})

function addOpenRouterModel(id: string) {
  if (!form.models.includes(id)) form.models.push(id)
}

function ensureOpenRouterModelsLoaded() {
  if (store.openRouterModels.length === 0 && !store.openRouterLoading) {
    store.fetchOpenRouterModels()
  }
}

// The catalog is fetched once, lazily, the first time the form is showing
// (or switches to) the openrouter kind — never on module load.
watch(() => form.kind, (kind) => {
  if (kind === 'openrouter') ensureOpenRouterModelsLoaded()
}, { immediate: true })

// --------------------------------------------------------------------------

async function handleTestConnection() {
  testing.value = true
  testResult.value = null
  try {
    testResult.value = await store.testConnection({
      id: props.provider?.id,
      kind: form.kind,
      baseUrl: form.baseUrl,
      model: form.model,
      apiKey: form.apiKey,
    })
  } catch (err) {
    testResult.value = { ok: false, error: err instanceof Error ? err.message : 'Test failed' }
  } finally {
    testing.value = false
  }
}

async function handleSubmit() {
  errors.name = form.name.trim() ? undefined : 'Name is required'
  errors.baseUrl = isBaseUrlRequired.value && !form.baseUrl.trim() ? 'Base URL is required' : undefined
  errors.model = form.model.trim() ? undefined : 'Model is required'

  const temperature = parseTemperature(form.temperatureInput)
  errors.temperature = temperature === 'invalid' ? 'Must be a number' : undefined

  const hasErrors = Object.values(errors).some((value) => value !== undefined)
  if (hasErrors) {
    await nextTick()
    formRef.value?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
    return
  }

  saving.value = true
  formError.value = null
  const resolvedTemperature = temperature === 'invalid' ? null : temperature
  try {
    const result = props.provider
      ? await store.updateProvider(props.provider.id, {
          name: form.name,
          kind: form.kind,
          baseUrl: form.baseUrl,
          model: form.model,
          apiKey: form.apiKey,
          temperature: resolvedTemperature,
          models: form.models,
        })
      : await store.createProvider({
          name: form.name,
          kind: form.kind,
          baseUrl: form.baseUrl,
          model: form.model,
          apiKey: form.apiKey,
          makeDefault: form.makeDefault,
          temperature: resolvedTemperature,
          models: form.models,
        })
    emit('saved', result)
  } catch (err) {
    formError.value = err instanceof Error ? err.message : 'Failed to save provider'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <form ref="formRef" class="flex flex-col gap-4" novalidate @submit.prevent="handleSubmit">
    <Field label="Name" required :error="errors.name" v-slot="{ id, describedBy, invalid }">
      <Input :id="id" v-model="form.name" autocomplete="off" :aria-describedby="describedBy" :aria-invalid="invalid" />
    </Field>

    <Field label="Kind" required v-slot="{ id, describedBy }">
      <Select
        :id="id"
        :model-value="form.kind"
        :items="kindOptions"
        :aria-describedby="describedBy"
        @update:model-value="(value) => (form.kind = value as ProviderKind)"
      />
    </Field>

    <Field
      v-if="form.kind !== 'openrouter'"
      label="Base URL"
      :required="isBaseUrlRequired"
      :error="errors.baseUrl"
      :description="isBaseUrlRequired ? undefined : 'Optional — uses the provider\'s default endpoint.'"
      v-slot="{ id, describedBy, invalid }"
    >
      <Input
        :id="id"
        v-model="form.baseUrl"
        type="url"
        inputmode="url"
        placeholder="https://api.openai.com/v1"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field
      v-if="form.kind !== 'openrouter'"
      label="Model"
      required
      :error="errors.model"
      v-slot="{ id, describedBy, invalid }"
    >
      <Input
        :id="id"
        v-model="form.model"
        placeholder="gpt-4o"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field
      label="API key"
      :description="isEditing ? 'Leave blank to keep the current key.' : undefined"
      v-slot="{ id, describedBy }"
    >
      <Input
        :id="id"
        v-model="form.apiKey"
        type="password"
        autocomplete="off"
        :placeholder="isEditing ? 'leave blank to keep current' : 'sk-...'"
        :aria-describedby="describedBy"
      />
    </Field>

    <Field
      label="Temperature"
      description="Leave blank to use the provider default."
      :error="errors.temperature"
      v-slot="{ id, describedBy, invalid }"
    >
      <Input
        :id="id"
        v-model="form.temperatureInput"
        inputmode="decimal"
        placeholder="0.7"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <div class="flex flex-col gap-1.5">
      <span class="text-sm font-medium text-text">Models</span>

      <div v-if="form.kind === 'openrouter'" class="flex flex-col gap-1.5">
        <div
          class="flex h-8 items-center gap-2 rounded-md border border-line bg-bg-panel px-2.5 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-focus-ring"
        >
          <Icon name="search" size="sm" class="shrink-0 text-text-muted" />
          <input
            v-model="openRouterQuery"
            type="text"
            placeholder="Search OpenRouter models…"
            class="h-full min-w-0 flex-1 bg-transparent text-sm text-text outline-none placeholder:text-text-placeholder"
            @focus="ensureOpenRouterModelsLoaded"
          />
        </div>

        <!-- In-flow results panel: rendered in normal document flow (no
             portal/floating layer) so it pushes the rest of the form down
             instead of overlapping the Temperature field above. -->
        <div class="max-h-60 overflow-y-auto rounded-md border border-line bg-bg-panel-raised p-1">
          <div v-if="store.openRouterLoading" class="flex items-center gap-2 p-2">
            <Spinner size="sm" />
            <Text size="sm" muted>Loading models…</Text>
          </div>
          <p v-else-if="store.openRouterError" class="p-2 text-xs text-danger-text">
            {{ store.openRouterError }}
          </p>
          <p v-else-if="!openRouterQuery.trim()" class="p-2 text-xs text-text-muted">
            Type to search models…
          </p>
          <p v-else-if="filteredOpenRouterModels.length === 0" class="p-2 text-xs text-text-muted">
            No models match.
          </p>
          <ul v-else class="flex flex-col gap-0.5">
            <li v-for="orModel in filteredOpenRouterModels" :key="orModel.id">
              <button
                type="button"
                data-testid="openrouter-result"
                class="flex w-full flex-col gap-0.5 rounded-md px-2.5 py-1.5 text-left text-sm text-text outline-none hover:bg-accent-subtle-bg hover:text-accent-text-strong focus-visible:bg-accent-subtle-bg focus-visible:text-accent-text-strong"
                @click="addOpenRouterModel(orModel.id)"
              >
                <span class="truncate">{{ orModel.name }}</span>
                <span class="text-xs text-text-muted">{{ orModel.contextLength.toLocaleString() }} ctx</span>
              </button>
            </li>
          </ul>
        </div>
      </div>

      <div v-else class="flex gap-2">
        <Input v-model="newModelInput" placeholder="Add a model id" @keydown.enter.prevent="addModel" />
        <Button type="button" variant="outline" @click="addModel">Add</Button>
      </div>

      <div v-if="form.models.length > 0" class="flex flex-wrap gap-1.5">
        <Badge v-for="model in form.models" :key="model">
          {{ model }}
          <button
            type="button"
            class="ml-1 text-text-muted transition-colors hover:text-danger-text"
            :aria-label="`Remove ${model}`"
            @click="removeModel(model)"
          >
            <Icon name="x" size="xs" />
          </button>
        </Badge>
      </div>
    </div>

    <Field
      v-if="form.kind === 'openrouter'"
      label="Model"
      required
      :error="errors.model"
      :description="form.models.length === 0 ? 'Add models above first.' : undefined"
      v-slot="{ id, describedBy, invalid }"
    >
      <select
        :id="id"
        v-model="form.model"
        :disabled="form.models.length === 0"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
        class="flex h-8 w-full items-center justify-between gap-2 rounded-md border border-line bg-bg-panel px-2.5 text-sm text-text transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 aria-[invalid=true]:border-danger-solid"
      >
        <option value="" disabled>Select a default model…</option>
        <option v-for="modelId in form.models" :key="modelId" :value="modelId">{{ modelId }}</option>
      </select>
    </Field>

    <div v-if="!isEditing" class="flex items-center gap-2">
      <Checkbox
        id="provider-make-default"
        :model-value="form.makeDefault"
        @update:model-value="(value) => (form.makeDefault = value === true)"
      />
      <label for="provider-make-default" class="text-sm text-text">Make this the default provider</label>
    </div>

    <div class="flex items-center gap-3 border-t border-line-subtle pt-3">
      <Button type="button" variant="outline" :loading="testing" @click="handleTestConnection">
        Test connection
      </Button>
      <Alert v-if="testResult?.ok" status="success">Connection OK</Alert>
      <Alert v-else-if="testResult && !testResult.ok" status="danger">
        {{ testResult.error ?? 'Connection failed' }}
      </Alert>
    </div>

    <Alert v-if="formError" status="danger">{{ formError }}</Alert>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add provider' }}</Button>
    </div>
  </form>
</template>
