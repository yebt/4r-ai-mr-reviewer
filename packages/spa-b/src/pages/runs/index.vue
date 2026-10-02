<script setup lang="ts">
/**
 * Runs list page (`/runs`). Renders inside AppShell (see src/core/App.vue) —
 * the "Runs" nav item (sidebar/bottom-nav/command palette; see
 * src/core/nav.ts) already routes here. Was the flat `src/pages/runs.vue`;
 * moved into this folder so `src/pages/runs/[id].vue` can add the detail
 * route (unplugin-vue-router auto-generates `/runs` + `/runs/:id` from this
 * folder). All list behavior (filters, live polling, archive/cancel/delete)
 * lives in `RunsListSection`.
 */
import { RunsListSection } from '@modules/runs'
</script>

<template>
  <RunsListSection />
</template>

<route lang="json">
{ "meta": { "title": "Runs" } }
</route>
