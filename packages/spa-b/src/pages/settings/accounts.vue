<script setup lang="ts">
/**
 * Accounts settings (`/settings/accounts`) — one of the dedicated
 * per-section settings pages from the IA restructure (see
 * src/pages/settings/index.vue). Renders the Accounts module's own section
 * component; this page only owns the route shell (back link, heading,
 * title).
 */
import { Icon } from '@shared/ui/design-system'
import { AccountsSection } from '@modules/accounts'
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
    <AccountsSection />
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Accounts" } }
</route>
