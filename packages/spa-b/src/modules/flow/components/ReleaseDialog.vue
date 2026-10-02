<script setup lang="ts">
/**
 * Release dialog — one component drives both release flows (`flow` prop),
 * opened either per-row from `MergeRequestListSection` (`flow="dev"`, an
 * existing MR targeting `development`) or from the MRs tab's header
 * (`flow="main"`, no MR — the routine creates the development→main MR
 * itself). Owns its own `DialogRoot`, controlled via `open`/`update:open`
 * (mirrors `ReviewLaunchDialog`'s shape).
 *
 * Ported from the old app's `packages/spa/src/modules/routines/components/
 * ReleaseModal.vue` (behavior source of truth — bump, live tag preview,
 * emojis, includeDev, branch pickers, merge options, per-flow defaults),
 * NOT its UnoCSS styling. Pure helpers (defaults/payload-building/emoji
 * rules) live in `../release.ts` so this component stays view-only.
 *
 * Unlike launching a review, submitting this dialog makes REAL GitLab writes
 * immediately (reactions + approval, and for the main flow an actual MR) —
 * see the outward-effects `Alert` above the submit button, and the dev
 * flow's `verify → react → approve → compute_tag → confirm → ...` / main
 * flow's `compute_tag → create_mr → wait_pipeline → approve → react →
 * confirm → ...` step ledgers in `server/internal/app/routines/service.go`.
 * Merging and tagging themselves stay behind the run page's confirm gate
 * (`src/pages/runs/[id].vue`).
 */
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useQueryCache } from '@pinia/colada'
import { watchDebounced } from '@vueuse/core'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Alert, Button, Checkbox, Combobox, Field, Input, Select } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { listRepoBranches } from '@modules/repos/api'
import type { MergeRequest, Repo } from '@modules/repos/types'
import { createMainRelease, createRelease, previewRoutineTag, repoRoutinesQueryKey } from '@modules/runs/api'
import { useRunsStore } from '@modules/runs/store'
import type { PreviewTagResult } from '@modules/runs/types'
import {
  EMOJI_CHOICES,
  MAX_RELEASE_EMOJIS,
  addEmoji,
  buildDevReleasePayload,
  buildMainReleasePayload,
  customEmojis,
  defaultReleaseFormState,
  isValidEmojiName,
  mainReleaseBranchProblem,
  missingConventionalBranches,
  normalizeCustomEmojiName,
  resolveDefaultBranches,
  toggleEmoji,
  type ReleaseFlow,
} from '../release'

const props = defineProps<{
  open: boolean
  flow: ReleaseFlow
  repo: Repo
  /** Dev flow only — the MR being released. */
  mergeRequest?: MergeRequest | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const router = useRouter()
const toast = useToast()
const queryCache = useQueryCache()
const runsStore = useRunsStore()

const form = reactive(defaultReleaseFormState(props.flow))

const bumpHint = computed(() => {
  switch (form.bump) {
    case 'major':
      return 'Major — resets to the next whole version (e.g. 0.86.1 → 1.0.0), regardless of commits.'
    case 'patch':
      return 'Patch — raises the patch number by the number of fix commits (feats ignored).'
    default:
      return 'Minor — raises the minor number per feat commit, the patch per trailing fix.'
  }
})

const includeDevHint = computed(() =>
  form.includeDev
    ? 'Bases on the highest tag including -dev, promoting the latest dev version (the suffix is dropped), then applies the bump.'
    : 'Bases on the highest pure X.Y.Z release; -dev tags are ignored.',
)

const mergeWhenPipelineSucceedsHint = computed(() =>
  props.flow === 'dev'
    ? 'Merges automatically once GitLab reports a green pipeline, instead of merging immediately at confirm.'
    : "The main flow already waits for a green pipeline before your confirmation, so merging the pinned head immediately is safer than scheduling another async merge.",
)

// --- main-flow branches (source/target Combobox pickers) ---

const branches = ref<string[]>([])
const branchesLoaded = ref(false)
const branchesError = ref<string | null>(null)

async function loadBranches() {
  if (props.flow !== 'main') return
  branchesLoaded.value = false
  branchesError.value = null
  try {
    branches.value = await listRepoBranches(props.repo.id)
    branchesLoaded.value = true
    const defaults = resolveDefaultBranches(branches.value)
    if (!form.sourceBranch) form.sourceBranch = defaults.source
    if (!form.targetBranch) form.targetBranch = defaults.target
  } catch (err) {
    branchesError.value = resolveErrorMessage(err, 'Failed to load branches')
  }
}

const branchItems = computed<SelectItemOption[]>(() => branches.value.map((b) => ({ label: b, value: b })))
const missingBranches = computed(() =>
  branchesLoaded.value ? missingConventionalBranches(branches.value) : [],
)

// --- emojis ---

const customEmojiInput = ref('')
const customEmojiError = ref<string | null>(null)

// `toggleEmoji`/`addEmoji` already refuse to grow the selection past this,
// but disabling the unselected chips + the custom-name input/button at the
// cap makes that limit visible instead of a click silently doing nothing.
const atEmojiCap = computed(() => form.emojis.length >= MAX_RELEASE_EMOJIS)

function handleAddCustomEmoji() {
  const name = normalizeCustomEmojiName(customEmojiInput.value)
  if (!name) {
    customEmojiInput.value = ''
    return
  }
  if (!isValidEmojiName(name)) {
    customEmojiError.value = 'Emoji names use lowercase letters, numbers, "_", "+" or "-" only.'
    return
  }
  customEmojiError.value = null
  form.emojis = addEmoji(form.emojis, name)
  customEmojiInput.value = ''
}

// --- live tag preview (debounced, stale-response-guarded) ---

const preview = ref<PreviewTagResult | null>(null)
const previewLoading = ref(false)
const previewError = ref<string | null>(null)
// Bumped on every preview request; a response is only applied when it's
// still the most recent one in flight, so a slow earlier request can never
// clobber a faster later one (stale-response protection).
let previewRequestId = 0

async function loadPreview() {
  if (!props.open) return
  if (props.flow === 'dev' && props.mergeRequest?.iid == null) {
    preview.value = null
    return
  }
  // Main flow: wait until the branch defaults are resolved; otherwise the first
  // preview fires with no branches and is immediately superseded (a wasted
  // GitLab round-trip, since preview-tag reads tags/commits live).
  if (props.flow === 'main' && (!form.sourceBranch.trim() || !form.targetBranch.trim())) return
  const requestId = ++previewRequestId
  previewLoading.value = true
  previewError.value = null
  try {
    const result = await previewRoutineTag(props.repo.id, {
      flow: props.flow,
      bump: form.bump,
      mrIid: props.flow === 'dev' ? props.mergeRequest?.iid : undefined,
      source: props.flow === 'main' ? form.sourceBranch.trim() : undefined,
      target: props.flow === 'main' ? form.targetBranch.trim() : undefined,
      includeDev: props.flow === 'main' ? form.includeDev : undefined,
    })
    if (requestId !== previewRequestId) return
    preview.value = result
  } catch (err) {
    if (requestId !== previewRequestId) return
    // Preview failing must never block the release — just show the error.
    preview.value = null
    previewError.value = resolveErrorMessage(err, 'Failed to preview the release tag')
  } finally {
    if (requestId === previewRequestId) previewLoading.value = false
  }
}

watchDebounced(
  () => [
    props.open,
    props.flow,
    form.bump,
    form.includeDev,
    form.sourceBranch,
    form.targetBranch,
    props.mergeRequest?.iid,
  ] as const,
  loadPreview,
  { debounce: 350, immediate: true },
)

// --- reset + submit ---

const submitting = ref(false)
const submitError = ref<string | null>(null)

// Watch each value as its own source: a getter returning a fresh array would
// re-fire whenever the repo/MR list refetches (e.g. on window focus) and hands
// down a new object with the same id, wiping the dialog's in-progress state.
watch(
  [() => props.open, () => props.flow, () => props.repo.id, () => props.mergeRequest?.iid],
  ([open]) => {
    if (!open) return
    submitError.value = null
    customEmojiInput.value = ''
    customEmojiError.value = null
    preview.value = null
    previewError.value = null
    Object.assign(form, defaultReleaseFormState(props.flow))
    branches.value = []
    branchesLoaded.value = false
    branchesError.value = null
    void loadBranches()
  },
  { immediate: true },
)

// Main flow: both branches must be picked explicitly — a blank one would make
// the backend fall back to development/main, which may not exist on this repo.
const branchProblem = computed(() => (props.flow === 'main' ? mainReleaseBranchProblem(form) : null))
const canSubmit = computed(() => !(props.flow === 'dev' && !props.mergeRequest) && !branchProblem.value)

async function handleSubmit() {
  if (submitting.value) return
  if (!canSubmit.value) return
  submitting.value = true
  submitError.value = null
  try {
    const run =
      props.flow === 'dev'
        ? await createRelease(props.repo.id, buildDevReleasePayload(form, props.mergeRequest!.iid))
        : await createMainRelease(props.repo.id, buildMainReleasePayload(form))
    emit('update:open', false)
    toast.success(props.flow === 'dev' ? 'Release started' : 'Release to main started')
    await queryCache.invalidateQueries({ key: repoRoutinesQueryKey(props.repo.id) })
    await runsStore.refetch()
    router.push(`/runs/${run.id}`)
  } catch (err) {
    submitError.value = resolveErrorMessage(err, 'Failed to start the release')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(30rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold">{{ flow === 'main' ? 'Release to main' : 'Release' }}</DialogTitle>
        <DialogDescription v-if="flow === 'dev' && mergeRequest" class="mt-1 text-sm text-text-muted">
          <span class="font-mono">!{{ mergeRequest.iid }}</span>
          {{ mergeRequest.title }}
          <span class="mt-0.5 block truncate font-mono text-xs">
            {{ mergeRequest.sourceBranch }} → {{ mergeRequest.targetBranch }}
          </span>
        </DialogDescription>
        <DialogDescription v-else class="mt-1 text-sm text-text-muted">
          <template v-if="!branchProblem">
            Cuts a release from <span class="font-mono text-text">{{ form.sourceBranch.trim() }}</span> into
            <span class="font-mono text-text">{{ form.targetBranch.trim() }}</span>. Change the branches below if
            needed.
          </template>
          <template v-else>Cuts a release from one branch into another. Pick both branches below.</template>
        </DialogDescription>

        <form class="mt-4 flex flex-col gap-4" @submit.prevent="handleSubmit">
          <Field label="Version bump" :description="bumpHint" v-slot="{ id, describedBy }">
            <Select
              :id="id"
              v-model="form.bump"
              :items="[
                { label: 'Major', value: 'major' },
                { label: 'Minor', value: 'minor' },
                { label: 'Patch', value: 'patch' },
              ]"
              :aria-describedby="describedBy"
            />
          </Field>

          <!-- Live tag preview — the exact tag this release will create. -->
          <div class="rounded-md border border-line-subtle bg-bg-hover px-3 py-2.5" aria-live="polite">
            <div class="mb-1 flex items-center gap-1.5 font-mono text-xs text-text-muted">
              Next tag
              <span
                v-if="previewLoading"
                class="size-2.5 animate-spin rounded-full border border-text-muted border-t-transparent"
                aria-hidden="true"
              />
            </div>
            <div v-if="previewLoading && !preview" class="text-sm text-text-muted">Computing…</div>
            <Alert v-else-if="previewError" status="warning">
              Couldn't preview the tag ({{ previewError }}). It's still computed and shown for approval at the
              confirmation gate.
            </Alert>
            <template v-else-if="preview">
              <div class="transition-opacity" :class="previewLoading ? 'opacity-40' : ''">
                <div class="font-mono text-lg font-semibold text-text">{{ preview.nextTag }}</div>
                <div class="mt-0.5 text-xs text-text-muted">
                  from <span class="font-mono">{{ preview.lastTag || 'no previous tag' }}</span> ·
                  {{ preview.featCount }} feat, {{ preview.fixCount }} fix
                </div>
              </div>
            </template>
            <div v-else class="text-xs text-text-muted">
              {{
                flow === 'main'
                  ? 'Pick the source and target branches to preview the next tag.'
                  : 'Pick a merge request to preview its release tag.'
              }}
            </div>
          </div>

          <template v-if="flow === 'main'">
            <label class="flex cursor-pointer items-start gap-2 text-sm">
              <Checkbox v-model="form.includeDev" class="mt-0.5" />
              <span>
                <span class="text-text">Count <span class="font-mono">-dev</span> prereleases</span>
                <span class="mt-0.5 block text-xs text-text-muted">{{ includeDevHint }}</span>
              </span>
            </label>

            <Alert v-if="missingBranches.length > 0" status="warning">
              <span class="font-medium">
                {{ missingBranches.map((b) => `"${b}"`).join(' and ') }}
                {{ missingBranches.length === 1 ? 'branch' : 'branches' }} not found
              </span>
              <span class="mt-1 block">
                This repo has no {{ missingBranches.join('/') }} branch — pick the branches to release from and
                into below.
              </span>
            </Alert>
            <Alert v-else-if="branchesError" status="warning">
              Couldn't load branches ({{ branchesError }}). Type the branch names below.
            </Alert>

            <Field label="Source branch" v-slot="{ id, describedBy }">
              <Combobox
                v-if="!branchesError"
                :id="id"
                v-model="form.sourceBranch"
                :items="branchItems"
                :disabled="!branchesLoaded"
                :placeholder="branchesLoaded ? 'development' : 'Loading branches…'"
                :aria-describedby="describedBy"
              />
              <Input v-else v-model="form.sourceBranch" placeholder="development" :aria-describedby="describedBy" />
            </Field>
            <Field label="Target branch" v-slot="{ id, describedBy }">
              <Combobox
                v-if="!branchesError"
                :id="id"
                v-model="form.targetBranch"
                :items="branchItems"
                :disabled="!branchesLoaded"
                :placeholder="branchesLoaded ? 'main' : 'Loading branches…'"
                :aria-describedby="describedBy"
              />
              <Input v-else v-model="form.targetBranch" placeholder="main" :aria-describedby="describedBy" />
            </Field>
          </template>

          <div class="flex flex-col gap-1.5">
            <span class="text-sm font-medium text-text">Reaction emojis</span>
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="choice in EMOJI_CHOICES"
                :key="choice.name"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border px-2 py-1 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                :class="
                  form.emojis.includes(choice.name)
                    ? 'border-accent bg-accent-subtle-bg text-accent-text-strong'
                    : 'border-line text-text-muted hover:text-text'
                "
                :aria-pressed="form.emojis.includes(choice.name)"
                :disabled="atEmojiCap && !form.emojis.includes(choice.name)"
                @click="form.emojis = toggleEmoji(form.emojis, choice.name)"
              >
                <span aria-hidden="true">{{ choice.glyph }}</span>
                <span class="font-mono">{{ choice.name }}</span>
              </button>
              <button
                v-for="name in customEmojis(form.emojis)"
                :key="name"
                type="button"
                class="inline-flex items-center gap-1.5 rounded-md border border-accent bg-accent-subtle-bg px-2 py-1 font-mono text-xs text-accent-text-strong"
                :aria-label="`Remove ${name}`"
                @click="form.emojis = toggleEmoji(form.emojis, name)"
              >
                {{ name }}
              </button>
            </div>
            <p class="text-xs text-text-muted">Up to {{ MAX_RELEASE_EMOJIS }} reactions</p>
            <div class="flex items-center gap-2">
              <Input
                v-model="customEmojiInput"
                placeholder="add another emoji name…"
                autocomplete="off"
                aria-label="Custom emoji name"
                :disabled="atEmojiCap"
                @keydown.enter.prevent="handleAddCustomEmoji"
              />
              <Button type="button" variant="outline" size="sm" :disabled="atEmojiCap" @click="handleAddCustomEmoji">
                Add
              </Button>
            </div>
            <p v-if="customEmojiError" class="text-xs text-danger-text">{{ customEmojiError }}</p>
          </div>

          <!-- Main flow's source is the long-lived development branch: never offer to delete it. -->
          <label v-if="flow !== 'main'" class="flex cursor-pointer items-center gap-2 text-sm">
            <Checkbox v-model="form.removeSourceBranch" />
            <span class="text-text">Remove source branch after merge</span>
          </label>
          <label class="flex cursor-pointer items-start gap-2 text-sm">
            <Checkbox v-model="form.mergeWhenPipelineSucceeds" class="mt-0.5" />
            <span>
              <span class="text-text">Merge when pipeline succeeds</span>
              <span class="mt-0.5 block text-xs text-text-muted">{{ mergeWhenPipelineSucceedsHint }}</span>
            </span>
          </label>

          <Alert status="info">
            <template v-if="flow === 'dev' && mergeRequest">
              Starting adds the selected reactions and approves
              <span class="font-mono">!{{ mergeRequest.iid }}</span> on GitLab now; merging and tagging wait for
              your confirmation on the run page.
            </template>
            <template v-else-if="branchProblem">
              {{ branchProblem }} Starting opens a merge request between them on GitLab, waits for its pipeline,
              then approves it and adds the selected reactions; merging and tagging wait for your confirmation.
            </template>
            <template v-else>
              Starting opens a merge request
              <span class="font-mono">{{ form.sourceBranch.trim() }} → {{ form.targetBranch.trim() }}</span>
              on GitLab now, waits for its pipeline, then approves it and adds the selected reactions — all before
              your confirmation on the run page; merging and tagging wait there too.
            </template>
          </Alert>

          <Alert v-if="submitError" status="danger">{{ submitError }}</Alert>

          <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
            <Button type="button" variant="ghost" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" :loading="submitting" :disabled="!canSubmit">
              {{ flow === 'main' ? 'Start release to main' : 'Start release' }}
            </Button>
          </div>
        </form>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
