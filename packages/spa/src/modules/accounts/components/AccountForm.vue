<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { useAccountsStore } from '@modules/accounts/store'

const store = useAccountsStore()

const form = reactive({ name: '', baseUrl: 'https://gitlab.com', token: '' })
const submitting = ref(false)
const error = ref<string | null>(null)

const valid = computed(() => form.name.trim() && form.baseUrl.trim() && form.token.trim())

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    await store.add({ name: form.name.trim(), baseUrl: form.baseUrl.trim(), token: form.token.trim() })
    form.name = ''
    form.token = ''
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
      <span class="inline-block h-3.5 w-0.5 bg-accent" aria-hidden="true" />
      New account
    </h2>

    <div>
      <label class="field-label" for="acc-name">Name</label>
      <input id="acc-name" v-model="form.name" class="field-underline" placeholder="work" autocomplete="off" />
    </div>

    <div>
      <label class="field-label" for="acc-url">GitLab base URL</label>
      <input id="acc-url" v-model="form.baseUrl" class="field-underline" placeholder="https://gitlab.com" autocomplete="off" />
    </div>

    <div>
      <label class="field-label" for="acc-token">Access token</label>
      <input id="acc-token" v-model="form.token" type="password" class="field-underline" placeholder="glpat-…" autocomplete="off" />
      <p class="mt-1.5 text-xs text-muted/70">Stored encrypted by the backend. Write-only.</p>
    </div>

    <p v-if="error" class="text-sm text-danger">{{ error }}</p>

    <button type="submit" class="btn-accent self-start" :disabled="!valid || submitting">
      <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
      {{ submitting ? 'Saving' : 'Add account' }}
    </button>
  </form>
</template>
