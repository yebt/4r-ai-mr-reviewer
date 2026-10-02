<script setup lang="ts">
/**
 * Wraps Reka UI's `SwitchRoot` + `SwitchThumb`, token-styled.
 */
import { SwitchRoot, SwitchThumb } from 'reka-ui'

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

const modelValue = defineModel<boolean>({ default: false })
</script>

<template>
  <SwitchRoot
    :id="id"
    v-model="modelValue"
    :disabled="disabled"
    :aria-invalid="ariaInvalid"
    :aria-describedby="ariaDescribedby"
    class="relative inline-flex h-5 w-9 shrink-0 items-center rounded-full bg-bg-active transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-accent"
  >
    <SwitchThumb
      class="block size-4 translate-x-0.5 rounded-full bg-bg-panel-raised shadow-token-sm transition-transform data-[state=checked]:translate-x-4"
    />
  </SwitchRoot>
</template>
