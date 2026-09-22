import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Button from './Button.vue'

describe('Button', () => {
  it('emits click when enabled', async () => {
    const wrapper = mount(Button, { slots: { default: 'Save' } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
  })

  it('does not emit click when disabled', async () => {
    const wrapper = mount(Button, { props: { disabled: true }, slots: { default: 'Save' } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
    expect(wrapper.attributes('disabled')).toBeDefined()
  })

  it('does not emit click when loading', async () => {
    const wrapper = mount(Button, { props: { loading: true }, slots: { default: 'Save' } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeUndefined()
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.find('svg[role="status"]').exists()).toBe(true)
  })
})
