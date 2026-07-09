<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Repo } from '@shared/api/types'
import { useReposStore } from '@modules/repos/store'
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'
import { matchAccountId, parseRepoUrl } from '@modules/repos/url'

const props = defineProps<{ editing?: Repo | null }>()
const emit = defineEmits<{ done: [] }>()

const repos = useReposStore()
const accounts = useAccountsStore()
const providers = useProvidersStore()

onMounted(() => {
  if (accounts.items.length === 0) accounts.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
})

const blank = () => ({ name: '', url: '', accountId: '', providerId: '', model: '' })
const form = reactive(blank())
const submitting = ref(false)
const error = ref<string | null>(null)

const isEdit = computed(() => !!props.editing)
const valid = computed(() => isEdit.value || (form.name.trim() && form.url.trim() && form.accountId))

// Derive fields from the pasted URL, keeping any manual edits.
const parsed = computed(() => parseRepoUrl(form.url))
const lastAutoName = ref('')
const matchedAccount = computed(() => accounts.items.find((a) => a.id === form.accountId) ?? null)

// Model suggestions come from the chosen provider (or the default one).
const selectedProvider = computed(() => {
  if (form.providerId) return providers.items.find((p) => p.id === form.providerId) ?? null
  return providers.items.find((p) => p.isDefault) ?? null
})
const modelPresets = computed(() => selectedProvider.value?.models ?? [])

// Resolve name + account from the URL when the field loses focus, keeping any
// manual edits the user already made.
function resolveFromUrl() {
  const p = parsed.value
  if (!p.valid) return
  if (form.name === '' || form.name === lastAutoName.value) {
    form.name = p.name
    lastAutoName.value = p.name
  }
  if (form.accountId === '') {
    const id = matchAccountId(p.origin, accounts.items)
    if (id) form.accountId = id
  }
}

watch(
  () => props.editing,
  (r) => {
    if (r) {
      form.name = r.name
      form.url = r.url
      form.accountId = r.accountId
      form.providerId = r.providerId
      form.model = r.model
    } else {
      Object.assign(form, blank())
      lastAutoName.value = ''
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
    if (props.editing) {
      await repos.assign(props.editing.id, { providerId: form.providerId, model: form.model.trim() })
      toast.success('Repository updated')
    } else {
      await repos.add({
        name: form.name.trim(),
        url: form.url.trim(),
        accountId: form.accountId,
        providerId: form.providerId,
        model: form.model.trim(),
      })
      toast.success('Repository added')
    }
    Object.assign(form, blank())
    lastAutoName.value = ''
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
    <template v-if="!isEdit">
      <div>
        <label class="field-label" for="rp-url">Project URL</label>
        <input id="rp-url" v-model="form.url" class="field-underline" placeholder="https://gitlab.com/group/project" autocomplete="off" @blur="resolveFromUrl" />
        <p v-if="form.url && !parsed.valid" class="mt-1.5 text-xs text-warn">
          Enter a full project URL (https://host/group/project).
        </p>
        <p v-else-if="parsed.valid" class="mt-1.5 label-mono">
          {{ parsed.path }}
          <template v-if="matchedAccount"> · matched {{ matchedAccount.name }}</template>
          <template v-else-if="accounts.items.length"> · no account for this host</template>
        </p>
      </div>

      <div>
        <label class="field-label" for="rp-name">Name <span class="text-muted/60 normal-case">— from URL, editable</span></label>
        <input id="rp-name" v-model="form.name" class="field-underline" placeholder="project" autocomplete="off" />
      </div>

      <div>
        <label class="field-label" for="rp-account">Account</label>
        <select id="rp-account" v-model="form.accountId" class="field-underline">
          <option value="" disabled>Select an account…</option>
          <option v-for="a in accounts.items" :key="a.id" :value="a.id">{{ a.name }} — {{ a.baseUrl }}</option>
        </select>
        <p v-if="accounts.items.length === 0" class="mt-1.5 text-xs text-muted/70">Add an account first.</p>
      </div>
    </template>

    <template v-else>
      <div class="font-mono text-xs text-muted">{{ form.name }} · {{ form.url }}</div>
    </template>

    <div>
      <label class="field-label" for="rp-provider">Provider <span class="text-muted/60 normal-case">— optional</span></label>
      <select id="rp-provider" v-model="form.providerId" class="field-underline">
        <option value="">Use default provider</option>
        <option v-for="p in providers.items" :key="p.id" :value="p.id">{{ p.name }}{{ p.isDefault ? ' (default)' : '' }}</option>
      </select>
    </div>

    <div>
      <label class="field-label" for="rp-model">Model <span class="text-muted/60 normal-case">— optional</span></label>
      <input
        id="rp-model"
        v-model="form.model"
        class="field-underline"
        list="rp-model-presets"
        placeholder="use provider's model"
        autocomplete="off"
      />
      <datalist id="rp-model-presets">
        <option v-for="m in modelPresets" :key="m" :value="m" />
      </datalist>
      <p v-if="modelPresets.length" class="mt-1.5 text-xs text-muted/70">
        Presets: {{ modelPresets.join(', ') }}
      </p>
    </div>

    <p v-if="error" class="text-sm text-danger">{{ error }}</p>

    <div class="flex items-center gap-3">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save' : 'Track repository' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
    </div>
  </form>
</template>
