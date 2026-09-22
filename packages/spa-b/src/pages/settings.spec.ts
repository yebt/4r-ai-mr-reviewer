import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { TabsContent } from 'reka-ui'
import SettingsLayout, { type SettingsSection } from '@shared/ui/design-system/organisms/SettingsLayout.vue'

// Mock the Providers module so the settings shell is tested in isolation from
// provider data-fetching. (@modules/providers now exists on disk, so the mock
// resolves and settings.vue can be mounted for real.)
vi.mock('@modules/providers', () => ({
  ProvidersSection: { name: 'ProvidersSection', template: '<div>providers-panel</div>' },
}))

const sections: SettingsSection[] = [
  { id: 'providers', label: 'Providers', icon: 'lucide:cable' },
  { id: 'accounts', label: 'Accounts', icon: 'lucide:user' },
  { id: 'telegram', label: 'Telegram', icon: 'lucide:send' },
  { id: 'profiles', label: 'Profiles', icon: 'lucide:users' },
  { id: 'repos', label: 'Repos', icon: 'lucide:git-branch' },
  { id: 'notifications-security', label: 'Notifications & Security', icon: 'lucide:shield' },
]

function mountSettingsLayout() {
  return mount(
    {
      components: { SettingsLayout, TabsContent },
      setup() {
        return { sections }
      },
      template: `
        <SettingsLayout v-model="active" :sections="sections">
          <TabsContent value="providers">Providers panel</TabsContent>
          <TabsContent value="accounts">Accounts panel</TabsContent>
          <TabsContent value="telegram">Telegram panel</TabsContent>
          <TabsContent value="profiles">Profiles panel</TabsContent>
          <TabsContent value="repos">Repos panel</TabsContent>
          <TabsContent value="notifications-security">Notifications &amp; Security panel</TabsContent>
        </SettingsLayout>
      `,
      data() {
        return { active: 'providers' }
      },
    },
    { global: { stubs: { Icon: true } } },
  )
}

describe('settings section nav (SettingsLayout)', () => {
  it('shows a tab for every settings section, including Providers', () => {
    const wrapper = mountSettingsLayout()
    const tabs = wrapper.findAll('[role="tab"]')

    expect(tabs).toHaveLength(6)
    expect(tabs.map((tab) => tab.text())).toContain('Providers')
  })

  it('renders the Providers panel by default', () => {
    const wrapper = mountSettingsLayout()

    expect(wrapper.text()).toContain('Providers panel')
  })

  it('switches to the Accounts panel when its tab is activated', async () => {
    const wrapper = mountSettingsLayout()
    const tabs = wrapper.findAll('[role="tab"]')
    const accountsTab = tabs.find((tab) => tab.text() === 'Accounts')

    expect(accountsTab).toBeTruthy()
    // Reka's TabsTrigger activates on `mousedown` (or focus in automatic
    // mode), not `click` — see node_modules/reka-ui/dist/Tabs/TabsTrigger.js.
    await accountsTab!.trigger('mousedown', { button: 0 })

    expect(wrapper.text()).toContain('Accounts panel')
  })
})

describe('settings.vue (unified settings page)', () => {
  it('mounts the real page with all 6 section tabs and the Providers panel by default', async () => {
    const SettingsPage = (await import('./settings.vue')).default
    const wrapper = mount(SettingsPage, { global: { stubs: { Icon: true } } })

    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs).toHaveLength(6)
    expect(tabs.map((tab) => tab.text())).toContain('Providers')
    // The mocked ProvidersSection renders inside the default (providers) panel.
    expect(wrapper.text()).toContain('providers-panel')
  })
})
