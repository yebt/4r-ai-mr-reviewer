<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import type { Provider, ProviderKind } from '@shared/api/types'
import { useProvidersStore } from '@modules/providers/store'

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
})
const form = reactive(blank())
const submitting = ref(false)
const error = ref<string | null>(null)

const isEdit = computed(() => !!props.editing)
// In edit mode the API key is optional (blank = keep the stored one).
const valid = computed(() => form.name.trim() && (isEdit.value || form.apiKey.trim()))

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
    } else {
      Object.assign(form, blank())
    }
    error.value = null
  },
  { immediate: true },
)

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    const payload = {
      name: form.name.trim(),
      kind: form.kind,
      baseUrl: form.baseUrl.trim(),
      model: form.model.trim(),
      apiKey: form.apiKey.trim(),
    }
    if (props.editing) {
      await store.update(props.editing.id, payload)
    } else {
      await store.add({ ...payload, makeDefault: form.makeDefault })
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
    <h2 class="section-title flex items-center gap-2">
      <span class="inline-block h-3.5 w-0.5" :class="isEdit ? 'bg-flame' : 'bg-accent'" aria-hidden="true" />
      {{ isEdit ? 'Edit provider' : 'New provider' }}
    </h2>

    <div>
      <label class="field-label" for="pv-name">Name</label>
      <input id="pv-name" v-model="form.name" class="field-underline" placeholder="groq" autocomplete="off" />
    </div>

    <div>
      <label class="field-label" for="pv-kind">Kind</label>
      <select id="pv-kind" v-model="form.kind" class="field-underline">
        <option value="openai-compat">openai-compat (Groq, OpenAI, Moonshot, Kimi, OpenRouter)</option>
        <option value="anthropic">anthropic (Claude)</option>
      </select>
    </div>

    <div>
      <label class="field-label" for="pv-url">Base URL <span class="text-muted/60 normal-case">— optional</span></label>
      <input id="pv-url" v-model="form.baseUrl" class="field-underline" placeholder="https://api.groq.com/openai/v1" autocomplete="off" />
    </div>

    <div>
      <label class="field-label" for="pv-model">Default model</label>
      <input id="pv-model" v-model="form.model" class="field-underline" placeholder="llama-3.3-70b-versatile" autocomplete="off" />
    </div>

    <div>
      <label class="field-label" for="pv-key">
        API key <span v-if="isEdit" class="text-muted/60 normal-case">— leave blank to keep</span>
      </label>
      <input id="pv-key" v-model="form.apiKey" type="password" class="field-underline" :placeholder="isEdit ? '••••••••' : 'sk-… / gsk_…'" autocomplete="off" />
    </div>

    <label v-if="!isEdit" class="flex cursor-pointer items-center gap-2 text-sm text-muted select-none">
      <input v-model="form.makeDefault" type="checkbox" class="accent-accent" />
      Make default
    </label>

    <p v-if="error" class="text-sm text-danger">{{ error }}</p>

    <div class="flex items-center gap-3">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save changes' : 'Add provider' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
    </div>
  </form>
</template>
