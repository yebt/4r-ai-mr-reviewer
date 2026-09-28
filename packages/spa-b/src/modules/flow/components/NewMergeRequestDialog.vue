<script setup lang="ts">
/**
 * "New MR" dialog — opened from the MRs tab's `#header-actions` (next to
 * "Release to main"), always visible so it works with 0 open MRs too. Owns
 * its own `DialogRoot` (mirrors `ReleaseDialog`/`ReviewLaunchDialog`'s
 * shape), controlled via `open`/`repo`/`update:open`.
 *
 * Ported from the old app's `packages/spa/src/modules/reviews/components/
 * CreateMergeRequestModal.vue` (behavior source of truth — branch pickers,
 * voice profile, provider/model overrides, "Generate with AI", editable
 * draft, dirty-close guard), NOT its styling. Pure helpers (default target
 * branch/profile, form validity, dirty-draft detection) live in
 * `../newMergeRequest.ts`; provider/model defaults are reused from
 * `../reviewLaunch.ts` instead of being reimplemented here.
 *
 * Two-step flow: `generateMergeRequest` (read-only, LLM-only) drafts a
 * title+description from the branch diff into editable fields; nothing
 * reaches GitLab until `createMergeRequest`, which really opens the merge
 * request — see the outward-effects note near the submit button. Regenerating
 * over an already-edited draft, and closing with unsaved edits, both go
 * through a non-danger confirm (own controlled `AlertDialogRoot`s below —
 * `ConfirmDialog` isn't reusable here since its trigger is a rendered slot,
 * not something opened programmatically).
 */
import { computed, reactive, ref, watch } from 'vue'
import { useQueryCache } from '@pinia/colada'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { Alert, Button, Combobox, Field, Icon, Input, Select, Textarea } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { createMergeRequest, generateMergeRequest, listRepoBranches } from '@modules/repos/api'
import type { Repo } from '@modules/repos/types'
import { useProfilesStore } from '@modules/profiles/store'
import { useProvidersStore } from '@modules/providers/store'
import { resolveDefaultModel, resolveDefaultProviderId } from '../reviewLaunch'
import {
  hasDraftText,
  isDraftDirty,
  isMergeRequestFormValid,
  isSameBranch,
  readyVoiceProfiles,
  resolveDefaultProfileId,
  resolveDefaultTargetBranch,
} from '../newMergeRequest'
import { repoMergeRequestsQueryKey } from '../mergeRequests'

const props = defineProps<{
  open: boolean
  repo: Repo
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const NO_PROFILE_VALUE = '__no_voice_profile__'
const NO_MODEL_VALUE = '__use_provider_default_model__'

const toast = useToast()
const queryCache = useQueryCache()
const profilesStore = useProfilesStore()
const providersStore = useProvidersStore()

const form = reactive({
  sourceBranch: '',
  targetBranch: '',
  profileId: '',
  providerId: '',
  model: '',
  title: '',
  description: '',
})

// --- branches (source/target Combobox pickers) ---

const branches = ref<string[]>([])
const branchesLoaded = ref(false)
const branchesError = ref<string | null>(null)

async function loadBranches() {
  branchesLoaded.value = false
  branchesError.value = null
  try {
    branches.value = await listRepoBranches(props.repo.id)
    branchesLoaded.value = true
    if (!form.targetBranch) form.targetBranch = resolveDefaultTargetBranch(branches.value)
  } catch (err) {
    branchesError.value = resolveErrorMessage(err, 'Failed to load branches')
  }
}

const branchItems = computed<SelectItemOption[]>(() => branches.value.map((b) => ({ label: b, value: b })))

// --- voice profile ---

const voiceProfiles = computed(() => readyVoiceProfiles(profilesStore.profiles))
const profileOptions = computed<SelectItemOption[]>(() => [
  { label: 'No voice profile', value: NO_PROFILE_VALUE },
  ...voiceProfiles.value.map((profile) => ({ label: profile.name, value: profile.id })),
])
const profileSelectValue = computed<string>({
  get: () => form.profileId || NO_PROFILE_VALUE,
  set: (value) => {
    form.profileId = value === NO_PROFILE_VALUE ? '' : value
  },
})

// --- provider + model (same precedence/UI split as ReviewLaunchDialog) ---

const hasProviders = computed(() => providersStore.providers.length > 0)
const providerOptions = computed<SelectItemOption[]>(() =>
  providersStore.providers.map((provider) => ({ label: provider.name, value: provider.id })),
)
const selectedProviderModels = computed<string[]>(
  () => providersStore.providers.find((provider) => provider.id === form.providerId)?.models ?? [],
)
const hasProviderModels = computed(() => selectedProviderModels.value.length > 0)
const modelOptions = computed<SelectItemOption[]>(() => [
  { label: 'Provider default', value: NO_MODEL_VALUE },
  ...selectedProviderModels.value.map((model) => ({ label: model, value: model })),
])
const modelSelectValue = computed<string>({
  get: () => (form.model ? form.model : NO_MODEL_VALUE),
  set: (value) => {
    form.model = value === NO_MODEL_VALUE ? '' : value
  },
})

function onProviderChange(id: string) {
  form.providerId = id
  form.model = resolveDefaultModel(props.repo, id, providersStore.providers)
}

// --- generate (read-only draft) ---

const generating = ref(false)
const generateError = ref<string | null>(null)
const overwriteConfirmOpen = ref(false)

const sameBranch = computed(() => isSameBranch(form))
const canGenerate = computed(
  () => !!form.sourceBranch.trim() && !!form.targetBranch.trim() && !sameBranch.value && !generating.value,
)

function handleGenerateClick() {
  if (!canGenerate.value) return
  if (hasDraftText(form)) {
    overwriteConfirmOpen.value = true
    return
  }
  void runGenerate()
}

function confirmOverwrite() {
  overwriteConfirmOpen.value = false
  void runGenerate()
}

async function runGenerate() {
  generating.value = true
  generateError.value = null
  try {
    const draft = await generateMergeRequest(props.repo.id, {
      sourceBranch: form.sourceBranch.trim(),
      targetBranch: form.targetBranch.trim(),
      ...(form.profileId ? { profileId: form.profileId } : {}),
      ...(form.providerId ? { providerId: form.providerId } : {}),
      ...(form.model ? { model: form.model } : {}),
    })
    form.title = draft.title
    form.description = draft.description
  } catch (err) {
    generateError.value = resolveErrorMessage(err, 'Failed to generate a draft')
  } finally {
    generating.value = false
  }
}

// --- create (real GitLab write) ---

const submitting = ref(false)
const submitError = ref<string | null>(null)

const canSubmit = computed(() => isMergeRequestFormValid(form) && !submitting.value)

async function handleSubmit() {
  if (!canSubmit.value) return
  submitting.value = true
  submitError.value = null
  try {
    const mr = await createMergeRequest(props.repo.id, {
      sourceBranch: form.sourceBranch.trim(),
      targetBranch: form.targetBranch.trim(),
      title: form.title.trim(),
      description: form.description.trim(),
    })
    emit('update:open', false)
    toast.success(`Merge request !${mr.iid} created`)
    await queryCache.invalidateQueries({ key: repoMergeRequestsQueryKey(props.repo.id) })
  } catch (err) {
    submitError.value = resolveErrorMessage(err, 'Failed to create the merge request')
  } finally {
    submitting.value = false
  }
}

// --- dirty-close guard ---

const isDirty = computed(() => generating.value || submitting.value || isDraftDirty(form))
const discardConfirmOpen = ref(false)

function requestClose() {
  if (isDirty.value) {
    discardConfirmOpen.value = true
    return
  }
  emit('update:open', false)
}

function confirmDiscard() {
  discardConfirmOpen.value = false
  emit('update:open', false)
}

function onDialogOpenUpdate(value: boolean) {
  if (value) {
    emit('update:open', true)
    return
  }
  requestClose()
}

// Resets local, dialog-scoped state whenever the dialog opens (or targets a
// different repo), so reopening never leaks a previous draft or error into
// the next one — mirrors ReleaseDialog/ReviewLaunchDialog's reset watcher.
watch(
  () => [props.open, props.repo.id] as const,
  ([open]) => {
    if (!open) return
    submitError.value = null
    generateError.value = null
    discardConfirmOpen.value = false
    overwriteConfirmOpen.value = false
    form.sourceBranch = ''
    form.targetBranch = ''
    form.profileId = resolveDefaultProfileId(props.repo, profilesStore.profiles)
    form.providerId = resolveDefaultProviderId(props.repo, providersStore.providers)
    form.model = resolveDefaultModel(props.repo, form.providerId, providersStore.providers)
    form.title = ''
    form.description = ''
    branches.value = []
    branchesLoaded.value = false
    branchesError.value = null
    void loadBranches()
  },
  { immediate: true },
)
</script>

<template>
  <DialogRoot :open="open" @update:open="onDialogOpenUpdate">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold">New merge request</DialogTitle>
        <DialogDescription class="mt-1 text-sm text-text-muted">
          Opens a merge request on {{ repo.name }} — pick the branches, optionally draft the title and
          description with AI, then review and edit before creating it.
        </DialogDescription>

        <form class="mt-4 flex flex-col gap-4" @submit.prevent="handleSubmit">
          <Alert v-if="branchesError" status="warning">
            Couldn't load branches ({{ branchesError }}). Type the branch names below.
          </Alert>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Source branch" required v-slot="{ id, describedBy }">
              <Combobox
                v-if="!branchesError"
                :id="id"
                v-model="form.sourceBranch"
                :items="branchItems"
                :disabled="!branchesLoaded"
                :placeholder="branchesLoaded ? 'Select branch…' : 'Loading branches…'"
                :aria-describedby="describedBy"
              />
              <Input v-else :id="id" v-model="form.sourceBranch" placeholder="feature/x" :aria-describedby="describedBy" />
            </Field>
            <Field label="Target branch" required v-slot="{ id, describedBy }">
              <Combobox
                v-if="!branchesError"
                :id="id"
                v-model="form.targetBranch"
                :items="branchItems"
                :disabled="!branchesLoaded"
                :placeholder="branchesLoaded ? 'Select branch…' : 'Loading branches…'"
                :aria-describedby="describedBy"
              />
              <Input v-else :id="id" v-model="form.targetBranch" placeholder="development" :aria-describedby="describedBy" />
            </Field>
          </div>
          <p v-if="sameBranch" class="text-xs text-danger-text">Source and target must be different branches.</p>

          <Field
            label="Voice profile"
            description="Optional — drafts the description in this author's voice."
            v-slot="{ id, describedBy }"
          >
            <Select
              :id="id"
              v-model="profileSelectValue"
              :items="profileOptions"
              :aria-describedby="describedBy"
            />
          </Field>

          <Alert v-if="!hasProviders" status="info">
            No providers configured yet — generating will use the account's default settings once one is added.
          </Alert>
          <div v-else class="flex flex-wrap items-end gap-3">
            <Field label="Provider" class="min-w-0 flex-1" v-slot="{ id, describedBy }">
              <Select
                :id="id"
                :model-value="form.providerId"
                :items="providerOptions"
                placeholder="Select a provider…"
                :aria-describedby="describedBy"
                @update:model-value="(value) => onProviderChange(value ?? '')"
              />
            </Field>
            <Field label="Model" class="min-w-0 flex-1" v-slot="{ id, describedBy }">
              <Select
                v-if="hasProviderModels"
                :id="id"
                v-model="modelSelectValue"
                :items="modelOptions"
                :aria-describedby="describedBy"
              />
              <Input
                v-else
                :id="id"
                v-model="form.model"
                placeholder="Provider default"
                autocomplete="off"
                :aria-describedby="describedBy"
              />
            </Field>
          </div>

          <div>
            <Button type="button" variant="outline" size="sm" :loading="generating" :disabled="!canGenerate" @click="handleGenerateClick">
              <template #leading><Icon name="sparkles" size="sm" /></template>
              {{ generating ? 'Drafting…' : 'Generate with AI' }}
            </Button>
          </div>
          <Alert v-if="generateError" status="danger">{{ generateError }}</Alert>

          <Field label="Title" required v-slot="{ id, describedBy }">
            <Input
              :id="id"
              v-model="form.title"
              placeholder="A concise, imperative summary"
              autocomplete="off"
              :aria-describedby="describedBy"
            />
          </Field>

          <Field label="Description" description="Markdown" v-slot="{ id, describedBy }">
            <Textarea
              :id="id"
              v-model="form.description"
              :rows="8"
              placeholder="Describe the change, or generate it from the diff above."
              class="font-mono text-xs"
              :aria-describedby="describedBy"
            />
          </Field>

          <Alert status="info">Opens the merge request on GitLab.</Alert>

          <Alert v-if="submitError" status="danger">{{ submitError }}</Alert>

          <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
            <Button type="button" variant="ghost" @click="requestClose">Cancel</Button>
            <Button type="submit" :loading="submitting" :disabled="!canSubmit">Create merge request</Button>
          </div>
        </form>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>

  <!-- Regenerating over an already-edited draft would silently discard it. -->
  <AlertDialogRoot :open="overwriteConfirmOpen" @update:open="(value) => (overwriteConfirmOpen = value)">
    <AlertDialogPortal>
      <AlertDialogOverlay class="overlay z-40" />
      <AlertDialogContent
        class="fixed left-1/2 top-1/2 z-50 w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <AlertDialogTitle class="text-md font-semibold text-text">Overwrite current draft?</AlertDialogTitle>
        <AlertDialogDescription class="mt-1 text-sm text-text-muted">
          Generating a new draft replaces the current title and description.
        </AlertDialogDescription>
        <div class="mt-5 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="ghost" size="sm">Keep current draft</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button variant="accent" size="sm" @click="confirmOverwrite">Generate</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>

  <!-- Closing (Escape/overlay/Cancel) with unsaved edits would silently discard them. -->
  <AlertDialogRoot :open="discardConfirmOpen" @update:open="(value) => (discardConfirmOpen = value)">
    <AlertDialogPortal>
      <AlertDialogOverlay class="overlay z-40" />
      <AlertDialogContent
        class="fixed left-1/2 top-1/2 z-50 w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <AlertDialogTitle class="text-md font-semibold text-text">Discard this draft?</AlertDialogTitle>
        <AlertDialogDescription class="mt-1 text-sm text-text-muted">
          Your branch, title, and description edits will be lost.
        </AlertDialogDescription>
        <div class="mt-5 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="ghost" size="sm">Keep editing</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button variant="accent" size="sm" @click="confirmDiscard">Discard</Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
