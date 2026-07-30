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
import { ref, watch } from 'vue'
import Modal from '@shared/components/ui/Modal.vue'
import type { Bump, MergeRequest } from '@shared/api/types'

const props = defineProps<{
  open: boolean
  flow: 'dev' | 'main'
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
  },
  { immediate: true },
)

function onSubmit() {
  if (props.flow === 'main') {
    emit('submit', {
      flow: 'main',
      input: {
        bump: bump.value,
        sourceBranch: sourceBranch.value.trim(),
        targetBranch: targetBranch.value.trim(),
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
