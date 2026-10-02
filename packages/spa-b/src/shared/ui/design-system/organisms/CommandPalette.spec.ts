import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CommandPalette from './CommandPalette.vue'
import { useCommandPalette } from '@shared/composables/useCommandPalette'

// jsdom doesn't implement scrollIntoView; Reka's Combobox listbox calls it
// internally when managing highlight state. Unrelated to the bug under
// test, so it's stubbed here rather than left to throw as an unhandled
// rejection.
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {}
}

// Regression test for the ESC-doesn't-close bug: ComboboxContent's
// force-mount DismissableLayer sat above the Dialog's own layer and
// swallowed Escape before the Dialog ever saw it. Fixed by handling Reka's
// `escapeKeyDown` event on ComboboxContent directly (see CommandPalette.vue).
describe('CommandPalette — Escape closes the dialog', () => {
  let wrapper: ReturnType<typeof mount> | undefined

  beforeEach(() => {
    useCommandPalette().open()
  })

  afterEach(() => {
    wrapper?.unmount()
    useCommandPalette().close()
  })

  it('closes the palette when Escape is pressed', async () => {
    const palette = useCommandPalette()
    expect(palette.isOpen.value).toBe(true)

    // DismissableLayer's escape handling listens on `document`, so the
    // component needs to be attached to a live document for the real
    // Escape keydown to reach it.
    wrapper = mount(CommandPalette, { attachTo: document.body })
    await flushPromises()

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await flushPromises()

    expect(palette.isOpen.value).toBe(false)
  })
})
