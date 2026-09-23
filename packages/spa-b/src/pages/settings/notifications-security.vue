<script setup lang="ts">
/**
 * Notifications & Security settings (`/settings/notifications-security`) —
 * one of the dedicated per-section settings pages from the IA restructure
 * (see src/pages/settings/index.vue). No real UI yet, so this renders the
 * shared "coming soon" stub; this page only owns the route shell (back
 * link, title).
 */
import { Icon } from '@shared/ui/design-system'
import SettingsSectionStub from '@shared/ui/design-system/organisms/SettingsSectionStub.vue'
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
    <SettingsSectionStub
      title="Notifications & Security"
      description="Manage notification preferences and security settings. Coming soon."
      icon="shield"
    />
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Notifications & Security" } }
</route>
