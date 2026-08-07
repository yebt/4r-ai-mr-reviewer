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
const removeSourceBranch = ref(false)
const mergeWhenPipelineSucceeds = ref(true)
// Main flow only. Empty means "use the backend default" (development → main).
const sourceBranch = ref('')
const targetBranch = ref('')
// GitLab emoji NAMES (not unicode), comma-separated. Parsed on submit into a
// string[]; left/parsed empty means "use the backend default".
const DEFAULT_EMOJIS = 'thumbsup, seedling'
const emojiInput = ref(DEFAULT_EMOJIS)

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
  () => [props.open, props.flow, bump.value, sourceBranch.value, targetBranch.value, props.mergeRequest?.iid],
  loadPreview,
  { debounce: 350, immediate: true },
)

function parseEmojis(raw: string): string[] {
  return raw
    .split(',')
    .map((name) => name.trim())
    .filter((name) => name.length > 0)
}

// Reset to per-flow defaults every time the modal opens so a previous edit never
// leaks into the next run. The main flow defaults mergeWhenPipelineSucceeds off.
watch(
  () => [props.open, props.flow] as const,
  ([open]) => {
    if (!open) return
    bump.value = 'minor'
    removeSourceBranch.value = false
    mergeWhenPipelineSucceeds.value = props.flow === 'dev'
    sourceBranch.value = ''
    targetBranch.value = ''
    emojiInput.value = DEFAULT_EMOJIS
  },
  { immediate: true },
)

function onSubmit() {
  const emojis = parseEmojis(emojiInput.value)
  if (props.flow === 'main') {
    emit('submit', {
      flow: 'main',
      input: {
        bump: bump.value,
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
        <div class="label-mono mb-1">Next tag</div>
        <div v-if="previewLoading && !preview" class="text-muted flex items-center gap-2 text-sm">
          <span class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
          Computing…
        </div>
        <div v-else-if="previewError" class="text-warn text-xs">
          Couldn't preview the tag ({{ previewError }}). It's still computed and shown for approval
          at the confirmation gate.
        </div>
        <template v-else-if="preview">
          <div class="text-ink font-mono text-lg font-semibold">{{ preview.nextTag }}</div>
          <div class="text-muted mt-0.5 text-xs">
            from <span class="font-mono">{{ preview.lastTag || 'no previous tag' }}</span> ·
            {{ preview.featCount }} feat, {{ preview.fixCount }} fix
          </div>
        </template>
        <div v-else class="text-muted text-xs">
          Pick a merge request to preview its release tag.
        </div>
      </div>

      <!-- Main-flow gotcha: the main release ignores -dev prerelease tags, so it
           bases the next version on the highest PURE release, not a dev tag. -->
      <p v-if="flow === 'main'" class="text-muted text-xs">
        Based on the highest <span class="text-ink font-mono">X.Y.Z</span> release tag —
        <span class="text-ink">-dev tags are ignored</span>. Nothing is merged or tagged until you
        confirm at the gate.
      </p>

      <label class="block">
        <span class="field-label">Reaction emojis</span>
        <input
          v-model="emojiInput"
          type="text"
          class="field-underline"
          placeholder="thumbsup, seedling"
        />
        <span class="text-muted mt-1 block text-xs">GitLab emoji names, comma-separated</span>
      </label>

      <template v-if="flow === 'main'">
        <label class="block">
          <span class="field-label">Source branch</span>
          <input v-model="sourceBranch" type="text" class="field-underline" placeholder="development" />
        </label>
        <label class="block">
          <span class="field-label">Target branch</span>
          <input v-model="targetBranch" type="text" class="field-underline" placeholder="main" />
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
