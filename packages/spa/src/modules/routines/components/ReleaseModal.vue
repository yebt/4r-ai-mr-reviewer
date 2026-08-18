<script lang="ts">
import type { MainReleaseInput, ReleaseInput } from '@shared/api/types'

// One modal drives both release flows: the `dev` flow releases a single merge
// request (target must be `development`); the `main` flow opens/merges a
// development → main release and exposes the source/target branch overrides.
// Exported from a normal <script> block because <script setup> cannot export.
export type ReleaseSubmit =
  | { flow: 'dev'; input: ReleaseInput }
  | { flow: 'main'; input: MainReleaseInput }
</script>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { watchDebounced } from '@vueuse/core'
import Modal from '@shared/components/ui/Modal.vue'
import { api, errorMessage } from '@shared/api/client'
import type { Bump, MergeRequest } from '@shared/api/types'

const props = defineProps<{
  open: boolean
  flow: 'dev' | 'main'
  repoId: string
  // Only set for the dev flow, to show which MR is being released.
  mergeRequest?: MergeRequest | null
  submitting?: boolean
}>()
const emit = defineEmits<{ submit: [payload: ReleaseSubmit]; close: [] }>()

const bump = ref<Bump>('minor')
// Main flow only. When true, the release bases on the highest tag INCLUDING
// "-dev" prereleases (promoting the latest dev version) instead of ignoring them.
const includeDev = ref(false)
const removeSourceBranch = ref(false)
const mergeWhenPipelineSucceeds = ref(true)
// Main flow only. Empty means "use the backend default" (development → main).
const sourceBranch = ref('')
const targetBranch = ref('')

// The repo's branches back the main-flow branch pickers. Routines assume
// development → main exist; when they don't, warn so the user picks alternatives.
const branches = ref<string[]>([])
const branchesLoaded = ref(false)
const branchesError = ref<string | null>(null)
async function loadBranches() {
  if (props.flow !== 'main') return
  branchesLoaded.value = false
  branchesError.value = null
  try {
    branches.value = await api.listRepoBranches(props.repoId)
    branchesLoaded.value = true
    if (!sourceBranch.value)
      sourceBranch.value = branches.value.includes('development') ? 'development' : ''
    if (!targetBranch.value) targetBranch.value = branches.value.includes('main') ? 'main' : ''
  } catch (e) {
    branchesError.value = errorMessage(e)
  }
}
// Which conventional branches the repo is missing (main flow, once loaded).
const missingConventional = computed(() => {
  if (props.flow !== 'main' || !branchesLoaded.value) return [] as string[]
  return ['development', 'main'].filter((b) => !branches.value.includes(b))
})
// Reaction emojis posted on the released MR. GitLab uses award-emoji NAMES (not
// unicode), so a curated quick-pick set shows the glyph for recognition while
// storing the name; a small input adds any name not on the list. Empty selection
// means "use the backend default".
const EMOJI_CHOICES: { name: string; glyph: string }[] = [
  { name: 'thumbsup', glyph: '👍' },
  { name: 'seedling', glyph: '🌱' },
  { name: 'rocket', glyph: '🚀' },
  { name: 'tada', glyph: '🎉' },
  { name: 'fire', glyph: '🔥' },
  { name: 'white_check_mark', glyph: '✅' },
  { name: 'eyes', glyph: '👀' },
  { name: 'heart', glyph: '❤️' },
  { name: 'sparkles', glyph: '✨' },
  { name: 'raised_hands', glyph: '🙌' },
]
const DEFAULT_EMOJIS = ['thumbsup', 'seedling']
const selectedEmojis = ref<Set<string>>(new Set(DEFAULT_EMOJIS))
const customEmoji = ref('')
function toggleEmoji(name: string) {
  const next = new Set(selectedEmojis.value)
  if (next.has(name)) next.delete(name)
  else next.add(name)
  selectedEmojis.value = next
}
function addCustomEmoji() {
  const name = customEmoji.value.trim().replace(/^:|:$/g, '')
  if (name && !selectedEmojis.value.has(name)) {
    selectedEmojis.value = new Set([...selectedEmojis.value, name])
  }
  customEmoji.value = ''
}
// Custom (non-curated) names the user added, so they render as removable chips too.
const customSelected = computed(() =>
  [...selectedEmojis.value].filter((n) => !EMOJI_CHOICES.some((c) => c.name === n)),
)

// What each bump produces, so the resulting tag never surprises. Kept in sync
// with the backend version calculator (NextRelease): major resets to
// (major+1).0.0; minor raises the minor per feat; patch raises the patch per fix.
const bumpHint = computed(() => {
  switch (bump.value) {
    case 'major':
      return 'Major — resets to the next whole version (e.g. 0.86.1 → 1.0.0), regardless of commits.'
    case 'patch':
      return 'Patch — raises the patch number by the number of fix commits (feats ignored).'
    default:
      return 'Minor — raises the minor number per feat commit, the patch per trailing fix.'
  }
})

// Live tag preview: a backend dry-run that computes the EXACT next tag the way the
// routine will (main ignores -dev tags; dev counts the MR's commits), so the
// version never surprises at the confirmation gate. Debounced on every input that
// affects the result.
type Preview = { nextTag: string; lastTag: string; featCount: number; fixCount: number }
const preview = ref<Preview | null>(null)
const previewLoading = ref(false)
const previewError = ref<string | null>(null)

async function loadPreview() {
  if (!props.open) return
  // Dev flow needs an MR; without one there is nothing to preview yet.
  if (props.flow === 'dev' && props.mergeRequest?.iid == null) {
    preview.value = null
    return
  }
  previewLoading.value = true
  previewError.value = null
  try {
    preview.value = await api.previewRoutineTag(props.repoId, {
      flow: props.flow,
      bump: bump.value,
      mrIid: props.flow === 'dev' ? props.mergeRequest?.iid : undefined,
      source: props.flow === 'main' ? sourceBranch.value.trim() : undefined,
      target: props.flow === 'main' ? targetBranch.value.trim() : undefined,
      includeDev: props.flow === 'main' ? includeDev.value : undefined,
    })
  } catch (e) {
    preview.value = null
    previewError.value = errorMessage(e)
  } finally {
    previewLoading.value = false
  }
}

// Recompute when any input that changes the tag changes (debounced to avoid a
// request per keystroke on the branch fields).
watchDebounced(
  () => [
    props.open,
    props.flow,
    bump.value,
    includeDev.value,
    sourceBranch.value,
    targetBranch.value,
    props.mergeRequest?.iid,
  ],
  loadPreview,
  { debounce: 350, immediate: true },
)

// Reset to per-flow defaults every time the modal opens so a previous edit never
// leaks into the next run. The main flow defaults mergeWhenPipelineSucceeds off.
watch(
  () => [props.open, props.flow] as const,
  ([open]) => {
    if (!open) return
    bump.value = 'minor'
    includeDev.value = false
    removeSourceBranch.value = false
    mergeWhenPipelineSucceeds.value = props.flow === 'dev'
    sourceBranch.value = ''
    targetBranch.value = ''
    selectedEmojis.value = new Set(DEFAULT_EMOJIS)
    customEmoji.value = ''
    branches.value = []
    branchesLoaded.value = false
    void loadBranches()
  },
  { immediate: true },
)

function onSubmit() {
  const emojis = [...selectedEmojis.value]
  if (props.flow === 'main') {
    emit('submit', {
      flow: 'main',
      input: {
        bump: bump.value,
        includeDev: includeDev.value,
        sourceBranch: sourceBranch.value.trim(),
        targetBranch: targetBranch.value.trim(),
        // Omit when empty so the backend default applies.
        ...(emojis.length > 0 ? { emojis } : {}),
        removeSourceBranch: removeSourceBranch.value,
        mergeWhenPipelineSucceeds: mergeWhenPipelineSucceeds.value,
      },
    })
    return
  }
  const iid = props.mergeRequest?.iid
  if (iid == null) return
  emit('submit', {
    flow: 'dev',
    input: {
      mrIid: iid,
      bump: bump.value,
      // Omit when empty so the backend default applies.
      ...(emojis.length > 0 ? { emojis } : {}),
      removeSourceBranch: removeSourceBranch.value,
      mergeWhenPipelineSucceeds: mergeWhenPipelineSucceeds.value,
    },
  })
}
</script>

<template>
  <Modal
    :open="open"
    :title="flow === 'main' ? 'Release to main' : 'Release'"
    @close="emit('close')"
  >
    <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
      <p v-if="flow === 'dev' && mergeRequest" class="text-muted text-sm">
        <span class="text-ink font-mono">!{{ mergeRequest.iid }}</span>
        {{ mergeRequest.title }}
        <span class="label-mono mt-0.5 block">
          {{ mergeRequest.sourceBranch }} → {{ mergeRequest.targetBranch }}
        </span>
      </p>
      <p v-else-if="flow === 'main'" class="text-muted text-sm">
        Cut a release from <span class="text-ink font-mono">development</span> into
        <span class="text-ink font-mono">main</span>. Override the branches below if needed.
      </p>

      <label class="block">
        <span class="field-label">Version bump</span>
        <select v-model="bump" class="field-underline">
          <option value="major">major</option>
          <option value="minor">minor</option>
          <option value="patch">patch</option>
        </select>
        <span class="text-muted mt-1.5 block text-xs">{{ bumpHint }}</span>
      </label>

      <!-- Live tag preview: the exact tag this release will create. -->
      <div class="border-line bg-surface/40 border-l-2 px-3 py-2.5" aria-live="polite">
        <div class="label-mono mb-1 flex items-center gap-1.5">
          Next tag
          <!-- Spinner while recomputing after a bump/branch change. -->
          <span
            v-if="previewLoading"
            class="i-lucide-loader-circle text-muted animate-spin text-xs"
            aria-hidden="true"
          />
        </div>
        <div v-if="previewLoading && !preview" class="text-muted text-sm">Computing…</div>
        <div v-else-if="previewError" class="text-warn text-xs">
          Couldn't preview the tag ({{ previewError }}). It's still computed and shown for approval
          at the confirmation gate.
        </div>
        <template v-else-if="preview">
          <!-- Dim the value while a recompute is in flight so the spinner reads. -->
          <div class="transition-opacity" :class="previewLoading ? 'opacity-40' : ''">
            <div class="text-ink font-mono text-lg font-semibold">{{ preview.nextTag }}</div>
            <div class="text-muted mt-0.5 text-xs">
              from <span class="font-mono">{{ preview.lastTag || 'no previous tag' }}</span> ·
              {{ preview.featCount }} feat, {{ preview.fixCount }} fix
            </div>
          </div>
        </template>
        <div v-else class="text-muted text-xs">
          Pick a merge request to preview its release tag.
        </div>
      </div>

      <!-- Main-flow base selection: by default the release ignores -dev prerelease
           tags and bases on the highest pure X.Y.Z release. Opt in to count them
           and promote the latest dev version (e.g. v2.0.0-dev → v2.0.0) instead. -->
      <template v-if="flow === 'main'">
        <label class="flex cursor-pointer items-start gap-2 text-sm">
          <input v-model="includeDev" type="checkbox" class="accent-accent mt-0.5" />
          <span>
            <span class="text-ink">Count <span class="font-mono">-dev</span> prereleases</span>
            <span class="text-muted mt-0.5 block text-xs">
              {{
                includeDev
                  ? 'Bases on the highest tag including -dev, promoting the latest dev version (the suffix is dropped), then applies the bump.'
                  : 'Bases on the highest pure X.Y.Z release; -dev tags are ignored.'
              }}
            </span>
          </span>
        </label>
        <p class="text-muted text-xs">Nothing is merged or tagged until you confirm at the gate.</p>
      </template>

      <label class="block">
        <span class="field-label">Reaction emojis</span>
        <div class="mt-1.5 flex flex-wrap gap-1.5">
          <button
            v-for="e in EMOJI_CHOICES"
            :key="e.name"
            type="button"
            class="inline-flex items-center gap-1.5 border px-2 py-1 text-xs transition-colors"
            :class="
              selectedEmojis.has(e.name)
                ? 'border-accent bg-accent/10 text-ink'
                : 'border-line text-muted hover:text-ink'
            "
            :aria-pressed="selectedEmojis.has(e.name)"
            @click="toggleEmoji(e.name)"
          >
            <span aria-hidden="true">{{ e.glyph }}</span>
            <span class="font-mono">{{ e.name }}</span>
          </button>
          <!-- Custom names the user typed -->
          <button
            v-for="name in customSelected"
            :key="name"
            type="button"
            class="border-accent bg-accent/10 text-ink inline-flex items-center gap-1.5 border px-2 py-1 font-mono text-xs"
            :aria-label="`Remove ${name}`"
            @click="toggleEmoji(name)"
          >
            {{ name }}
            <span class="i-lucide-x text-xs" aria-hidden="true" />
          </button>
        </div>
        <input
          v-model="customEmoji"
          type="text"
          class="field-underline mt-2"
          placeholder="add another emoji name…"
          autocomplete="off"
          @keydown.enter.prevent="addCustomEmoji"
        />
      </label>

      <template v-if="flow === 'main'">
        <!-- Alert: the routine assumes development → main; warn when they're absent
             so the user picks real branches instead of hitting a mid-run failure. -->
        <p
          v-if="missingConventional.length"
          class="border-warn/40 bg-warn/5 text-muted border-l-2 px-3 py-2 text-xs"
          role="alert"
        >
          <span class="text-warn flex items-center gap-1.5 font-medium">
            <span class="i-lucide-triangle-alert shrink-0 text-sm" aria-hidden="true" />
            {{ missingConventional.map((b) => `“${b}”`).join(' and ') }}
            {{ missingConventional.length === 1 ? 'branch' : 'branches' }} not found
          </span>
          <span class="mt-1 block">
            This repo has no {{ missingConventional.join('/') }} branch — pick the branches to release
            from and into below.
          </span>
        </p>
        <p v-else-if="branchesError" class="text-warn text-xs">
          Couldn't load branches ({{ branchesError }}). Type the branch names below.
        </p>

        <label class="block">
          <span class="field-label">Source branch</span>
          <select v-if="branches.length" v-model="sourceBranch" class="field-underline">
            <option value="" disabled>Select a branch…</option>
            <option v-for="b in branches" :key="b" :value="b">{{ b }}</option>
          </select>
          <input
            v-else
            v-model="sourceBranch"
            type="text"
            class="field-underline"
            placeholder="development"
          />
        </label>
        <label class="block">
          <span class="field-label">Target branch</span>
          <select v-if="branches.length" v-model="targetBranch" class="field-underline">
            <option value="" disabled>Select a branch…</option>
            <option v-for="b in branches" :key="b" :value="b">{{ b }}</option>
          </select>
          <input
            v-else
            v-model="targetBranch"
            type="text"
            class="field-underline"
            placeholder="main"
          />
        </label>
      </template>

      <label class="flex cursor-pointer items-center gap-2 text-sm">
        <input v-model="removeSourceBranch" type="checkbox" class="accent-accent" />
        <span class="text-ink">Remove source branch after merge</span>
      </label>
      <label class="flex cursor-pointer items-center gap-2 text-sm">
        <input v-model="mergeWhenPipelineSucceeds" type="checkbox" class="accent-accent" />
        <span class="text-ink">Merge when pipeline succeeds</span>
      </label>

      <div class="flex gap-2">
        <button type="submit" class="btn-accent w-full text-xs" :disabled="submitting">
          <span
            v-if="submitting"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-rocket text-sm" aria-hidden="true" />
          {{ flow === 'main' ? 'Release to main' : 'Start release' }}
        </button>
        <button type="button" class="btn-line w-full text-xs" @click="emit('close')">Cancel</button>
      </div>
    </form>
  </Modal>
</template>
