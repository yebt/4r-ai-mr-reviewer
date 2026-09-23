<script setup lang="ts">
/**
 * Settings menu (`/settings`, file-based route: `settings/index.vue`). The
 * IA restructure's entry point — replaces the old unified Tabs shell
 * (settings.vue + SettingsLayout.vue, both removed) with one dedicated page
 * per grouped section, linked from here. Each row routes to its own
 * `/settings/<section>` page (settings/providers.vue, etc.).
 *
 * Styled like the mobile More-drawer rows (AppBottomNav.vue): leading icon,
 * label, muted description, trailing chevron — but as full RouterLinks
 * rather than drawer buttons, since this is itself a destination page, not
 * an overlay.
 */
import { Heading, Icon, Text } from '@shared/ui/design-system'

interface SettingsMenuItem {
  to: string
  label: string
  description: string
  icon: string
}

const sections: SettingsMenuItem[] = [
  { to: '/settings/providers', label: 'Providers', description: 'AI provider connections and API keys.', icon: 'cable' },
  { to: '/settings/accounts', label: 'Accounts', description: 'Connected source-control accounts.', icon: 'user' },
  { to: '/settings/telegram', label: 'Telegram', description: 'Notification delivery targets.', icon: 'send' },
  { to: '/settings/profiles', label: 'Profiles', description: 'Review style guides and profiles.', icon: 'users' },
  { to: '/settings/repos', label: 'Repos', description: 'Manage connected repositories.', icon: 'git-branch' },
  {
    to: '/settings/notifications-security',
    label: 'Notifications & Security',
    description: 'Notification preferences and security settings.',
    icon: 'shield',
  },
]
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <Heading :level="1" size="2xl">Settings</Heading>
      <Text muted>Centralized configuration — providers, accounts, and integrations in one place.</Text>
    </div>

    <nav aria-label="Settings sections" class="flex flex-col gap-1">
      <RouterLink
        v-for="section in sections"
        :key="section.to"
        :to="section.to"
        class="flex min-h-14 items-center gap-3 rounded-md border border-line bg-bg-panel px-3 py-2.5 text-text transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon :name="section.icon" size="sm" class="shrink-0 text-text-muted" />
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-medium">{{ section.label }}</span>
          <span class="block truncate text-xs text-text-muted">{{ section.description }}</span>
        </span>
        <Icon name="chevron-down" size="sm" class="-rotate-90 shrink-0 text-text-muted" />
      </RouterLink>
    </nav>
  </div>
</template>

<route lang="json">
{ "meta": { "title": "Settings" } }
</route>
