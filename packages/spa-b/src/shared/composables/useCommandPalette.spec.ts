import { nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import { useCommandPalette } from './useCommandPalette'

function pressCombo(modifierKey: 'Meta' | 'Control', modifierProp: 'metaKey' | 'ctrlKey') {
  // Real keyboard combos fire the modifier's own keydown first, then the
  // letter with the modifier flag set — useMagicKeys' `meta+k`/`ctrl+k`
  // combos only resolve to true once both underlying refs are true.
  window.dispatchEvent(new KeyboardEvent('keydown', { key: modifierKey, [modifierProp]: true }))
  window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', [modifierProp]: true }))
}

function release(modifierKey: 'Meta' | 'Control') {
  window.dispatchEvent(new KeyboardEvent('keyup', { key: 'k' }))
  window.dispatchEvent(new KeyboardEvent('keyup', { key: modifierKey }))
}

describe('useCommandPalette', () => {
  beforeEach(() => {
    useCommandPalette().close()
  })

  it('starts closed', () => {
    expect(useCommandPalette().isOpen.value).toBe(false)
  })

  it('open() sets isOpen to true', () => {
    const palette = useCommandPalette()
    palette.open()
    expect(palette.isOpen.value).toBe(true)
  })

  it('close() sets isOpen to false', () => {
    const palette = useCommandPalette()
    palette.open()
    palette.close()
    expect(palette.isOpen.value).toBe(false)
  })

  it('toggle() flips isOpen', () => {
    const palette = useCommandPalette()
    expect(palette.isOpen.value).toBe(false)
    palette.toggle()
    expect(palette.isOpen.value).toBe(true)
    palette.toggle()
    expect(palette.isOpen.value).toBe(false)
  })

  it('is shared across every call site (singleton state)', () => {
    const a = useCommandPalette()
    const b = useCommandPalette()
    a.open()
    expect(b.isOpen.value).toBe(true)
  })

  it('opens on Meta+K (⌘K)', async () => {
    const palette = useCommandPalette()
    pressCombo('Meta', 'metaKey')
    await nextTick()
    expect(palette.isOpen.value).toBe(true)
    release('Meta')
  })

  it('opens on Ctrl+K', async () => {
    const palette = useCommandPalette()
    pressCombo('Control', 'ctrlKey')
    await nextTick()
    expect(palette.isOpen.value).toBe(true)
    release('Control')
  })

  it('toggles closed on a second ⌘K (holding Meta, tapping K twice)', async () => {
    const palette = useCommandPalette()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Meta', metaKey: true }))
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))
    await nextTick()
    expect(palette.isOpen.value).toBe(true)

    // Release just the letter, then tap it again while Meta is still held —
    // the combo must re-trigger `toggle` on the second rising edge too. A
    // `nextTick` between the two events matters: without it, Vue's batched
    // watcher would only see the net true->true change and never fire.
    window.dispatchEvent(new KeyboardEvent('keyup', { key: 'k', metaKey: true }))
    await nextTick()
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))
    await nextTick()
    expect(palette.isOpen.value).toBe(false)

    release('Meta')
  })
})
