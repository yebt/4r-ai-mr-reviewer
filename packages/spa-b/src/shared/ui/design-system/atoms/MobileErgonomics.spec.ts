import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Button from './Button.vue'
import Input from './Input.vue'
import Textarea from './Textarea.vue'

// Class-contract guards (jsdom has no layout): iOS Safari zooms the page when a
// focused field renders below 16px, and touch targets under 40px are easy to
// miss. Desktop density (`md:` / `sm:`) must stay as it was.
describe('mobile ergonomics', () => {
  it.each([
    ['Input', () => mount(Input)],
    ['Textarea', () => mount(Textarea)],
  ])('%s renders 16px below md and keeps 13px from md up', (_name, make) => {
    const classes = make().classes()
    expect(classes).toContain('text-[1rem]')
    expect(classes).toContain('md:text-sm')
    expect(classes).not.toContain('text-sm')
  })

  it('sm Button is at least 40px tall below the sm breakpoint and 28px from sm up', () => {
    const classes = mount(Button, { props: { size: 'sm' } }).classes()
    expect(classes).toContain('h-10')
    expect(classes).toContain('sm:h-7')
    expect(classes).not.toContain('h-7')
  })

  it('md and lg Buttons keep their sizes', () => {
    expect(mount(Button, { props: { size: 'md' } }).classes()).toContain('h-8')
    expect(mount(Button, { props: { size: 'lg' } }).classes()).toContain('h-10')
  })
})
