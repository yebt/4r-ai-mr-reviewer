<script setup lang="ts">
/**
 * Runs placeholder. Renders inside AppShell (see src/core/App.vue) — the
 * "Runs" nav item (sidebar/bottom-nav/command palette; see
 * src/core/nav.ts) already routes to /runs. This keeps that nav item from
 * hitting the 404 catch-all until the real runs list/detail UI lands in a
 * later milestone.
 */
import { Heading, Text } from '@shared/ui/design-system'
</script>

<template>
  <div class="flex flex-col items-start gap-1">
    <Heading :level="2" size="2xl">Runs</Heading>
    <Text muted>The runs list and detail view are coming soon.</Text>
  </div>
</template>
