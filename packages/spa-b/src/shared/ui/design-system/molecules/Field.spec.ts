import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Field from './Field.vue'
import Input from '../atoms/Input.vue'

describe('Field', () => {
  it('associates label, description and error via id + aria-describedby, and sets aria-invalid', () => {
    const wrapper = mount({
      components: { Field, Input },
      props: {},
      template: `
        <Field label="Email" description="We only use this to sign you in" error="Enter a valid email" v-slot="{ id, describedBy, invalid }">
          <Input :id="id" :aria-describedby="describedBy" :aria-invalid="invalid" />
        </Field>
      `,
    })

    const label = wrapper.find('label')
    const input = wrapper.find('input')

    expect(label.attributes('for')).toBe(input.attributes('id'))

    const describedBy = input.attributes('aria-describedby') ?? ''
    const describedByIds = describedBy.split(' ')
    expect(describedByIds).toHaveLength(2)

    const descriptionEl = wrapper.find(`#${describedByIds[0]}`)
    const errorEl = wrapper.find(`#${describedByIds[1]}`)
    expect(descriptionEl.text()).toBe('We only use this to sign you in')
    expect(errorEl.text()).toBe('Enter a valid email')
    expect(errorEl.attributes('role')).toBe('alert')

    expect(input.attributes('aria-invalid')).toBe('true')
  })

  it('omits aria-describedby and sets aria-invalid=false when there is no description or error', () => {
    const wrapper = mount({
      components: { Field, Input },
      template: `
        <Field label="Email" v-slot="{ id, describedBy, invalid }">
          <Input :id="id" :aria-describedby="describedBy" :aria-invalid="invalid" />
        </Field>
      `,
    })

    const input = wrapper.find('input')
    expect(input.attributes('aria-describedby')).toBeUndefined()
    expect(input.attributes('aria-invalid')).toBe('false')
  })
})
