import { describe, expect, it } from 'vitest'
import { confirm, resolveConfirm, useConfirmDialog } from '@shared/composables/useConfirm'

describe('useConfirm', () => {
  it('resolves true when confirmed', async () => {
    const p = confirm({ message: 'Sure?', danger: true })
    const { state } = useConfirmDialog()
    expect(state.open).toBe(true)
    expect(state.confirmText).toBe('Delete') // danger default
    resolveConfirm(true)
    await expect(p).resolves.toBe(true)
    expect(state.open).toBe(false)
  })

  it('resolves false when cancelled', async () => {
    const p = confirm({ message: 'Sure?' })
    resolveConfirm(false)
    await expect(p).resolves.toBe(false)
  })

  it('resolveConfirm is a no-op when closed', () => {
    expect(() => resolveConfirm(true)).not.toThrow()
  })
})
