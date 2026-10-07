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
    expect(wrapper.find('svg[role="status"]').exists()).toBe(true)
  })

  it('stays focusable while loading: aria-disabled + aria-busy, no native disabled', () => {
    const wrapper = mount(Button, { props: { loading: true }, slots: { default: 'Save' } })
    expect(wrapper.attributes('disabled')).toBeUndefined()
    expect(wrapper.attributes('aria-disabled')).toBe('true')
    expect(wrapper.attributes('aria-busy')).toBe('true')
  })

  it('keeps native disabled (and no aria-busy) for the disabled prop', () => {
    const wrapper = mount(Button, { props: { disabled: true }, slots: { default: 'Save' } })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.attributes('aria-busy')).toBeUndefined()
  })

  it('does not submit its form while loading (Enter in a field or activating the button)', () => {
    const submitted: Event[] = []
    const form = document.createElement('form')
    form.addEventListener('submit', (event) => {
      event.preventDefault()
      submitted.push(event)
    })
    document.body.appendChild(form)
    const wrapper = mount(Button, { props: { type: 'submit', loading: true }, slots: { default: 'Save' }, attachTo: form })
    // Implicit submission (Enter in a field) and Space/Enter on the button both
    // reach the form through a click on the submit button.
    wrapper.element.click()
    expect(submitted).toHaveLength(0)
    wrapper.unmount()
    form.remove()
  })

  it('still submits its form when not loading', () => {
    const submitted: Event[] = []
    const form = document.createElement('form')
    form.addEventListener('submit', (event) => {
      event.preventDefault()
      submitted.push(event)
    })
    document.body.appendChild(form)
    const wrapper = mount(Button, { props: { type: 'submit' }, slots: { default: 'Save' }, attachTo: form })
    wrapper.element.click()
    expect(submitted).toHaveLength(1)
    wrapper.unmount()
    form.remove()
  })
})
