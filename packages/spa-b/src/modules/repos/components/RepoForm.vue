<script setup lang="ts">
/**
 * Create/reassign form for a single repo. Used inside a Reka `Dialog` by
 * `RepositoriesSection`. Keyed by `repo?.id ?? 'create'` in the parent so
 * switching context always remounts with fresh local state.
 *
 * CREATE mode: Account (required Select) + Project (required search
 * combobox, see below) + Name (required Input) + Provider/Model/Profile
 * (all optional — "use default"). If there are no accounts yet, the form is
 * replaced by an info Alert pointing at /settings/accounts, since
 * `accountId` is required by the backend and there's nothing to pick.
 *
 * Project field: once an Account is selected, the field searches that
 * account's GitLab projects (`GET /accounts/{id}/projects?search=`, only
 * while the field is focused/non-empty — the same anti-lag, capped,
 * only-on-query shape as ProviderForm's OpenRouter model search) and shows
 * matches inline below it. The field itself is bound to `form.url` — it
 * doubles as the URL, so a manual paste is a fully valid fallback exactly
 * like the old spa's picker allowed; picking a result overwrites `form.url`
 * with the project's `webUrl` and pre-fills Name from the project's name,
 * but only while Name is still empty or holds a previous auto-fill (never
 * clobbers a hand-typed name).
 *
 * EDIT mode ("Reassign"): `name`/`url` are NOT editable after create (per
 * `PATCH /repos/{id}/assign`), so they render as read-only text; only
 * Account/Provider/Model/Profile are editable and submit `assignRepo`.
 *
 * Provider/Profile are optional-with-a-"use default"-choice, but Reka's
 * `Select` reserves the empty string for its own placeholder/unset state, so
 * the sentinel values below stand in for "" in the UI and get mapped back to
 * "" right before the payload is built.
 *
 * Model is free text by default, but when the selected provider (resolved
 * from `form.providerId` against `useProvidersStore().providers`) declares a
 * non-empty `models` list (e.g. an OpenRouter provider), it becomes a
 * `Select` over that list — picking an arbitrary model id wouldn't be valid
 * for that provider. `modelSelectValue` is the same "" -> sentinel proxy used
 * for Provider/Profile above, so `form.model` itself stays the single source
 * of truth (and the submit payload) whether the free-text Input or the
 * Select is rendered.
 */
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { Alert, Button, Field, Icon, Input, Select, Spinner, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
// Deep-imported (not through each module's barrel, which only re-exports
// its Section component + types, not the store) — read-only reuse, per the
// task boundary: this module never edits accounts/providers/profiles.
import { useAccountsStore } from '@modules/accounts/store'
import { useProfilesStore } from '@modules/profiles/store'
import { useProvidersStore } from '@modules/providers/store'
import { useReposStore } from '../store'
import { searchAccountProjects } from '../api'
import { projectToDraft } from '../projectSearch'
import type { AccountProject, Repo } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const NO_PROVIDER_VALUE = '__use_default_provider__'
const NO_PROFILE_VALUE = '__no_default_profile__'
const NO_MODEL_VALUE = '__use_default_model__'

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

// The selected provider's `models` list (empty when no provider is selected,
// the "use default" sentinel is selected, or the resolved provider declares
// no models) — drives whether the Model field renders as a Select or falls
// back to free text.
const selectedProviderModels = computed<string[]>(() => {
  if (form.providerId === NO_PROVIDER_VALUE) return []
  return providersStore.providers.find((provider) => provider.id === form.providerId)?.models ?? []
})
const hasProviderModels = computed(() => selectedProviderModels.value.length > 0)
const modelOptions = computed<SelectItemOption[]>(() => [
  { label: "Use provider's default model", value: NO_MODEL_VALUE },
  ...selectedProviderModels.value.map((model) => ({ label: model, value: model })),
])
// Proxies form.model (the actual payload value, "" meaning "use default")
// through the sentinel Select needs, without changing what handleSubmit reads.
const modelSelectValue = computed<string>({
  get: () => (form.model ? form.model : NO_MODEL_VALUE),
  set: (value) => {
    form.model = value === NO_MODEL_VALUE ? '' : value
  },
})

type FormErrors = Partial<Record<'accountId' | 'url' | 'name', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const saving = ref(false)
const formError = ref<string | null>(null)

// --- Project search (inline, only-on-query, focus-gated) -------------------
// The field is bound to `form.url` itself: typing searches the selected
// account's projects, and whatever ends up in `form.url` is what gets
// submitted, so a manual paste that matches nothing is still a valid URL.
// Results only render while the field is focused, so a stale "no matches"
// panel never lingers after a pick or after the user tabs away.
const projects = ref<AccountProject[]>([])
const searching = ref(false)
const searchError = ref<string | null>(null)
const resultsOpen = ref(false)
const pickedFromSearch = ref(false)
const lastAutoName = ref('')

async function runProjectSearch() {
  if (!resultsOpen.value || !form.accountId) return
  const query = form.url.trim()
  if (!query) {
    projects.value = []
    searchError.value = null
    return
  }
  searching.value = true
  searchError.value = null
  try {
    projects.value = await searchAccountProjects(form.accountId, query)
  } catch (err) {
    searchError.value = resolveErrorMessage(err, 'Failed to search projects')
    projects.value = []
  } finally {
    searching.value = false
  }
}

watchDebounced(() => form.url, runProjectSearch, { debounce: 300 })

// Switching accounts invalidates any in-flight/previous results — they
// belong to a different account's project catalog.
watch(
  () => form.accountId,
  () => {
    projects.value = []
    searchError.value = null
    resultsOpen.value = false
  },
)

function openResults() {
  if (!form.accountId) return
  resultsOpen.value = true
  void runProjectSearch()
}

function closeResults() {
  resultsOpen.value = false
}

function selectProject(project: AccountProject) {
  const draft = projectToDraft(project)
  form.url = draft.url
  pickedFromSearch.value = true
  // Only fill the name when the user hasn't typed their own (or only has a
  // previously auto-filled one) — never clobber a hand-typed name.
  if (form.name.trim() === '' || form.name === lastAutoName.value) {
    form.name = draft.name
    lastAutoName.value = draft.name
  }
  resultsOpen.value = false
  projects.value = []
}

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
    formError.value = resolveErrorMessage(err, 'Failed to save repository')
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
    <div class="dialog-footer">
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

    <Field
      v-if="!isEditing"
      label="Project"
      required
      :error="errors.url"
      description="Search your account's GitLab projects, or paste a project URL."
      v-slot="{ id, describedBy, invalid }"
    >
      <p v-if="!form.accountId" class="text-xs text-text-muted">
        Select an account first to search its projects.
      </p>
      <div v-else class="flex flex-col gap-1.5">
        <div
          class="flex h-8 items-center gap-2 rounded-md border bg-bg-panel px-2.5 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-focus-ring"
          :class="invalid ? 'border-danger-solid' : 'border-line'"
        >
          <Icon name="search" size="sm" class="shrink-0 text-text-muted" />
          <input
            :id="id"
            v-model="form.url"
            type="text"
            autocomplete="off"
            role="combobox"
            aria-controls="repo-project-results"
            :aria-expanded="resultsOpen"
            :aria-describedby="describedBy"
            :aria-invalid="invalid"
            placeholder="Search projects or paste a URL…"
            class="h-full min-w-0 flex-1 bg-transparent text-[1rem] text-text outline-none placeholder:text-text-placeholder md:text-sm"
            @focus="openResults"
            @blur="closeResults"
            @keydown.esc="closeResults"
          />
        </div>

        <div
          v-if="resultsOpen"
          id="repo-project-results"
          class="max-h-56 overflow-y-auto rounded-md border border-line bg-bg-panel-raised p-1"
        >
          <div v-if="searching" class="flex items-center gap-2 p-2">
            <Spinner size="sm" />
            <Text size="sm" muted>Searching projects…</Text>
          </div>
          <p v-else-if="searchError" class="p-2 text-xs text-danger-text">{{ searchError }}</p>
          <p v-else-if="!form.url.trim()" class="p-2 text-xs text-text-muted">
            Type to search your account's projects.
          </p>
          <p v-else-if="projects.length === 0" class="p-2 text-xs text-text-muted">
            No matches — you can still use this as a manual project URL.
          </p>
          <ul v-else class="flex flex-col gap-0.5">
            <li v-for="project in projects" :key="project.id">
              <button
                type="button"
                data-testid="project-result"
                class="flex w-full flex-col gap-0.5 rounded-md px-2.5 py-1.5 text-left text-sm text-text outline-none hover:bg-accent-subtle-bg hover:text-accent-text-strong focus-visible:bg-accent-subtle-bg focus-visible:text-accent-text-strong"
                @mousedown.prevent
                @click="selectProject(project)"
              >
                <span class="truncate font-mono text-xs">{{ project.pathWithNamespace }}</span>
                <span class="truncate text-text-muted">{{ project.name }}</span>
              </button>
            </li>
          </ul>
        </div>

        <p v-if="pickedFromSearch" class="text-xs text-text-muted">Selected from search.</p>
      </div>
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
      <Select
        v-if="hasProviderModels"
        :id="id"
        v-model="modelSelectValue"
        :items="modelOptions"
        :aria-describedby="describedBy"
      />
      <Input v-else :id="id" v-model="form.model" placeholder="gpt-4o" :aria-describedby="describedBy" />
    </Field>

    <Field label="Default voice profile" v-slot="{ id, describedBy }">
      <Select :id="id" v-model="form.profileId" :items="profileOptions" :aria-describedby="describedBy" />
    </Field>

    <Alert v-if="formError" status="danger">{{ formError }}</Alert>

    <div class="dialog-footer">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add repository' }}</Button>
    </div>
  </form>
</template>
