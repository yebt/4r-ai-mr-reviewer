import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Alert from './Alert.vue'
import Icon from '../atoms/Icon.vue'

describe('Alert', () => {
  it('defaults to the info status token classes and icon', () => {
    const wrapper = mount(Alert, { slots: { default: 'Message' } })
    expect(wrapper.attributes('role')).toBe('alert')
    expect(wrapper.classes()).toEqual(expect.arrayContaining(['bg-info-bg', 'text-info-text']))
    expect(wrapper.attributes('data-status')).toBe('info')
  })

  it.each([
    ['success', 'bg-success-bg', 'text-success-text', 'circle-check'],
    ['warning', 'bg-warning-bg', 'text-warning-text', 'triangle-alert'],
    ['danger', 'bg-danger-bg', 'text-danger-text', 'circle-x'],
    ['info', 'bg-info-bg', 'text-info-text', 'info'],
    ['neutral', 'bg-bg-hover', 'text-text-muted', 'info'],
  ] as const)('renders the %s status token classes and icon', (status, bgClass, textClass, iconName) => {
    const wrapper = mount(Alert, { props: { status }, slots: { default: 'Message' } })
    expect(wrapper.classes()).toEqual(expect.arrayContaining([bgClass, textClass]))
    expect(wrapper.attributes('data-status')).toBe(status)
    expect(wrapper.findComponent(Icon).props('name')).toBe(iconName)
  })

  it('renders the optional title', () => {
    const wrapper = mount(Alert, {
      props: { title: 'Heads up' },
      slots: { default: 'Message' },
    })
    expect(wrapper.text()).toContain('Heads up')
    expect(wrapper.text()).toContain('Message')
  })

  it('does not render a dismiss button by default', () => {
    const wrapper = mount(Alert, { slots: { default: 'Message' } })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('emits dismiss when the dismiss button is clicked', async () => {
    const wrapper = mount(Alert, {
      props: { dismissible: true },
      slots: { default: 'Message' },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('dismiss')).toHaveLength(1)
  })
})
