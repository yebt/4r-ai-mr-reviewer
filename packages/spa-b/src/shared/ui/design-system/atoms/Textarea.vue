<script setup lang="ts">
/**
 * Multi-line text input. Same styling and attrs-fallthrough contract as
 * Input; used inside Field.
 */
defineOptions({ inheritAttrs: false })

withDefaults(
  defineProps<{
    id?: string
    placeholder?: string
    disabled?: boolean
    rows?: number
  }>(),
  {
    rows: 3,
  },
)

const modelValue = defineModel<string>({ default: '' })
</script>

<template>
  <textarea
    :id="id"
    v-model="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    :rows="rows"
    v-bind="$attrs"
    class="w-full resize-y rounded-md border border-line-control bg-bg-panel px-2.5 py-2 text-[1rem] text-text md:text-sm placeholder:text-text-placeholder transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 aria-[invalid=true]:border-danger-solid"
  />
</template>
