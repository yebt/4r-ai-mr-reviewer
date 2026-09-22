<script setup lang="ts">
/**
 * Create/edit form for a single GitLab account. Used inside a Reka `Dialog`
 * by `AccountsSection`. Keyed by `account?.id ?? 'create'` in the parent so
 * switching context (create -> edit another row) always remounts with fresh
 * local state, rather than trying to reset a shared instance in place.
 */
import { nextTick, reactive, ref, computed } from 'vue'
import { Field, Input, Button } from '@shared/ui/design-system'
import { useAccountsStore } from '../store'
import type { Account } from '../types'

const props = defineProps<{
  account?: Account | null
}>()

const emit = defineEmits<{
  saved: [account: Account]
  cancel: []
}>()

const store = useAccountsStore()

const isEditing = computed(() => props.account != null)

const form = reactive({
  name: props.account?.name ?? '',
  baseUrl: props.account?.baseUrl ?? '',
  // Write-only on the server: a saved account never returns its token, so
  // edit mode always starts blank ("leave blank to keep current").
  token: '',
})

type FormErrors = Partial<Record<'name' | 'baseUrl', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const saving = ref(false)
const formError = ref<string | null>(null)

async function handleSubmit() {
  errors.name = form.name.trim() ? undefined : 'Name is required'
  errors.baseUrl = form.baseUrl.trim() ? undefined : 'Base URL is required'

  const hasErrors = Object.values(errors).some((value) => value !== undefined)
  if (hasErrors) {
    await nextTick()
    formRef.value?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
    return
  }

  saving.value = true
  formError.value = null
  try {
    const result = props.account
      ? await store.updateAccount(props.account.id, {
          name: form.name,
          baseUrl: form.baseUrl,
          token: form.token,
        })
      : await store.createAccount({
          name: form.name,
          baseUrl: form.baseUrl,
          token: form.token,
        })
    emit('saved', result)
  } catch (err) {
    formError.value = err instanceof Error ? err.message : 'Failed to save account'
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

    <Field label="Base URL" required :error="errors.baseUrl" v-slot="{ id, describedBy, invalid }">
      <Input
        :id="id"
        v-model="form.baseUrl"
        type="url"
        inputmode="url"
        placeholder="https://gitlab.com"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field
      label="Token"
      :description="isEditing ? 'Leave blank to keep the current token.' : undefined"
      v-slot="{ id, describedBy }"
    >
      <Input
        :id="id"
        v-model="form.token"
        type="password"
        autocomplete="off"
        :placeholder="isEditing ? 'leave blank to keep current' : 'glpat-...'"
        :aria-describedby="describedBy"
      />
    </Field>

    <p v-if="formError" role="alert" class="text-sm text-danger-text">{{ formError }}</p>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add account' }}</Button>
    </div>
  </form>
</template>
