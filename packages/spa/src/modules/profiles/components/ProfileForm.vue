<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Profile } from '@shared/api/types'
import { useProfilesStore } from '@modules/profiles/store'

const props = defineProps<{ editing?: Profile | null }>()
const emit = defineEmits<{ done: [] }>()

const store = useProfilesStore()

const blank = () => ({
  name: '',
  language: '',
  formality: '',
  emojis: false,
  // Raw textarea; blank-line-separated blocks become individual samples.
  samplesText: '',
})
const form = reactive(blank())
const submitting = ref(false)
const error = ref<string | null>(null)

const isEdit = computed(() => !!props.editing)
const valid = computed(() => !!form.name.trim())

// Blank-line separated blocks → sample array. A single writing snippet may span
// multiple lines, so we split on blank lines (not every newline) and drop empties.
function parseSamples(text: string): string[] {
  return text
    .split(/\n\s*\n+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

watch(
  () => props.editing,
  (p) => {
    if (p) {
      form.name = p.name
      form.language = p.language
      form.formality = p.formality
      form.emojis = p.emojis
      form.samplesText = p.samples.join('\n\n')
    } else {
      Object.assign(form, blank())
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
    const base = {
      name: form.name.trim(),
      language: form.language.trim(),
      formality: form.formality.trim(),
      emojis: form.emojis,
      samples: parseSamples(form.samplesText),
    }
    if (props.editing) {
      await store.update(props.editing.id, base)
      toast.success('Profile updated')
    } else {
      await store.create(base)
      toast.success('Profile added')
    }
    Object.assign(form, blank())
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
    <div>
      <label class="field-label" for="pf-name">Name</label>
      <input
        id="pf-name"
        v-model="form.name"
        class="field-underline"
        placeholder="my voice"
        autocomplete="off"
      />
    </div>

    <div>
      <label class="field-label" for="pf-language"
        >Language <span class="text-muted/60 normal-case">— optional</span></label
      >
      <input
        id="pf-language"
        v-model="form.language"
        class="field-underline"
        placeholder="es-AR, en, …"
        autocomplete="off"
      />
    </div>

    <div>
      <label class="field-label" for="pf-formality"
        >Formality <span class="text-muted/60 normal-case">— optional</span></label
      >
      <input
        id="pf-formality"
        v-model="form.formality"
        class="field-underline"
        placeholder="casual / neutral / formal"
        autocomplete="off"
      />
    </div>

    <label class="text-muted flex cursor-pointer items-center gap-2 text-sm select-none">
      <input v-model="form.emojis" type="checkbox" class="accent-accent" />
      Use emojis
    </label>

    <div>
      <label class="field-label" for="pf-samples">
        Writing samples
        <span class="text-muted/60 normal-case">— separate snippets with a blank line</span>
      </label>
      <textarea
        id="pf-samples"
        v-model="form.samplesText"
        class="field-underline min-h-40 resize-y"
        placeholder="Paste a snippet of your writing…&#10;&#10;…and another below a blank line."
      />
      <p class="text-muted/70 mt-1.5 text-xs">
        Saving with samples starts distillation — the style guide appears in the list when ready.
      </p>
    </div>

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div class="flex items-center gap-3">
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : isEdit ? 'Save changes' : 'Add profile' }}
      </button>
      <button v-if="isEdit" type="button" class="btn-ghost" @click="emit('done')">Cancel</button>
    </div>
  </form>
</template>
