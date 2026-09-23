<script setup lang="ts">
/**
 * Reviews page. Renders inside AppShell (see src/core/App.vue) — the
 * "Reviews" nav item (sidebar/bottom-nav/command palette; see
 * src/core/nav.ts) already routes to /reviews.
 *
 * First cut: list only (no launch modal, no detail route — those land in a
 * later milestone). All data fetching/mutations live in
 * `modules/reviews/store.ts`; this page just mounts the section.
 */
import { ReviewsListSection } from '@modules/reviews'
</script>

<template>
  <ReviewsListSection />
</template>

<route lang="json">
{ "meta": { "title": "Reviews" } }
</route>
