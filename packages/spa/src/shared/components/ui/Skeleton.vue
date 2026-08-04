<script setup lang="ts">
// A single pulsing placeholder block used to build list/row loading skeletons.
// Width/height default to a text-line shape; override via the `w`/`h` props
// (any UnoCSS sizing utility, e.g. "w-1/2", "h-4") or extra classes on the tag.
withDefaults(defineProps<{ w?: string; h?: string; rounded?: string }>(), {
  w: 'w-full',
  h: 'h-4',
  rounded: 'rounded',
})
</script>

<template>
  <span
    class="bg-line/60 block animate-pulse"
    :class="[w, h, rounded]"
    aria-hidden="true"
  />
</template>
