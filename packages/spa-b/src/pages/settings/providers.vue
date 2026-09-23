<script setup lang="ts">
/**
 * Providers settings (`/settings/providers`) — one of the dedicated
 * per-section settings pages from the IA restructure (see
 * src/pages/settings/index.vue). Renders the Providers module's own
 * section component; this page only owns the route shell (back link,
 * heading, title).
 */
import { Icon } from '@shared/ui/design-system'
import { ProvidersSection } from '@modules/providers'
</script>

<template>
  <div class="flex flex-col gap-4">
    <RouterLink
      to="/settings"
      class="inline-flex w-fit items-center gap-1.5 text-sm text-text-muted transition-colors hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
    >
      <Icon name="chevron-down" size="sm" class="rotate-90" />
      Settings
    </RouterLink>
    <ProvidersSection />
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Providers" } }
</route>
