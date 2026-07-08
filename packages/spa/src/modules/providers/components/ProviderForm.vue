<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import type { ProviderKind } from '@shared/api/types'
import { useProvidersStore } from '@modules/providers/store'

const store = useProvidersStore()

const form = reactive({
  name: '',
  kind: 'openai-compat' as ProviderKind,
  baseUrl: '',
  model: '',
  apiKey: '',
  makeDefault: false,
})
const submitting = ref(false)
const error = ref<string | null>(null)

const valid = computed(() => form.name.trim() && form.apiKey.trim())

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    await store.add({
      name: form.name.trim(),
      kind: form.kind,
      baseUrl: form.baseUrl.trim(),
      model: form.model.trim(),
      apiKey: form.apiKey.trim(),
      makeDefault: form.makeDefault,
    })
    form.name = ''
    form.baseUrl = ''
    form.model = ''
    form.apiKey = ''
    form.makeDefault = false
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="flex flex-col gap-5" @submit.prevent="submit">
    <div class="label-mono">New provider</div>

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
      <label class="field-label" for="pv-key">API key</label>
      <input id="pv-key" v-model="form.apiKey" type="password" class="field-underline" placeholder="sk-… / gsk_…" autocomplete="off" />
    </div>

    <label class="flex cursor-pointer items-center gap-2 text-sm text-muted select-none">
      <input v-model="form.makeDefault" type="checkbox" class="accent-accent" />
      Make default
    </label>

    <p v-if="error" class="text-sm text-danger">{{ error }}</p>

    <button type="submit" class="btn-accent self-start" :disabled="!valid || submitting">
      <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
      {{ submitting ? 'Saving' : 'Add provider' }}
    </button>
  </form>
</template>
