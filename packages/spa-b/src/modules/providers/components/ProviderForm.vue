<script setup lang="ts">
/**
 * Create/edit form for a single provider. Used inside a Reka `Dialog` by
 * `ProvidersSection`. Keyed by `provider?.id ?? 'create'` in the parent so
 * switching context (create -> edit another row) always remounts with fresh
 * local state, rather than trying to reset a shared instance in place.
 */
import { nextTick, reactive, ref, computed } from 'vue'
import { Badge, Button, Checkbox, Field, Icon, Input, Select, Spinner, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useProvidersStore } from '../store'
import type { OpenRouterModel, Provider, ProviderKind, TestProviderResult } from '../types'

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

type FormErrors = Partial<Record<'name' | 'baseUrl' | 'model' | 'temperature', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const newModelInput = ref('')
const showModelBrowser = ref(false)
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

async function toggleModelBrowser() {
  showModelBrowser.value = !showModelBrowser.value
  if (showModelBrowser.value && store.openRouterModels.length === 0) {
    await store.fetchOpenRouterModels()
  }
}

function addOpenRouterModel(model: OpenRouterModel) {
  if (!form.models.includes(model.id)) form.models.push(model.id)
}

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
  errors.baseUrl = form.baseUrl.trim() ? undefined : 'Base URL is required'
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

    <Field label="Base URL" required :error="errors.baseUrl" v-slot="{ id, describedBy, invalid }">
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

    <Field label="Model" required :error="errors.model" v-slot="{ id, describedBy, invalid }">
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

      <div class="flex gap-2">
        <Input v-model="newModelInput" placeholder="Add a model id" @keydown.enter.prevent="addModel" />
        <Button type="button" variant="outline" @click="addModel">Add</Button>
      </div>

      <template v-if="form.kind === 'openrouter'">
        <Button type="button" variant="ghost" size="sm" class="self-start" @click="toggleModelBrowser">
          <template #leading><Icon name="list" size="sm" /></template>
          {{ showModelBrowser ? 'Hide' : 'Browse' }} OpenRouter models
        </Button>

        <div
          v-if="showModelBrowser"
          class="max-h-40 overflow-y-auto rounded-md border border-line bg-bg-panel p-1"
        >
          <div v-if="store.openRouterLoading" class="flex items-center gap-2 p-2">
            <Spinner size="sm" />
            <Text size="sm" muted>Loading models…</Text>
          </div>
          <p v-else-if="store.openRouterError" class="p-2 text-xs text-danger-text">
            {{ store.openRouterError }}
          </p>
          <p v-else-if="store.openRouterModels.length === 0" class="p-2 text-xs text-text-muted">
            No models found.
          </p>
          <button
            v-for="orModel in store.openRouterModels"
            :key="orModel.id"
            type="button"
            class="flex w-full items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-sm text-text transition-colors hover:bg-bg-hover"
            @click="addOpenRouterModel(orModel)"
          >
            <span class="truncate">{{ orModel.name }}</span>
            <span class="shrink-0 text-xs text-text-muted">{{ orModel.contextLength }} ctx</span>
          </button>
        </div>
      </template>
    </div>

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
      <Text v-if="testResult?.ok" size="sm" class="text-success-text">Connection OK</Text>
      <Text v-else-if="testResult && !testResult.ok" size="sm" class="text-danger-text">
        {{ testResult.error ?? 'Connection failed' }}
      </Text>
    </div>

    <p v-if="formError" role="alert" class="text-sm text-danger-text">{{ formError }}</p>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add provider' }}</Button>
    </div>
  </form>
</template>
