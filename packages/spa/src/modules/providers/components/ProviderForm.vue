<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { OpenRouterModel, Provider, ProviderKind } from '@shared/api/types'
import { useProvidersStore } from '@modules/providers/store'

// Case-insensitive alphabetical comparator, reused for model lists/chips.
const byLower = (a: string, b: string) => a.toLowerCase().localeCompare(b.toLowerCase())

const props = defineProps<{ editing?: Provider | null }>()
const emit = defineEmits<{ done: [] }>()

const store = useProvidersStore()

const blank = () => ({
  name: '',
  kind: 'openai-compat' as ProviderKind,
  baseUrl: '',
  model: '',
  apiKey: '',
  makeDefault: false,
  temperature: '', // string; empty = don't send (model default)
  models: [] as string[],
})
const form = reactive(blank())
const submitting = ref(false)
const error = ref<string | null>(null)
const showAdvanced = ref(false)
const modelDraft = ref('')

const isEdit = computed(() => !!props.editing)
const valid = computed(() => form.name.trim() && (isEdit.value || form.apiKey.trim()))

// Selected model presets, shown alphabetically (non-mutating copy).
const sortedModels = computed(() => [...form.models].sort(byLower))

// --- OpenRouter model browser ---
// The catalog is fetched once (lazily, when the browser is first shown) and
// cached in-component so switching kinds back and forth doesn't refetch.
const orModels = ref<OpenRouterModel[]>([])
const orLoaded = ref(false)
const orLoading = ref(false)
const orError = ref<string | null>(null)
const modelSearch = ref('')

async function loadOpenRouterModels() {
  if (orLoaded.value || orLoading.value) return
  orLoading.value = true
  orError.value = null
  try {
    orModels.value = await api.listOpenRouterModels()
    orLoaded.value = true
  } catch (e) {
    orError.value = errorMessage(e)
  } finally {
    orLoading.value = false
  }
}

// Alphabetical by id (the backend already sorts; sort again defensively so the
// list is stable regardless of source), filtered by the search text.
const filteredOrModels = computed(() => {
  const q = modelSearch.value.trim().toLowerCase()
  const list = [...orModels.value].sort((a, b) => byLower(a.id, b.id))
  if (!q) return list
  return list.filter((m) => m.id.toLowerCase().includes(q) || m.name.toLowerCase().includes(q))
})

function pickModel(id: string) {
  if (!form.models.includes(id)) form.models.push(id)
  if (!form.model.trim()) form.model = id
}

// Lazy-load the catalog whenever the kind becomes openrouter (first time only).
watch(
  () => form.kind,
  (kind) => {
    if (kind === 'openrouter') void loadOpenRouterModels()
  },
  { immediate: true },
)

watch(
  () => props.editing,
  (p) => {
    if (p) {
      form.name = p.name
      form.kind = p.kind
      form.baseUrl = p.baseUrl
      form.model = p.model
      form.apiKey = ''
      form.makeDefault = false
      form.temperature = p.temperature == null ? '' : String(p.temperature)
      form.models = [...p.models]
      showAdvanced.value = p.temperature != null || p.models.length > 0
    } else {
      Object.assign(form, blank())
      showAdvanced.value = false
    }
    error.value = null
  },
  { immediate: true },
)

function addModel() {
  const m = modelDraft.value.trim()
  if (m && !form.models.includes(m)) form.models.push(m)
  modelDraft.value = ''
}

function removeModel(m: string) {
  form.models = form.models.filter((x) => x !== m)
}

function parseTemperature(): number | null | 'invalid' {
  const raw = form.temperature.trim()
  if (raw === '') return null
  const n = Number(raw)
  return Number.isNaN(n) ? 'invalid' : n
}

async function submit() {
  if (!valid.value || submitting.value) return
  const temperature = parseTemperature()
  if (temperature === 'invalid') {
    error.value = 'Temperature must be a number.'
    return
  }

  submitting.value = true
  error.value = null
  try {
    const base = {
      name: form.name.trim(),
      kind: form.kind,
      baseUrl: form.baseUrl.trim(),
      model: form.model.trim(),
      apiKey: form.apiKey.trim(),
      temperature,
      models: [...form.models],
    }
    if (props.editing) {
      await store.update(props.editing.id, base)
      toast.success('Provider updated')
    } else {
      await store.add({ ...base, makeDefault: form.makeDefault })
      toast.success('Provider added')
    }
    Object.assign(form, blank())
    emit('done')
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="flex flex-col gap-5" @submit.prevent="submit">
    <div>
      <label class="field-label" for="pv-name">Name</label>
      <input
        id="pv-name"
        v-model="form.name"
        class="field-underline"
        placeholder="groq"
        autocomplete="off"
      />
    </div>

    <div>
      <label class="field-label" for="pv-kind">Kind</label>
      <select id="pv-kind" v-model="form.kind" class="field-underline">
        <option value="openai-compat">
          openai-compat (Groq, OpenAI, Moonshot, Kimi, OpenRouter)
        </option>
        <option value="anthropic">anthropic (Claude)</option>
        <option value="gemini">gemini (Google Gemini)</option>
        <option value="openrouter">openrouter (browse models)</option>
      </select>
    </div>

    <div>
      <label class="field-label" for="pv-url"
        >Base URL <span class="text-muted/60 normal-case">— optional</span></label
      >
      <input
        id="pv-url"
        v-model="form.baseUrl"
        class="field-underline"
        placeholder="https://api.groq.com/openai/v1"
        autocomplete="off"
      />
      <p v-if="form.kind === 'gemini'" class="text-muted mt-1 text-xs">
        Leave empty to use Google's Gemini endpoint. Use a Gemini model such as
        <code>gemini-2.5-flash</code>.
      </p>
      <p v-else-if="form.kind === 'openrouter'" class="text-muted mt-1 text-xs">
        Leave Base URL empty to use OpenRouter. Browse and add models below.
      </p>
    </div>

    <div>
      <label class="field-label" for="pv-model">Default model</label>
      <input
        id="pv-model"
        v-model="form.model"
        class="field-underline"
        placeholder="llama-3.3-70b-versatile"
        autocomplete="off"
      />
    </div>

    <!-- OpenRouter model browser -->
    <div v-if="form.kind === 'openrouter'">
      <label class="field-label" for="pv-or-search">Browse OpenRouter models</label>
      <input
        id="pv-or-search"
        v-model="modelSearch"
        class="field-underline"
        placeholder="search by id or name"
        autocomplete="off"
      />

      <p v-if="orLoading" class="text-muted mt-3 flex items-center gap-2 text-xs">
        <span class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        Loading models…
      </p>
      <p v-else-if="orError" class="text-danger mt-3 text-xs">
        {{ orError }}
        <button type="button" class="btn-line ml-2 text-xs" @click="loadOpenRouterModels">
          Retry
        </button>
      </p>
      <p v-else-if="filteredOrModels.length === 0" class="text-muted mt-3 text-xs">
        No models match “{{ modelSearch }}”.
      </p>
      <ul
        v-else
        class="border-line/50 mt-3 max-h-64 divide-y divide-line/40 overflow-y-auto border"
      >
        <li v-for="m in filteredOrModels" :key="m.id">
          <button
            type="button"
            class="hover:bg-surface flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left"
            :class="form.models.includes(m.id) ? 'text-muted' : 'text-ink'"
            @click="pickModel(m.id)"
          >
            <span class="flex w-full items-center gap-2">
              <span class="truncate font-mono text-xs">{{ m.id }}</span>
              <span
                v-if="form.models.includes(m.id)"
                class="text-ok ml-auto text-xs"
                aria-hidden="true"
                >added</span
              >
            </span>
            <span class="text-muted truncate text-xs">{{ m.name }}</span>
          </button>
        </li>
      </ul>
      <p class="text-muted/70 mt-2 text-xs">
        Click a model to add it to the preset list. The first one you add fills the default model
        when it's empty.
      </p>
    </div>

    <div>
      <label class="field-label" for="pv-key">
        API key <span v-if="isEdit" class="text-muted/60 normal-case">— leave blank to keep</span>
      </label>
      <input
        id="pv-key"
        v-model="form.apiKey"
        type="password"
        class="field-underline"
        :placeholder="isEdit ? '••••••••' : 'sk-… / gsk_…'"
        autocomplete="off"
      />
    </div>

    <label
      v-if="!isEdit"
      class="text-muted flex cursor-pointer items-center gap-2 text-sm select-none"
    >
      <input v-model="form.makeDefault" type="checkbox" class="accent-accent" />
      Make default
    </label>

    <!-- Advanced -->
    <div class="border-line/50 border-t pt-4">
      <button
        type="button"
        class="label-mono hover:text-ink flex items-center gap-1"
        @click="showAdvanced = !showAdvanced"
      >
        <span
          :class="showAdvanced ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
          aria-hidden="true"
        />
        Advanced
      </button>

      <div v-if="showAdvanced" class="mt-4 flex flex-col gap-5">
        <div>
          <label class="field-label" for="pv-temp">
            Temperature
            <span class="text-muted/60 normal-case">— blank uses the model default</span>
          </label>
          <input
            id="pv-temp"
            v-model="form.temperature"
            class="field-underline"
            inputmode="decimal"
            placeholder="model default"
            autocomplete="off"
          />
          <p class="text-muted/70 mt-1.5 text-xs">
            Some models reject any value other than their default — leave blank for those.
          </p>
        </div>

        <div>
          <label class="field-label">Model presets</label>
          <div class="flex gap-2">
            <input
              v-model="modelDraft"
              class="field-underline"
              placeholder="add a model name"
              autocomplete="off"
              @keydown.enter.prevent="addModel"
            />
            <button type="button" class="btn-line text-xs" @click="addModel">Add</button>
          </div>
          <div v-if="form.models.length" class="mt-3 flex flex-wrap gap-2">
            <span
              v-for="m in sortedModels"
              :key="m"
              class="border-line text-ink inline-flex items-center gap-1 border px-2 py-1 font-mono text-xs"
            >
              {{ m }}
              <button
                type="button"
                class="text-muted hover:text-danger flex"
                :aria-label="`Remove ${m}`"
                @click="removeModel(m)"
              >
                <span class="i-lucide-x text-xs" aria-hidden="true" />
              </button>
            </span>
          </div>
          <p class="text-muted/70 mt-2 text-xs">
            Pick from these (no typos) when configuring a repository.
          </p>
        </div>
      </div>
    </div>

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div class="flex items-center gap-3">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save changes' : 'Add provider' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
    </div>
  </form>
</template>
