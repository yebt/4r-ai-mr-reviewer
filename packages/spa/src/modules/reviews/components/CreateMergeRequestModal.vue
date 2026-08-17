<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Modal from '@shared/components/ui/Modal.vue'
import { api, errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { MergeRequest, Profile } from '@shared/api/types'

// Create a merge request between two existing branches, with an AI-drafted
// title and description. The user picks source → target, optionally a voice
// profile, then "Generate with AI" fills the editable title/description from the
// diff. Nothing is created until "Create merge request": the draft is always
// reviewed (and editable) first. Branch creation and commits are out of scope.
const props = defineProps<{
  open: boolean
  repoId: string
  branches: string[]
  branchesLoading?: boolean
  profiles: Profile[]
  // Preselected target branch (the repo's default branch), if known.
  defaultTargetBranch?: string
}>()
const emit = defineEmits<{
  created: [mr: MergeRequest]
  close: []
}>()

const source = ref('')
const target = ref('')
const profileId = ref('')
const title = ref('')
const description = ref('')
const generating = ref(false)
const creating = ref(false)

// Only profiles with a distilled style guide can voice the description; the
// backend rejects the rest with 409, so they are not offered here.
const voiceProfiles = computed(() => props.profiles.filter((p) => p.styleGuideStatus === 'ready'))

const sameBranch = computed(
  () => !!source.value && !!target.value && source.value === target.value,
)
const canGenerate = computed(
  () => !!source.value && !!target.value && !sameBranch.value && !generating.value && !creating.value,
)
const canCreate = computed(() => canGenerate.value && !!title.value.trim())

// Reset every time the modal opens so a previous draft never leaks into the
// next one; preselect the target to the repo's default branch when known.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    source.value = ''
    target.value = props.defaultTargetBranch && props.branches.includes(props.defaultTargetBranch)
      ? props.defaultTargetBranch
      : ''
    profileId.value = ''
    title.value = ''
    description.value = ''
    generating.value = false
    creating.value = false
  },
  { immediate: true },
)

async function onGenerate() {
  if (!canGenerate.value) return
  generating.value = true
  try {
    const draft = await api.generateMergeRequest(props.repoId, {
      sourceBranch: source.value,
      targetBranch: target.value,
      profileId: profileId.value || undefined,
    })
    title.value = draft.title
    description.value = draft.description
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    generating.value = false
  }
}

async function onCreate() {
  if (!canCreate.value) return
  creating.value = true
  try {
    const mr = await api.createMergeRequest(props.repoId, {
      sourceBranch: source.value,
      targetBranch: target.value,
      title: title.value.trim(),
      description: description.value.trim(),
    })
    emit('created', mr)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <Modal :open="open" title="New merge request" size="lg" @close="emit('close')">
    <form class="flex flex-col gap-4" @submit.prevent="onCreate">
      <!-- Direction: source → target -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_auto_1fr] sm:items-end">
        <label class="block">
          <span class="field-label">Source branch</span>
          <select v-model="source" class="field-underline" :disabled="branchesLoading">
            <option value="">{{ branchesLoading ? 'Loading…' : 'Select branch' }}</option>
            <option v-for="b in branches" :key="b" :value="b">{{ b }}</option>
          </select>
        </label>
        <span class="text-muted hidden pb-2 sm:block" aria-hidden="true">
          <span class="i-lucide-arrow-right text-sm" />
        </span>
        <label class="block">
          <span class="field-label">Target branch</span>
          <select v-model="target" class="field-underline" :disabled="branchesLoading">
            <option value="">{{ branchesLoading ? 'Loading…' : 'Select branch' }}</option>
            <option v-for="b in branches" :key="b" :value="b">{{ b }}</option>
          </select>
        </label>
      </div>
      <p v-if="sameBranch" class="text-warn text-xs">
        Source and target must be different branches.
      </p>

      <!-- Voice + generate -->
      <div class="flex flex-wrap items-end gap-3">
        <label class="block min-w-0 flex-1">
          <span class="field-label">
            Voice <span class="text-muted normal-case">— optional, drafts the description</span>
          </span>
          <select v-model="profileId" class="field-underline">
            <option value="">Plain English</option>
            <option v-for="p in voiceProfiles" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
        </label>
        <button
          type="button"
          class="btn-line text-xs"
          :disabled="!canGenerate"
          @click="onGenerate"
        >
          <span
            v-if="generating"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-sparkles text-sm" aria-hidden="true" />
          {{ generating ? 'Drafting…' : 'Generate with AI' }}
        </button>
      </div>

      <!-- Editable draft -->
      <label class="block">
        <span class="field-label">Title</span>
        <input
          v-model="title"
          type="text"
          class="field-underline"
          placeholder="A concise, imperative summary"
          autocomplete="off"
        />
      </label>

      <label class="block">
        <span class="field-label">
          Description <span class="text-muted normal-case">— Markdown</span>
        </span>
        <textarea
          v-model="description"
          rows="10"
          class="field-underline resize-y font-mono text-xs leading-relaxed"
          placeholder="Describe the change, or generate it from the diff above."
        />
      </label>

      <div class="flex gap-2">
        <button type="submit" class="btn-accent w-full text-xs" :disabled="!canCreate">
          <span
            v-if="creating"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-git-pull-request-create text-sm" aria-hidden="true" />
          Create merge request
        </button>
        <button type="button" class="btn-ghost w-full text-xs" @click="emit('close')">Cancel</button>
      </div>
    </form>
  </Modal>
</template>
