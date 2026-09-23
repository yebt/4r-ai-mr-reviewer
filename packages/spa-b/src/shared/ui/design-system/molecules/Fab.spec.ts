import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Fab from './Fab.vue'
import Icon from '../atoms/Icon.vue'

describe('Fab', () => {
  it('renders the plus icon by default', () => {
    const wrapper = mount(Fab, { props: { label: 'Add provider' } })
    expect(wrapper.findComponent(Icon).props('name')).toBe('plus')
  })

  it('applies the label as the accessible name', () => {
    const wrapper = mount(Fab, { props: { label: 'Add provider' } })
    expect(wrapper.attributes('aria-label')).toBe('Add provider')
  })

  it('emits click when pressed', async () => {
    const wrapper = mount(Fab, { props: { label: 'Add provider' } })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })

  it('is mobile-only (md:hidden)', () => {
    const wrapper = mount(Fab, { props: { label: 'Add provider' } })
    expect(wrapper.classes()).toContain('md:hidden')
  })
})
