<script setup lang="ts">
/**
 * Create/reassign form for a single repo. Used inside a Reka `Dialog` by
 * `RepositoriesSection`. Keyed by `repo?.id ?? 'create'` in the parent so
 * switching context always remounts with fresh local state.
 *
 * CREATE mode: Account (required Select — `name`/`url` are only ever set at
 * create time, so both are plain required Inputs here) + Provider/Model/
 * Profile (all optional — "use default"). If there are no accounts yet, the
 * form is replaced by an info Alert pointing at /settings/accounts, since
 * `accountId` is required by the backend and there's nothing to pick.
 *
 * EDIT mode ("Reassign"): `name`/`url` are NOT editable after create (per
 * `PATCH /repos/{id}/assign`), so they render as read-only text; only
 * Account/Provider/Model/Profile are editable and submit `assignRepo`.
 *
 * Provider/Profile are optional-with-a-"use default"-choice, but Reka's
 * `Select` reserves the empty string for its own placeholder/unset state, so
 * the sentinel values below stand in for "" in the UI and get mapped back to
 * "" right before the payload is built.
 */
import { computed, nextTick, reactive, ref } from 'vue'
import { Alert, Button, Field, Input, Select, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
// Deep-imported (not through each module's barrel, which only re-exports
// its Section component + types, not the store) — read-only reuse, per the
// task boundary: this module never edits accounts/providers/profiles.
import { useAccountsStore } from '@modules/accounts/store'
import { useProfilesStore } from '@modules/profiles/store'
import { useProvidersStore } from '@modules/providers/store'
import { useReposStore } from '../store'
import type { Repo } from '../types'

const NO_PROVIDER_VALUE = '__use_default_provider__'
const NO_PROFILE_VALUE = '__no_default_profile__'

const props = defineProps<{
  repo?: Repo | null
}>()

const emit = defineEmits<{
  saved: [repo: Repo]
  cancel: []
}>()

const store = useReposStore()
const accountsStore = useAccountsStore()
const providersStore = useProvidersStore()
const profilesStore = useProfilesStore()

const isEditing = computed(() => props.repo != null)
const hasAccounts = computed(() => accountsStore.accounts.length > 0)

const accountOptions = computed<SelectItemOption[]>(() =>
  accountsStore.accounts.map((account) => ({ label: account.name, value: account.id })),
)
const providerOptions = computed<SelectItemOption[]>(() => [
  { label: 'Use default provider', value: NO_PROVIDER_VALUE },
  ...providersStore.providers.map((provider) => ({ label: provider.name, value: provider.id })),
])
const profileOptions = computed<SelectItemOption[]>(() => [
  { label: 'No default voice', value: NO_PROFILE_VALUE },
  ...profilesStore.profiles.map((profile) => ({ label: profile.name, value: profile.id })),
])

const form = reactive({
  accountId: props.repo?.accountId ?? '',
  url: props.repo?.url ?? '',
  name: props.repo?.name ?? '',
  providerId: props.repo?.providerId ? props.repo.providerId : NO_PROVIDER_VALUE,
  model: props.repo?.model ?? '',
  profileId: props.repo?.defaultProfileId ? props.repo.defaultProfileId : NO_PROFILE_VALUE,
})

type FormErrors = Partial<Record<'accountId' | 'url' | 'name', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const saving = ref(false)
const formError = ref<string | null>(null)

async function handleSubmit() {
  errors.accountId = form.accountId ? undefined : 'Account is required'
  errors.url = !isEditing.value && !form.url.trim() ? 'URL is required' : undefined
  errors.name = !isEditing.value && !form.name.trim() ? 'Name is required' : undefined

  const hasErrors = Object.values(errors).some((value) => value !== undefined)
  if (hasErrors) {
    await nextTick()
    formRef.value?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
    return
  }

  saving.value = true
  formError.value = null
  const resolvedProviderId = form.providerId === NO_PROVIDER_VALUE ? '' : form.providerId
  const resolvedProfileId = form.profileId === NO_PROFILE_VALUE ? '' : form.profileId
  try {
    const result = props.repo
      ? await store.assignRepo(props.repo.id, {
          accountId: form.accountId,
          providerId: resolvedProviderId,
          model: form.model,
          profileId: resolvedProfileId,
        })
      : await store.createRepo({
          name: form.name,
          url: form.url,
          accountId: form.accountId,
          providerId: resolvedProviderId,
          model: form.model,
          profileId: resolvedProfileId,
        })
    emit('saved', result)
  } catch (err) {
    formError.value = err instanceof Error ? err.message : 'Failed to save repository'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-if="!isEditing && !hasAccounts" class="flex flex-col gap-4">
    <Alert status="info">
      You need at least one account before adding a repository.
      <RouterLink to="/settings/accounts" class="font-medium underline underline-offset-2">
        Add an account
      </RouterLink>
      first.
    </Alert>
    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
    </div>
  </div>

  <form v-else ref="formRef" class="flex flex-col gap-4" novalidate @submit.prevent="handleSubmit">
    <template v-if="isEditing">
      <div class="flex flex-col gap-1.5">
        <span class="text-sm font-medium text-text">Name</span>
        <Text size="sm" muted>{{ form.name }}</Text>
      </div>
      <div class="flex flex-col gap-1.5">
        <span class="text-sm font-medium text-text">URL</span>
        <Text size="sm" muted class="break-all">{{ form.url }}</Text>
      </div>
    </template>

    <Field
      v-if="!isEditing"
      label="Account"
      required
      :error="errors.accountId"
      v-slot="{ id, describedBy, invalid }"
    >
      <Select
        :id="id"
        v-model="form.accountId"
        :items="accountOptions"
        placeholder="Select an account…"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field v-if="!isEditing" label="URL" required :error="errors.url" v-slot="{ id, describedBy, invalid }">
      <Input
        :id="id"
        v-model="form.url"
        type="url"
        inputmode="url"
        placeholder="https://gitlab.com/group/project"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field v-if="!isEditing" label="Name" required :error="errors.name" v-slot="{ id, describedBy, invalid }">
      <Input :id="id" v-model="form.name" autocomplete="off" :aria-describedby="describedBy" :aria-invalid="invalid" />
    </Field>

    <Field
      v-if="isEditing"
      label="Account"
      required
      :error="errors.accountId"
      v-slot="{ id, describedBy, invalid }"
    >
      <Select
        :id="id"
        v-model="form.accountId"
        :items="accountOptions"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field label="Provider" v-slot="{ id, describedBy }">
      <Select :id="id" v-model="form.providerId" :items="providerOptions" :aria-describedby="describedBy" />
    </Field>

    <Field label="Model" description="Optional — overrides the provider's default model." v-slot="{ id, describedBy }">
      <Input :id="id" v-model="form.model" placeholder="gpt-4o" :aria-describedby="describedBy" />
    </Field>

    <Field label="Default voice profile" v-slot="{ id, describedBy }">
      <Select :id="id" v-model="form.profileId" :items="profileOptions" :aria-describedby="describedBy" />
    </Field>

    <Alert v-if="formError" status="danger">{{ formError }}</Alert>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add repository' }}</Button>
    </div>
  </form>
</template>
