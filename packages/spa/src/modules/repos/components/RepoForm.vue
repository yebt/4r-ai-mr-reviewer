<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import { api, errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { GitlabProject, Repo } from '@shared/api/types'
import DependencyAlert from '@shared/components/ui/DependencyAlert.vue'
import { useReposStore } from '@modules/repos/store'
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'
import { useProfilesStore } from '@modules/profiles/store'
import { matchAccountId, parseRepoUrl } from '@modules/repos/url'

const props = defineProps<{ editing?: Repo | null }>()
const emit = defineEmits<{ done: [] }>()

const repos = useReposStore()
const accounts = useAccountsStore()
const providers = useProvidersStore()
const profiles = useProfilesStore()

onMounted(() => {
  if (accounts.items.length === 0) accounts.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
  if (profiles.items.length === 0) profiles.fetchAll()
})

const blank = () => ({ name: '', url: '', accountId: '', providerId: '', model: '', profileId: '' })
const form = reactive(blank())
const submitting = ref(false)
const error = ref<string | null>(null)

const isEdit = computed(() => !!props.editing)

// Prerequisite guards: only after the store has settled (fetchAll flips `loading`
// synchronously on mount) so the warning never flashes during the initial load.
const noAccounts = computed(() => !accounts.loading && accounts.items.length === 0)
const noProviders = computed(() => !providers.loading && providers.items.length === 0)
const valid = computed(
  () => isEdit.value || (form.name.trim() && form.url.trim() && form.accountId),
)

// What's still needed to enable submit, shown beside the disabled button so the
// user never has to guess why it won't save.
const missing = computed(() => {
  if (isEdit.value) return [] as string[]
  const m: string[] = []
  if (!form.accountId) m.push('account')
  if (!form.url.trim()) m.push('project URL')
  if (!form.name.trim()) m.push('name')
  return m
})

// Derive fields from the pasted URL, keeping any manual edits.
const parsed = computed(() => parseRepoUrl(form.url))
const lastAutoName = ref('')
const matchedAccount = computed(() => accounts.items.find((a) => a.id === form.accountId) ?? null)

// Model suggestions come from the chosen provider (or the default one).
const selectedProvider = computed(() => {
  if (form.providerId) return providers.items.find((p) => p.id === form.providerId) ?? null
  return providers.items.find((p) => p.isDefault) ?? null
})
// Presets sorted alphabetically (case-insensitive), without mutating the store's array.
const modelPresets = computed(() =>
  [...(selectedProvider.value?.models ?? [])].sort((a, b) =>
    a.toLowerCase().localeCompare(b.toLowerCase()),
  ),
)

// --- GitLab project picker (fzf-style) ---
// When an account is selected the user searches the projects that account's
// token can see and picks one, auto-filling the URL (and name). The manual URL
// field below stays as a fallback.
const projectQuery = ref('')
const projects = ref<GitlabProject[]>([])
const searching = ref(false)
const searchError = ref<string | null>(null)
// Whether the results dropdown is open (focus opens it; picking closes it).
const pickerOpen = ref(false)
// Marks a URL that came from a picked project, so the "manually pasted" hint
// isn't shown for it.
const pickedFromSearch = ref(false)

async function runProjectSearch() {
  if (!form.accountId) return
  searching.value = true
  searchError.value = null
  try {
    projects.value = await api.searchAccountProjects(form.accountId, projectQuery.value.trim())
  } catch (e) {
    searchError.value = errorMessage(e)
    projects.value = []
  } finally {
    searching.value = false
  }
}

// Debounce keystrokes so we don't fire a request per character.
watchDebounced(projectQuery, runProjectSearch, { debounce: 300 })

// Switching accounts resets the picker and preloads the new account's most
// recently active projects.
watch(
  () => form.accountId,
  (id) => {
    projectQuery.value = ''
    projects.value = []
    searchError.value = null
    pickerOpen.value = false
    if (id) void runProjectSearch()
  },
)

function openPicker() {
  if (!form.accountId) return
  pickerOpen.value = true
  // Preload recent projects the first time the field is focused.
  if (projects.value.length === 0 && !searching.value && !searchError.value) {
    void runProjectSearch()
  }
}

function selectProject(p: GitlabProject) {
  form.url = p.webUrl
  pickedFromSearch.value = true
  // Only fill the name when the user hasn't typed their own (or only accepted a
  // previously auto-filled one) — never clobber a hand-typed name.
  if (form.name.trim() === '' || form.name === lastAutoName.value) {
    form.name = p.name
    lastAutoName.value = p.name
  }
  pickerOpen.value = false
}

// Resolve name + account from the URL when the field loses focus, keeping any
// manual edits the user already made.
function resolveFromUrl() {
  // A hand-edited URL is no longer the picked project's URL.
  pickedFromSearch.value = false
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
      form.profileId = r.defaultProfileId
    } else {
      Object.assign(form, blank())
      lastAutoName.value = ''
      projectQuery.value = ''
      projects.value = []
      searchError.value = null
      pickerOpen.value = false
      pickedFromSearch.value = false
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
      await repos.assign(props.editing.id, {
        providerId: form.providerId,
        model: form.model.trim(),
        accountId: form.accountId,
        profileId: form.profileId,
      })
      toast.success('Repository updated')
    } else {
      await repos.add({
        name: form.name.trim(),
        url: form.url.trim(),
        accountId: form.accountId,
        providerId: form.providerId,
        model: form.model.trim(),
        profileId: form.profileId,
      })
      toast.success('Repository added')
    }
    Object.assign(form, blank())
    lastAutoName.value = ''
    projectQuery.value = ''
    projects.value = []
    pickedFromSearch.value = false
    pickerOpen.value = false
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
      <!-- Repositories depend on a GitLab account; without one there is nothing to
           attach a repo to. -->
      <DependencyAlert
        v-if="noAccounts"
        message="No GitLab account connected — add one to add repositories."
        cta-label="Connect an account"
        cta-to="/accounts"
      />

      <div class="label-mono border-line/50 mb-1 border-b pb-2">GitLab connection</div>

      <div>
        <label class="field-label" for="rp-account">
          Account <span class="text-accent" aria-hidden="true">*</span>
        </label>
        <select
          id="rp-account"
          v-model="form.accountId"
          class="field-underline"
          aria-required="true"
        >
          <option value="" disabled>Select an account…</option>
          <option v-for="a in accounts.items" :key="a.id" :value="a.id">
            {{ a.name }} — {{ a.baseUrl }}
          </option>
        </select>
      </div>

      <!-- Project picker: search the projects the selected account's token can
           see and pick one to auto-fill the URL and name. -->
      <div>
        <label class="field-label" for="rp-project">
          Project
          <span class="text-muted normal-case">— search your account's projects</span>
        </label>
        <p v-if="!form.accountId" class="text-muted mt-1.5 text-xs">
          Select an account first to search its projects.
        </p>
        <template v-else>
          <input
            id="rp-project"
            v-model="projectQuery"
            class="field-underline"
            placeholder="Search projects…"
            autocomplete="off"
            role="combobox"
            aria-controls="rp-project-results"
            :aria-expanded="pickerOpen"
            @focus="openPicker"
          />
          <div
            v-if="pickerOpen"
            id="rp-project-results"
            class="border-line bg-surface mt-2 max-h-64 overflow-y-auto border"
          >
            <p v-if="searching" class="text-muted px-3 py-2 text-xs">Searching…</p>
            <p v-else-if="searchError" class="text-danger px-3 py-2 text-xs">{{ searchError }}</p>
            <p v-else-if="projects.length === 0" class="text-muted px-3 py-2 text-xs">
              No matches — refine your search or enter the URL manually below.
            </p>
            <ul v-else class="divide-line/40 divide-y">
              <li v-for="p in projects" :key="p.id">
                <button
                  type="button"
                  class="hover:bg-accent/10 flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left"
                  @click="selectProject(p)"
                >
                  <span class="text-ink font-mono text-sm">{{ p.pathWithNamespace }}</span>
                  <span class="text-muted text-xs">{{ p.name }}</span>
                </button>
              </li>
            </ul>
          </div>
        </template>
      </div>

      <div>
        <label class="field-label" for="rp-url">
          Project URL <span class="text-accent" aria-hidden="true">*</span>
          <span class="text-muted normal-case">— filled by the picker, or paste one</span>
        </label>
        <input
          id="rp-url"
          v-model="form.url"
          class="field-underline"
          placeholder="https://gitlab.com/group/project"
          autocomplete="off"
          aria-required="true"
          @blur="resolveFromUrl"
        />
        <p v-if="form.url && !parsed.valid" class="text-warn mt-1.5 text-xs">
          Enter a full project URL (https://host/group/project).
        </p>
        <p v-else-if="pickedFromSearch && parsed.valid" class="label-mono mt-1.5">
          {{ parsed.path }} · selected from search
        </p>
        <p v-else-if="parsed.valid" class="label-mono mt-1.5">
          {{ parsed.path }}
          <template v-if="matchedAccount"> · matched {{ matchedAccount.name }}</template>
          <template v-else-if="accounts.items.length"> · no account for this host</template>
        </p>
      </div>

      <div>
        <label class="field-label" for="rp-name">
          Name <span class="text-accent" aria-hidden="true">*</span>
          <span class="text-muted normal-case">— from the project, editable</span>
        </label>
        <input
          id="rp-name"
          v-model="form.name"
          class="field-underline"
          placeholder="project"
          autocomplete="off"
          aria-required="true"
        />
      </div>
    </template>

    <template v-else>
      <div class="text-muted font-mono text-xs">{{ form.name }} · {{ form.url }}</div>

      <div class="label-mono border-line/50 mb-1 border-b pb-2">GitLab connection</div>

      <div>
        <label class="field-label" for="rp-account-edit">
          Account <span class="text-muted normal-case">— reassign this repo to another account</span>
        </label>
        <select id="rp-account-edit" v-model="form.accountId" class="field-underline">
          <option v-for="a in accounts.items" :key="a.id" :value="a.id">
            {{ a.name }} — {{ a.baseUrl }}
          </option>
        </select>
      </div>
    </template>

    <!-- Reviews depend on an AI provider; a repo with no provider available can't
         be reviewed. -->
    <DependencyAlert
      v-if="noProviders"
      message="No AI provider configured — add one to run reviews."
      cta-label="Add a provider"
      cta-to="/providers"
    />

    <div class="label-mono border-line/50 mb-1 border-b pb-2">AI review</div>

    <div>
      <label class="field-label" for="rp-provider">
        Provider <span class="text-muted normal-case">— optional, uses your default</span>
      </label>
      <select id="rp-provider" v-model="form.providerId" class="field-underline">
        <option value="">Use default provider</option>
        <option v-for="p in providers.items" :key="p.id" :value="p.id">
          {{ p.name }}{{ p.isDefault ? ' (default)' : '' }}
        </option>
      </select>
    </div>

    <div>
      <label class="field-label" for="rp-model">
        Model <span class="text-muted normal-case">— optional, uses the provider's model</span>
      </label>
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
      <p v-if="modelPresets.length" class="text-muted/70 mt-1.5 text-xs">
        Presets: {{ modelPresets.join(', ') }}
      </p>
    </div>

    <div>
      <label class="field-label" for="rp-profile">
        Default review voice
        <span class="text-muted normal-case">— optional, pre-selected when humanizing reviews</span>
      </label>
      <select id="rp-profile" v-model="form.profileId" class="field-underline">
        <option value="">No default</option>
        <option v-for="p in profiles.items" :key="p.id" :value="p.id">
          {{ p.name }}
        </option>
      </select>
    </div>

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save' : 'Track repository' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
      <p v-if="!valid && missing.length" class="text-muted w-full text-xs sm:w-auto">
        Still needed: {{ missing.join(', ') }}.
      </p>
    </div>
  </form>
</template>
