<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import type { Repo } from '@shared/api/types'
import { useReposStore } from '@modules/repos/store'
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'

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
    } else {
      await repos.add({
        name: form.name.trim(),
        url: form.url.trim(),
        accountId: form.accountId,
        providerId: form.providerId,
        model: form.model.trim(),
      })
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
      {{ isEdit ? 'Reassign provider' : 'New repository' }}
    </h2>

    <template v-if="!isEdit">
      <div>
        <label class="field-label" for="rp-name">Name</label>
        <input id="rp-name" v-model="form.name" class="field-underline" placeholder="web" autocomplete="off" />
      </div>
      <div>
        <label class="field-label" for="rp-url">Project URL</label>
        <input id="rp-url" v-model="form.url" class="field-underline" placeholder="https://gitlab.com/group/project" autocomplete="off" />
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
      <div class="font-mono text-xs text-muted">
        {{ form.name }} · {{ form.url }}
      </div>
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
      <input id="rp-model" v-model="form.model" class="field-underline" placeholder="use provider's model" autocomplete="off" />
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
