<script setup lang="ts">
/**
 * Primary interactive atom. Owns no Reka primitive (plain `<button>`); Reka
 * doesn't ship a Button primitive, so this atom is the styled base every
 * other interactive atom composes around.
 */
import { computed } from 'vue'
import Spinner from './Spinner.vue'

type ButtonVariant = 'accent' | 'soft' | 'outline' | 'ghost' | 'danger'
type ButtonSize = 'sm' | 'md' | 'lg'

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariant
    size?: ButtonSize
    loading?: boolean
    disabled?: boolean
    /** Native `type` attribute; defaults to `button` so it never submits by accident. */
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'accent',
    size: 'md',
    loading: false,
    disabled: false,
    type: 'button',
  },
)

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()

// Only the real `disabled` prop uses the native attribute. A loading button
// must stay focusable (a natively disabled element drops focus to <body>
// mid-action), so it is announced via aria-disabled and blocked in JS.
const isInert = computed(() => props.disabled || props.loading)

const variantClassMap: Record<ButtonVariant, string> = {
  accent: 'bg-accent text-text-on-accent hover:bg-accent-hover',
  soft: 'bg-accent-subtle-bg text-accent-text hover:bg-accent-subtle-bg/80',
  outline: 'border border-line bg-transparent text-text hover:bg-bg-hover',
  ghost: 'bg-transparent text-text hover:bg-bg-hover',
  danger: 'bg-danger-solid text-danger-contrast hover:opacity-90',
}

const sizeClassMap: Record<ButtonSize, string> = {
  sm: 'h-7 gap-1.5 rounded-md px-2.5 text-xs',
  md: 'h-8 gap-2 rounded-md px-3 text-sm',
  lg: 'h-10 gap-2 rounded-lg px-4 text-base',
}

const spinnerSizeMap: Record<ButtonSize, 'xs' | 'sm' | 'md'> = {
  sm: 'xs',
  md: 'sm',
  lg: 'md',
}

const variantClass = computed(() => variantClassMap[props.variant])
const sizeClass = computed(() => sizeClassMap[props.size])
const spinnerSize = computed(() => spinnerSizeMap[props.size])

function onClick(event: MouseEvent) {
  if (isInert.value) return
  emit('click', event)
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    :aria-disabled="loading && !disabled ? 'true' : undefined"
    :aria-busy="loading || undefined"
    :class="[variantClass, sizeClass]"
    class="inline-flex shrink-0 items-center justify-center whitespace-nowrap font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 aria-disabled:cursor-progress aria-disabled:opacity-50"
    @click="onClick"
  >
    <Spinner v-if="loading" :size="spinnerSize" />
    <slot v-else name="leading" />
    <slot />
    <slot v-if="!loading" name="trailing" />
  </button>
</template>
