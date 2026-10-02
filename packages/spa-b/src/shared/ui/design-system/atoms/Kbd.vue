<script setup lang="ts">
/**
 * Keyboard-hint chip, e.g. `<Kbd>⌘K</Kbd>` next to a command-palette trigger.
 */
</script>

<template>
  <kbd
    class="inline-flex min-w-[1.375rem] items-center justify-center rounded-sm border border-line-subtle bg-bg-panel px-1.5 py-0.5 font-mono text-xs text-text-muted shadow-token-sm"
  >
    <slot />
  </kbd>
</template>
