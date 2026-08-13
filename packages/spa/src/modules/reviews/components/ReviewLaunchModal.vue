<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Modal from '@shared/components/ui/Modal.vue'
import type { MergeRequest, Provider } from '@shared/api/types'

// Launch modal for a review on an open merge request. The MR row no longer
// carries inline provider/model/mode selects; instead its "Review" button opens
// this modal where the provider, model and context mode are chosen, then a
// single "Start review" fires the create. Purely presentational: the consumer
// owns the actual create call via the `submit` event.
const props = defineProps<{
  open: boolean
  mergeRequest: MergeRequest | null
  providers: Provider[]
  // Preselected provider id (repo's provider, else the global default).
  defaultProviderId: string
  submitting?: boolean
}>()
const emit = defineEmits<{
  submit: [payload: { mode: string; providerId: string; model: string }]
  close: []
}>()

const providerId = ref('')
const model = ref('')
const mode = ref('fast')

// Models declared by the selected provider (empty if none), sorted alphabetically
// (case-insensitive) without mutating the store's array. Mirrors the logic that
// used to live in MergeRequestList's `providerModelsFor`.
const providerModels = computed(() => {
  const models = props.providers.find((p) => p.id === providerId.value)?.models ?? []
  return [...models].sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
})

// Reset to per-MR defaults every time the modal opens so a previous edit never
// leaks into the next launch.
watch(
  () => [props.open, props.defaultProviderId] as const,
  ([open]) => {
    if (!open) return
    providerId.value = props.defaultProviderId
    model.value = ''
    mode.value = 'fast'
  },
  { immediate: true },
)

// Switching provider drops the model override (a model from the old provider may
// not exist on the new one), matching the old inline-select behavior.
function onProviderChange(id: string) {
  providerId.value = id
  model.value = ''
}

function onSubmit() {
  if (props.submitting) return
  emit('submit', { mode: mode.value, providerId: providerId.value, model: model.value.trim() })
}
</script>

<template>
  <Modal :open="open" title="Start a review" @close="emit('close')">
    <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
      <p v-if="mergeRequest" class="text-muted text-sm">
        <span class="text-ink font-mono">!{{ mergeRequest.iid }}</span>
        {{ mergeRequest.title }}
        <span class="label-mono mt-0.5 block">
          {{ mergeRequest.sourceBranch }} → {{ mergeRequest.targetBranch }}
        </span>
      </p>

      <label v-if="providers.length" class="block">
        <span class="field-label">Provider</span>
        <select
          :value="providerId"
          class="field-underline"
          @change="onProviderChange(($event.target as HTMLSelectElement).value)"
        >
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </label>

      <label class="block">
        <span class="field-label">
          Model <span class="text-muted normal-case">— optional, uses the provider's model</span>
        </span>
        <select v-if="providerModels.length" v-model="model" class="field-underline">
          <option value="">default model</option>
          <option v-for="m in providerModels" :key="m" :value="m">{{ m }}</option>
        </select>
        <input
          v-else
          v-model="model"
          type="text"
          class="field-underline"
          placeholder="default model"
          autocomplete="off"
        />
      </label>

      <label class="block">
        <span class="field-label">Context mode</span>
        <select v-model="mode" class="field-underline">
          <option value="fast">fast</option>
          <option value="deep">deep</option>
        </select>
      </label>

      <div class="flex gap-2">
        <button type="submit" class="btn-accent w-full text-xs" :disabled="submitting">
          <span
            v-if="submitting"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-scan-search text-sm" aria-hidden="true" />
          Start review
        </button>
        <button type="button" class="btn-ghost w-full text-xs" @click="emit('close')">Cancel</button>
      </div>
    </form>
  </Modal>
</template>
