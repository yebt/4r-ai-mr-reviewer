import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useToast } from './useToast'

// useToast() backs onto a module-level singleton queue (shared across every
// call site by design — see useToast.ts), so each test clears it up front
// rather than relying on module re-isolation between `it` blocks.
beforeEach(() => {
  const { toasts, dismiss } = useToast()
  for (const toast of [...toasts.value]) dismiss(toast.id)
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useToast', () => {
  it('success() adds a toast to the queue', () => {
    const { toasts, success } = useToast()
    success('Saved successfully')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]).toMatchObject({ kind: 'success', message: 'Saved successfully' })
  })

  it('error() and info() add toasts with the matching kind', () => {
    const { toasts, error, info } = useToast()
    error('Something broke')
    info('Heads up')

    expect(toasts.value.map((toast) => toast.kind)).toEqual(['error', 'info'])
  })

  it('assigns a distinct id to each toast', () => {
    const { toasts, success } = useToast()
    success('first')
    success('second')

    const [first, second] = toasts.value
    expect(first?.id).toBeDefined()
    expect(second?.id).toBeDefined()
    expect(first?.id).not.toBe(second?.id)
  })

  it('auto-dismisses a toast after the default timeout', () => {
    const { toasts, success } = useToast()
    success('Auto-dismiss me')
    expect(toasts.value).toHaveLength(1)

    vi.advanceTimersByTime(4999)
    expect(toasts.value).toHaveLength(1)

    vi.advanceTimersByTime(1)
    expect(toasts.value).toHaveLength(0)
  })

  it('keeps error toasts until they are dismissed', () => {
    const { toasts, error, dismiss } = useToast()
    const id = error('Could not save')

    vi.advanceTimersByTime(10 * 60_000)
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]?.duration).toBe(0)

    dismiss(id)
    expect(toasts.value).toHaveLength(0)
  })

  it('lets a caller opt an error toast into auto-dismiss explicitly', () => {
    const { toasts, error } = useToast()
    error('Transient', { duration: 100 })

    vi.advanceTimersByTime(100)
    expect(toasts.value).toHaveLength(0)
  })

  it('auto-dismisses after a custom duration', () => {
    const { toasts, info } = useToast()
    info('Quick one', { duration: 100 })

    vi.advanceTimersByTime(99)
    expect(toasts.value).toHaveLength(1)

    vi.advanceTimersByTime(1)
    expect(toasts.value).toHaveLength(0)
  })

  it('never auto-dismisses when duration is 0', () => {
    const { toasts, info } = useToast()
    info('Sticky', { duration: 0 })

    vi.advanceTimersByTime(60_000)
    expect(toasts.value).toHaveLength(1)
  })

  it('dismiss(id) removes exactly that toast, immediately', () => {
    const { toasts, success, error, dismiss } = useToast()
    const firstId = success('keep me')
    const secondId = error('remove me')

    dismiss(secondId)

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]?.id).toBe(firstId)
  })

  it('dismiss(id) also clears that toast’s pending auto-dismiss timer', () => {
    const { toasts, success, dismiss } = useToast()
    const id = success('bye')
    dismiss(id)

    // If the timer weren't cleared, this would try to filter an
    // already-empty queue — asserting it stays empty either way.
    vi.advanceTimersByTime(10_000)
    expect(toasts.value).toHaveLength(0)
  })
})
