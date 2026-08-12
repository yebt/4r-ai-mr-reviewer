<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Account } from '@shared/api/types'
import { useAccountsStore } from '@modules/accounts/store'

const props = defineProps<{ editing?: Account | null }>()
const emit = defineEmits<{ done: [] }>()

const store = useAccountsStore()

const form = reactive({ name: '', baseUrl: 'https://gitlab.com', token: '' })
const submitting = ref(false)
const error = ref<string | null>(null)
const showToken = ref(false)

const isEdit = computed(() => !!props.editing)

// Editing pre-fills name + base URL, but never the token (it is write-only and
// never returned): a blank token keeps the stored one, a new value rotates it.
watch(
  () => props.editing,
  (acc) => {
    if (acc) {
      form.name = acc.name
      form.baseUrl = acc.baseUrl
      form.token = ''
    } else {
      form.name = ''
      form.baseUrl = 'https://gitlab.com'
      form.token = ''
    }
    error.value = null
  },
  { immediate: true },
)

// On edit the token is optional (blank keeps the current one); on create it is
// required alongside the name and base URL.
const valid = computed(
  () => form.name.trim() && form.baseUrl.trim() && (isEdit.value || form.token.trim()),
)
const missing = computed(() => {
  const m: string[] = []
  if (!form.name.trim()) m.push('name')
  if (!form.baseUrl.trim()) m.push('base URL')
  if (!isEdit.value && !form.token.trim()) m.push('access token')
  return m
})

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    const input = {
      name: form.name.trim(),
      baseUrl: form.baseUrl.trim(),
      token: form.token.trim(),
    }
    if (props.editing) {
      await store.edit(props.editing.id, input)
      toast.success('Account updated')
    } else {
      await store.add(input)
      toast.success('Account added')
      form.name = ''
    }
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
        Access token
        <span v-if="!isEdit" class="text-accent" aria-hidden="true">*</span>
      </label>
      <div class="flex items-center gap-2">
        <input
          id="acc-token"
          v-model="form.token"
          :type="showToken ? 'text' : 'password'"
          class="field-underline"
          :placeholder="isEdit ? 'Leave blank to keep the current token' : 'glpat-…'"
          autocomplete="off"
          spellcheck="false"
          :aria-required="!isEdit"
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
        <template v-if="isEdit">Leave blank to keep the current token. </template>A personal access
        token with the <span class="font-mono">api</span> scope (add
        <span class="font-mono">read_repository</span> for deep reviews). Encrypted at rest on your
        instance; not shown again after saving.
      </p>
    </div>

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save changes' : 'Add account' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
      <p v-if="!valid && missing.length" class="text-muted w-full text-xs sm:w-auto">
        Still needed: {{ missing.join(', ') }}.
      </p>
    </div>
  </form>
</template>
