<script setup lang="ts">
/**
 * Mobile-only floating action button (design-system molecule). Circular
 * accent button fixed to the bottom-right, used to surface a section's
 * primary "Add …" action without competing for space in a header on
 * narrow viewports. Desktop keeps the inline header button instead — this
 * component renders nothing above the `md` breakpoint (`md:hidden`).
 *
 * Positioning: `AppBottomNav.vue`'s tab row is `min-h-11` (2.75rem) but its
 * intrinsic content (a `size-md` icon + `gap-0.5` + `text-xs` label, inside
 * `py-2`) runs taller at ~3.125rem, so that governs the actual rendered
 * height. `bottom` below adds a comfortable ~1rem gap on top of that
 * (rounded up to 3.25rem) plus the device's safe-area inset, so the FAB
 * always clears the bottom nav bar.
 */
import Icon from '../atoms/Icon.vue'

withDefaults(
  defineProps<{
    /** Required accessible name — there is no visible text label. */
    label: string
    /** kebab-case Icon name, see Icon.vue's registry. */
    icon?: string
  }>(),
  {
    icon: 'plus',
  },
)

const emit = defineEmits<{
  click: [event: MouseEvent]
}>()
</script>

<template>
  <button
    type="button"
    :aria-label="label"
    class="fixed right-4 z-30 flex size-14 items-center justify-center rounded-full bg-accent text-text-on-accent shadow-token-lg transition-transform hover:bg-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring active:scale-95 motion-reduce:active:scale-100 md:hidden"
    style="bottom: calc(env(safe-area-inset-bottom, 0px) + 4.25rem)"
    @click="emit('click', $event)"
  >
    <Icon :name="icon" size="lg" />
  </button>
</template>
