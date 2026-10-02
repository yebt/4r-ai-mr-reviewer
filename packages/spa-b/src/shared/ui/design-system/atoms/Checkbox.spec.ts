import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Checkbox from './Checkbox.vue'

describe('Checkbox', () => {
  it('emits update:modelValue toggled to true when clicked unchecked', async () => {
    const wrapper = mount(Checkbox, { props: { modelValue: false } })
    await wrapper.find('button[role="checkbox"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('emits update:modelValue toggled to false when clicked checked', async () => {
    const wrapper = mount(Checkbox, { props: { modelValue: true } })
    await wrapper.find('button[role="checkbox"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([false])
  })
})
