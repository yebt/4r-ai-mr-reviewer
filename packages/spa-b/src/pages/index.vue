<script setup lang="ts">
/**
 * Home. Renders inside AppShell (see src/core/App.vue). Minimal on
 * purpose — this is a placeholder until the real dashboard lands; the
 * former token-preview content that used to live here moved to
 * `src/pages/design.vue` (dev-only reference route).
 */
import { Heading, Text } from '@shared/ui/design-system'
</script>

<template>
  <div class="flex flex-col items-start gap-4">
    <div class="flex flex-col gap-1">
      <Heading :level="2" size="2xl">Welcome back</Heading>
      <Text muted>The centralized shell is live — more lands here soon.</Text>
    </div>
  </div>
</template>
