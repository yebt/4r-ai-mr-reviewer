<script setup lang="ts">
/**
 * Runs page. Renders inside AppShell (see src/core/App.vue) — the "Runs"
 * nav item (sidebar/bottom-nav/command palette; see src/core/nav.ts)
 * already routes here. First cut: global list + live polling only (see
 * `RunsListSection`) — no launch modal, no detail route yet.
 */
import { RunsListSection } from '@modules/runs'
</script>

<template>
  <RunsListSection />
</template>

<route lang="json">
{ "meta": { "title": "Runs" } }
</route>
