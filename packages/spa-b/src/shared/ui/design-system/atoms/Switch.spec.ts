import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Switch from './Switch.vue'

describe('Switch', () => {
  it('emits update:modelValue toggled to true when clicked off', async () => {
    const wrapper = mount(Switch, { props: { modelValue: false } })
    await wrapper.find('button[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('emits update:modelValue toggled to false when clicked on', async () => {
    const wrapper = mount(Switch, { props: { modelValue: true } })
    await wrapper.find('button[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false])
  })
})
