<script setup lang="ts">
import { ref, watch } from 'vue'
import Modal from '@shared/components/ui/Modal.vue'
import type { ApproveAndTagInput, MergeRequest } from '@shared/api/types'

const props = defineProps<{
  open: boolean
  mergeRequest: MergeRequest | null
  submitting?: boolean
}>()
const emit = defineEmits<{ submit: [input: ApproveAndTagInput]; close: [] }>()

type Bump = 'major' | 'minor' | 'patch'

// Fields mirror the backend defaults so the shown values match what a bare run
// would do (emojis: thumbsup/seedling, comment: "LGFM", bump: patch).
const bump = ref<Bump>('patch')
const comment = ref('LGFM')
const emojis = ref('thumbsup, seedling')

// Reset to defaults every time the modal opens so a previous edit never leaks
// into the next merge request.
watch(
  () => props.open,
  (open) => {
    if (open) {
      bump.value = 'patch'
      comment.value = 'LGFM'
      emojis.value = 'thumbsup, seedling'
    }
  },
)

function onSubmit() {
  const iid = props.mergeRequest?.iid
  if (iid == null) return
  const parsedEmojis = emojis.value
    .split(',')
    .map((e) => e.trim())
    .filter(Boolean)
  emit('submit', {
    mrIid: iid,
    bump: bump.value,
    comment: comment.value.trim(),
    emojis: parsedEmojis,
  })
}
</script>

<template>
  <Modal :open="open" title="Approve & tag" @close="emit('close')">
    <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
      <p v-if="mergeRequest" class="text-muted text-sm">
        <span class="text-ink font-mono">!{{ mergeRequest.iid }}</span>
        {{ mergeRequest.title }}
      </p>

      <label class="block">
        <span class="field-label">Version bump</span>
        <select v-model="bump" class="field-underline">
          <option value="major">major</option>
          <option value="minor">minor</option>
          <option value="patch">patch</option>
        </select>
      </label>

      <label class="block">
        <span class="field-label">Comment</span>
        <input v-model="comment" type="text" class="field-underline" placeholder="LGFM" />
      </label>

      <label class="block">
        <span class="field-label">Emojis (comma-separated)</span>
        <input
          v-model="emojis"
          type="text"
          class="field-underline"
          placeholder="thumbsup, seedling"
        />
      </label>

      <div class="flex gap-2">
        <button type="submit" class="btn-accent w-full text-xs" :disabled="submitting">
          <span
            v-if="submitting"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-tag text-sm" aria-hidden="true" />
          Start routine
        </button>
        <button type="button" class="btn-line w-full text-xs" @click="emit('close')">Cancel</button>
      </div>
    </form>
  </Modal>
</template>
