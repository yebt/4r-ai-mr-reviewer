<script setup lang="ts">
/**
 * Field — the reusable form-wrapper molecule. Owns the a11y wiring so every
 * form control (Input, Textarea, Select, Checkbox, Switch) associates
 * correctly with its label/description/error without each atom having to
 * reimplement id generation.
 *
 * Usage:
 * ```html
 * <Field label="Email" :error="errors.email" v-slot="{ id, describedBy, invalid }">
 *   <Input :id="id" v-model="email" :aria-describedby="describedBy" :aria-invalid="invalid" />
 * </Field>
 * ```
 */
import { computed, useId } from 'vue'

const props = defineProps<{
  label: string
  description?: string
  error?: string
  required?: boolean
  /** Override the generated control id (e.g. to match an existing form schema). */
  id?: string
}>()

const generatedId = useId()
const controlId = computed(() => props.id ?? generatedId)
const descriptionId = computed(() => `${controlId.value}-description`)
const errorId = computed(() => `${controlId.value}-error`)

const invalid = computed(() => Boolean(props.error))

const describedBy = computed(() => {
  const ids = [
    props.description ? descriptionId.value : null,
    props.error ? errorId.value : null,
  ].filter((value): value is string => value !== null)
  return ids.length > 0 ? ids.join(' ') : undefined
})
</script>

<template>
  <div class="flex flex-col gap-1.5">
    <label :for="controlId" class="text-sm font-medium text-text">
      {{ label }}
      <span v-if="required" class="text-danger-text" aria-hidden="true">*</span>
    </label>

    <slot :id="controlId" :described-by="describedBy" :invalid="invalid" />

    <p v-if="description" :id="descriptionId" class="text-xs text-text-muted">
      {{ description }}
    </p>
    <p v-if="error" :id="errorId" role="alert" class="text-xs text-danger-text">
      {{ error }}
    </p>
  </div>
</template>
