<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Profile } from '@shared/api/types'
import { useProfilesStore } from '@modules/profiles/store'
import { SCENARIOS, answersToSamples } from '@modules/profiles/guided'

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

// How the user supplies samples. Guided walks them through review scenarios;
// paste is the advanced blank-line-separated textarea. New profiles default to
// guided; editing an existing profile defaults to paste since stored samples[]
// carries no scenario mapping to decompose back into guided answers.
type SamplesMode = 'guided' | 'paste'
const mode = ref<SamplesMode>('guided')

// Guided answers, keyed by scenario. Kept separate from samplesText so toggling
// modes never clobbers either side's input.
const emptyAnswers = () => Object.fromEntries(SCENARIOS.map((s) => [s.key, '']))
const answers = reactive<Record<string, string>>(emptyAnswers())
function resetAnswers() {
  for (const s of SCENARIOS) answers[s.key] = ''
}

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
      // Existing samples have no scenario mapping — show them in paste mode so
      // the user keeps their data. Guided stays selectable but starts blank.
      resetAnswers()
      mode.value = 'paste'
    } else {
      Object.assign(form, blank())
      resetAnswers()
      mode.value = 'guided'
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
      samples: mode.value === 'guided' ? answersToSamples(answers) : parseSamples(form.samplesText),
    }
    if (props.editing) {
      await store.update(props.editing.id, base)
      toast.success('Profile updated')
    } else {
      await store.create(base)
      toast.success('Profile added')
    }
    Object.assign(form, blank())
    resetAnswers()
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

    <div class="flex flex-col gap-3">
      <div class="flex items-center justify-between gap-3">
        <span class="field-label mb-0">Writing samples</span>
        <div class="flex" role="group" aria-label="Samples input mode">
          <button
            type="button"
            class="btn px-3 py-1 text-xs"
            :class="mode === 'guided' ? 'bg-accent text-accent-ink' : 'border border-line text-muted hover:text-ink'"
            :aria-pressed="mode === 'guided'"
            @click="mode = 'guided'"
          >
            Guided
          </button>
          <button
            type="button"
            class="btn px-3 py-1 text-xs"
            :class="mode === 'paste' ? 'bg-accent text-accent-ink' : 'border border-line text-muted hover:text-ink'"
            :aria-pressed="mode === 'paste'"
            @click="mode = 'paste'"
          >
            Paste samples
          </button>
        </div>
      </div>

      <!-- Guided: each answered scenario becomes one sample, in order. -->
      <div v-if="mode === 'guided'" class="flex flex-col gap-4">
        <p class="text-muted/70 text-xs">
          Answer a few in your own voice — each answer becomes one writing sample. Skip any that
          don't fit.
        </p>
        <div v-for="s in SCENARIOS" :key="s.key">
          <label class="field-label" :for="`pf-guided-${s.key}`">{{ s.label }}</label>
          <textarea
            :id="`pf-guided-${s.key}`"
            v-model="answers[s.key]"
            class="field-underline min-h-20 resize-y"
            :placeholder="s.hint"
          />
        </div>
      </div>

      <!-- Paste: advanced blank-line-separated textarea (original behavior). -->
      <div v-else>
        <label class="field-label" for="pf-samples">
          Snippets
          <span class="text-muted/60 normal-case">— separate snippets with a blank line</span>
        </label>
        <textarea
          id="pf-samples"
          v-model="form.samplesText"
          class="field-underline min-h-40 resize-y"
          placeholder="Paste a snippet of your writing…&#10;&#10;…and another below a blank line."
        />
      </div>

      <p class="text-muted/70 text-xs">
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
