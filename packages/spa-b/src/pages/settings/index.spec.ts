import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import SettingsIndexPage from './index.vue'

describe('settings/index.vue (Settings menu)', () => {
  it('renders a RouterLink for every settings section', () => {
    const wrapper = mount(SettingsIndexPage, {
      global: { stubs: { Icon: true, RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } } },
    })

    const links = wrapper.findAll('a')
    const hrefs = links.map((link) => link.attributes('href'))

    expect(hrefs).toEqual([
      '/settings/providers',
      '/settings/accounts',
      '/settings/telegram',
      '/settings/profiles',
      '/settings/repos',
      '/settings/notifications-security',
    ])
  })

  it('shows each section label and description', () => {
    const wrapper = mount(SettingsIndexPage, {
      global: { stubs: { Icon: true, RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] } } },
    })

    expect(wrapper.text()).toContain('Providers')
    expect(wrapper.text()).toContain('AI provider connections and API keys.')
    expect(wrapper.text()).toContain('Notifications & Security')
  })
})
