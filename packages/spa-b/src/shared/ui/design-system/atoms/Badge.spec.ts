import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Badge from './Badge.vue'

describe('Badge', () => {
  it('defaults to the neutral status token classes', () => {
    const wrapper = mount(Badge, { slots: { default: 'Open' } })
    expect(wrapper.classes()).toEqual(
      expect.arrayContaining(['bg-bg-hover', 'text-text-muted']),
    )
  })

  it.each([
    ['success', 'bg-success-bg', 'text-success-text'],
    ['warning', 'bg-warning-bg', 'text-warning-text'],
    ['danger', 'bg-danger-bg', 'text-danger-text'],
    ['info', 'bg-info-bg', 'text-info-text'],
  ] as const)('renders the %s status token classes', (status, bgClass, textClass) => {
    const wrapper = mount(Badge, { props: { status }, slots: { default: 'Status' } })
    expect(wrapper.classes()).toEqual(expect.arrayContaining([bgClass, textClass]))
    expect(wrapper.attributes('data-status')).toBe(status)
  })
})
