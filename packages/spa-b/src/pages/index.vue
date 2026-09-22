<script setup lang="ts">
/**
 * Home. Renders inside AppShell (see src/core/App.vue). Minimal on
 * purpose — this is a placeholder until the real dashboard lands; the
 * former token-preview content that used to live here moved to
 * `src/pages/design.vue` (dev-only reference route).
 */
import { Heading, Icon, Kbd, Text } from '@shared/ui/design-system'
import { useCommandPalette } from '@shared/composables/useCommandPalette'

const { open: openPalette } = useCommandPalette()
</script>

<template>
  <div class="flex flex-col items-start gap-4">
    <div class="flex flex-col gap-1">
      <Heading :level="2" size="2xl">Welcome back</Heading>
      <Text muted>The centralized shell is live — more lands here soon.</Text>
    </div>

    <button
      type="button"
      class="flex items-center gap-2 rounded-md border border-line bg-bg-panel px-3 py-2 text-sm text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      @click="openPalette()"
    >
      <Icon name="search" size="sm" />
      Try the command palette
      <Kbd>⌘K</Kbd>
    </button>
  </div>
</template>
