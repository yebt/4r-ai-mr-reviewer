<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import { useAccountsStore } from '@modules/accounts/store'

const emit = defineEmits<{ done: [] }>()

const store = useAccountsStore()

const form = reactive({ name: '', baseUrl: 'https://gitlab.com', token: '' })
const submitting = ref(false)
const error = ref<string | null>(null)
const showToken = ref(false)

const valid = computed(() => form.name.trim() && form.baseUrl.trim() && form.token.trim())
const missing = computed(() => {
  const m: string[] = []
  if (!form.name.trim()) m.push('name')
  if (!form.baseUrl.trim()) m.push('base URL')
  if (!form.token.trim()) m.push('access token')
  return m
})

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    await store.add({
      name: form.name.trim(),
      baseUrl: form.baseUrl.trim(),
      token: form.token.trim(),
    })
    toast.success('Account added')
    form.name = ''
    form.token = ''
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
      <label class="field-label" for="acc-name">
        Name <span class="text-accent" aria-hidden="true">*</span>
      </label>
      <input
        id="acc-name"
        v-model="form.name"
        class="field-underline"
        placeholder="work"
        autocomplete="off"
        aria-required="true"
      />
    </div>

    <div>
      <label class="field-label" for="acc-url">
        GitLab base URL <span class="text-accent" aria-hidden="true">*</span>
      </label>
      <input
        id="acc-url"
        v-model="form.baseUrl"
        class="field-underline"
        placeholder="https://gitlab.com"
        autocomplete="off"
        aria-required="true"
      />
    </div>

    <div>
      <label class="field-label" for="acc-token">
        Access token <span class="text-accent" aria-hidden="true">*</span>
      </label>
      <div class="flex items-center gap-2">
        <input
          id="acc-token"
          v-model="form.token"
          :type="showToken ? 'text' : 'password'"
          class="field-underline"
          placeholder="glpat-…"
          autocomplete="off"
          spellcheck="false"
          aria-required="true"
        />
        <button
          type="button"
          class="btn-ghost shrink-0"
          :aria-label="showToken ? 'Hide token' : 'Show token'"
          :aria-pressed="showToken"
          @click="showToken = !showToken"
        >
          <span
            :class="showToken ? 'i-lucide-eye-off' : 'i-lucide-eye'"
            class="text-sm"
            aria-hidden="true"
          />
        </button>
      </div>
      <p class="text-muted mt-1.5 text-xs">
        A personal access token with the <span class="font-mono">api</span> scope (add
        <span class="font-mono">read_repository</span> for deep reviews). Encrypted at rest on your
        instance; not shown again after saving.
      </p>
    </div>

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : 'Add account' }}
      </button>
      <p v-if="!valid && missing.length" class="text-muted w-full text-xs sm:w-auto">
        Still needed: {{ missing.join(', ') }}.
      </p>
    </div>
  </form>
</template>
