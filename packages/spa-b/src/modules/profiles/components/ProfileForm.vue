<script setup lang="ts">
/**
 * Create/edit form for a single writing-voice profile. Used inside a Reka
 * `Dialog` by `ProfilesSection`. Keyed by `profile?.id ?? 'create'` in the
 * parent so switching context (create -> edit another row) always remounts
 * with fresh local state, rather than trying to reset a shared instance in
 * place.
 */
import { nextTick, reactive, ref, computed } from 'vue'
import { Badge, Button, Checkbox, Field, Icon, Input, Select, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useProfilesStore } from '../store'
import type { Profile } from '../types'

const props = defineProps<{
  profile?: Profile | null
}>()

const emit = defineEmits<{
  saved: [profile: Profile]
  cancel: []
}>()

const store = useProfilesStore()

const isEditing = computed(() => props.profile != null)

const formalityOptions: SelectItemOption[] = [
  { label: 'Formal', value: 'formal' },
  { label: 'Neutral', value: 'neutral' },
  { label: 'Casual', value: 'casual' },
]

const form = reactive({
  name: props.profile?.name ?? '',
  language: props.profile?.language ?? '',
  formality: props.profile?.formality ?? 'neutral',
  emojis: props.profile?.emojis ?? false,
  samples: [...(props.profile?.samples ?? [])] as string[],
})

type FormErrors = Partial<Record<'name' | 'language' | 'samples', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const newSampleInput = ref('')
const saving = ref(false)
const formError = ref<string | null>(null)

function addSample() {
  const value = newSampleInput.value.trim()
  if (value && !form.samples.includes(value)) {
    form.samples.push(value)
  }
  newSampleInput.value = ''
}

function removeSample(sample: string) {
  form.samples = form.samples.filter((existing) => existing !== sample)
}

async function handleSubmit() {
  errors.name = form.name.trim() ? undefined : 'Name is required'
  errors.language = form.language.trim() ? undefined : 'Language is required'
  errors.samples = form.samples.length > 0 ? undefined : 'At least one sample is required'

  const hasErrors = Object.values(errors).some((value) => value !== undefined)
  if (hasErrors) {
    await nextTick()
    formRef.value?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
    return
  }

  saving.value = true
  formError.value = null
  try {
    const result = props.profile
      ? await store.updateProfile(props.profile.id, {
          name: form.name,
          language: form.language,
          formality: form.formality,
          emojis: form.emojis,
          samples: form.samples,
        })
      : await store.createProfile({
          name: form.name,
          language: form.language,
          formality: form.formality,
          emojis: form.emojis,
          samples: form.samples,
        })
    emit('saved', result)
  } catch (err) {
    formError.value = err instanceof Error ? err.message : 'Failed to save profile'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <form ref="formRef" class="flex flex-col gap-4" novalidate @submit.prevent="handleSubmit">
    <Field label="Name" required :error="errors.name" v-slot="{ id, describedBy, invalid }">
      <Input :id="id" v-model="form.name" autocomplete="off" :aria-describedby="describedBy" :aria-invalid="invalid" />
    </Field>

    <Field label="Language" required :error="errors.language" v-slot="{ id, describedBy, invalid }">
      <Input
        :id="id"
        v-model="form.language"
        placeholder="es, en, es-CO..."
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field label="Formality" required v-slot="{ id, describedBy }">
      <Select
        :id="id"
        :model-value="form.formality"
        :items="formalityOptions"
        :aria-describedby="describedBy"
        @update:model-value="(value) => (form.formality = value as string)"
      />
    </Field>

    <div class="flex items-center gap-2">
      <Checkbox
        id="profile-emojis"
        :model-value="form.emojis"
        @update:model-value="(value) => (form.emojis = value === true)"
      />
      <label for="profile-emojis" class="text-sm text-text">Allow emojis</label>
    </div>

    <div class="flex flex-col gap-1.5">
      <span class="text-sm font-medium text-text">Samples</span>
      <Text v-if="errors.samples" size="xs" class="text-danger-text">{{ errors.samples }}</Text>

      <div v-if="form.samples.length > 0" class="flex flex-col gap-1.5">
        <Badge v-for="sample in form.samples" :key="sample" class="w-fit max-w-full items-start whitespace-normal text-left">
          <span class="line-clamp-3">{{ sample }}</span>
          <button
            type="button"
            class="ml-1 shrink-0 text-text-muted transition-colors hover:text-danger-text"
            :aria-label="`Remove sample`"
            @click="removeSample(sample)"
          >
            <Icon name="x" size="xs" />
          </button>
        </Badge>
      </div>

      <div class="flex gap-2">
        <Input v-model="newSampleInput" placeholder="Add a writing sample" @keydown.enter.prevent="addSample" />
        <Button type="button" variant="outline" @click="addSample">Add</Button>
      </div>
    </div>

    <p v-if="formError" role="alert" class="text-sm text-danger-text">{{ formError }}</p>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add profile' }}</Button>
    </div>
  </form>
</template>
