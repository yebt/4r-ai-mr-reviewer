<script setup lang="ts">
/**
 * Unified Settings shell — the centralized configuration view (`/settings`,
 * file-based route). Today's product scatters config across six separate
 * pages; this replaces them with ONE view and a section nav (SettingsLayout,
 * Reka UI Tabs) switching between them. Renders inside AppShell (see
 * src/core/App.vue) — the app-level "Settings" nav item already routes
 * here (src/core/nav.ts).
 *
 * Providers is the only live section today (owned by the Providers-module
 * writer, `@modules/providers`); the rest render SettingsSectionStub until
 * their real UI lands in a later milestone.
 */
import { ref } from 'vue'
import { TabsContent } from 'reka-ui'
import { Heading, Text } from '@shared/ui/design-system'
import SettingsLayout, {
  type SettingsSection,
} from '@shared/ui/design-system/organisms/SettingsLayout.vue'
import SettingsSectionStub from '@shared/ui/design-system/organisms/SettingsSectionStub.vue'
import { ProvidersSection } from '@modules/providers'

const sections: SettingsSection[] = [
  { id: 'providers', label: 'Providers', icon: 'lucide:cable' },
  { id: 'accounts', label: 'Accounts', icon: 'lucide:user' },
  { id: 'telegram', label: 'Telegram', icon: 'lucide:send' },
  { id: 'profiles', label: 'Profiles', icon: 'lucide:users' },
  { id: 'repos', label: 'Repos', icon: 'lucide:git-branch' },
  { id: 'notifications-security', label: 'Notifications & Security', icon: 'lucide:shield' },
]

const activeSection = ref<string>(sections[0]?.id ?? 'providers')
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="flex flex-col gap-1">
      <Heading :level="1" size="2xl">Settings</Heading>
      <Text muted>Centralized configuration — providers, accounts, and integrations in one place.</Text>
    </div>

    <SettingsLayout v-model="activeSection" :sections="sections">
      <TabsContent value="providers">
        <ProvidersSection />
      </TabsContent>
      <TabsContent value="accounts">
        <SettingsSectionStub
          title="Accounts"
          description="Manage connected accounts and identities. Coming soon."
          icon="lucide:user"
        />
      </TabsContent>
      <TabsContent value="telegram">
        <SettingsSectionStub
          title="Telegram"
          description="Connect Telegram for review notifications. Coming soon."
          icon="lucide:send"
        />
      </TabsContent>
      <TabsContent value="profiles">
        <SettingsSectionStub
          title="Profiles"
          description="Manage review profiles and defaults. Coming soon."
          icon="lucide:users"
        />
      </TabsContent>
      <TabsContent value="repos">
        <SettingsSectionStub
          title="Repos"
          description="Manage connected repositories. Coming soon."
          icon="lucide:git-branch"
        />
      </TabsContent>
      <TabsContent value="notifications-security">
        <SettingsSectionStub
          title="Notifications & Security"
          description="Manage notification preferences and security settings. Coming soon."
          icon="lucide:shield"
        />
      </TabsContent>
    </SettingsLayout>
  </div>
</template>
