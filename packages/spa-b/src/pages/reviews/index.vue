<script setup lang="ts">
/**
 * Reviews list page. Renders inside AppShell (see src/core/App.vue) — the
 * "Reviews" nav item (sidebar/bottom-nav/command palette; see
 * src/core/nav.ts) already routes to /reviews.
 *
 * Was the flat `src/pages/reviews.vue`, moved into this folder alongside
 * `[id].vue` (the detail page) so unplugin-vue-router auto-generates both
 * `/reviews` (this file) and `/reviews/:id` (the sibling) from the same
 * route segment. All data fetching/mutations live in
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
