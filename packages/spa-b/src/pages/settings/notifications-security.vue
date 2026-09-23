<script setup lang="ts">
/**
 * Notifications & Security settings (`/settings/notifications-security`) —
 * one of the dedicated per-section settings pages from the IA restructure
 * (see src/pages/settings/index.vue). Composes the two independent
 * sub-modules owned by this page: notification rules and vault/security.
 * This page only owns the route shell (back link, title) plus the divider
 * between the two stacked sections.
 */
import { Icon } from '@shared/ui/design-system'
import { NotificationRulesSection } from '@modules/notifications'
import { SecurityCard } from '@modules/vault'
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

    <NotificationRulesSection />

    <hr class="border-line-subtle" />

    <SecurityCard />
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Notifications & Security" } }
</route>
