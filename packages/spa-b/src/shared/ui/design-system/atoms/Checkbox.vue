<script setup lang="ts">
/**
 * Wraps Reka UI's `CheckboxRoot` + `CheckboxIndicator`, token-styled.
 */
import { CheckboxRoot, CheckboxIndicator } from 'reka-ui'
import Icon from './Icon.vue'

const {
  id,
  disabled,
  'aria-invalid': ariaInvalid,
  'aria-describedby': ariaDescribedby,
} = defineProps<{
  id?: string
  disabled?: boolean
  'aria-invalid'?: boolean
  'aria-describedby'?: string
}>()

const modelValue = defineModel<boolean | 'indeterminate'>({ default: false })
</script>

<template>
  <CheckboxRoot
    :id="id"
    v-model="modelValue"
    :disabled="disabled"
    :aria-invalid="ariaInvalid"
    :aria-describedby="ariaDescribedby"
    class="flex size-4 shrink-0 items-center justify-center rounded-sm border border-line bg-bg-panel transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-accent data-[state=checked]:bg-accent data-[state=indeterminate]:border-accent data-[state=indeterminate]:bg-accent"
  >
    <CheckboxIndicator class="flex items-center justify-center text-text-on-accent">
      <Icon v-if="modelValue === 'indeterminate'" name="minus" size="xs" />
      <Icon v-else name="check" size="xs" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
