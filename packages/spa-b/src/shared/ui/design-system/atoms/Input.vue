<script setup lang="ts">
/**
 * Text input. Owns no Reka primitive — Reka doesn't ship a text input, so
 * this is a plain styled `<input>`. Designed to be used inside Field, which
 * supplies `id`/`aria-describedby`/`aria-invalid` via its scoped slot; those
 * fall through as ordinary attrs (inheritAttrs is off so they land on the
 * `<input>` itself rather than on some outer wrapper).
 */
defineOptions({ inheritAttrs: false })

withDefaults(
  defineProps<{
    id?: string
    type?: string
    placeholder?: string
    disabled?: boolean
  }>(),
  {
    type: 'text',
  },
)

const modelValue = defineModel<string>({ default: '' })
</script>

<template>
  <input
    :id="id"
    v-model="modelValue"
    :type="type"
    :placeholder="placeholder"
    :disabled="disabled"
    v-bind="$attrs"
    class="h-8 w-full rounded-md border border-line bg-bg-panel px-2.5 text-sm text-text placeholder:text-text-placeholder transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 aria-[invalid=true]:border-danger-solid"
  />
</template>
