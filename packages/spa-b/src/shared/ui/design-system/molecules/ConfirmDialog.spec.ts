import { describe, expect, it } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ConfirmDialog from './ConfirmDialog.vue'

function mountDialog(props: Record<string, unknown> = {}) {
  return mount(ConfirmDialog, {
    props: {
      title: 'Delete "demo"?',
      description: 'This removes the item. This cannot be undone.',
      ...props,
    },
    slots: {
      trigger: '<button type="button">Delete</button>',
    },
    global: {
      // Reka's AlertDialogPortal teleports its content to document.body;
      // stub Teleport so the dialog renders inline and stays inside the
      // wrapper, and render its default slot (auto-stubs don't by default).
      stubs: { teleport: true },
      renderStubDefaultSlot: true,
    },
  })
}

async function openDialog(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.find('button').trigger('click') // open via the trigger slot
  await flushPromises() // Reka's Presence mounts the content asynchronously
}

describe('ConfirmDialog', () => {
  it('emits confirm when the confirm button is clicked', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)

    const buttons = wrapper.findAll('button')
    const confirmButton = buttons[buttons.length - 1]!
    expect(confirmButton.text()).toBe('Delete')

    await confirmButton.trigger('click')

    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('shows a spinner and blocks confirm on the confirm button while pending', async () => {
    const wrapper = mountDialog({ pending: true })
    await openDialog(wrapper)

    const buttons = wrapper.findAll('button')
    const confirmButton = buttons[buttons.length - 1]!

    expect(confirmButton.attributes('aria-disabled')).toBe('true')
    expect(confirmButton.find('svg[role="status"]').exists()).toBe(true)

    // A loading Button swallows the click instead of emitting it.
    await confirmButton.trigger('click')
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })
})
