import { describe, expect, it } from 'vitest'
import { focusFirstAvailable, neighborId, restoreFocusOnClose } from './focusAfterRemoval'

describe('neighborId', () => {
  const ids = ['a', 'b', 'c']

  it('prefers the next row', () => {
    expect(neighborId(ids, 'a')).toBe('b')
    expect(neighborId(ids, 'b')).toBe('c')
  })

  it('falls back to the previous row when the last one is removed', () => {
    expect(neighborId(ids, 'c')).toBe('b')
  })

  it('returns null for a single-row list or an unknown id', () => {
    expect(neighborId(['a'], 'a')).toBeNull()
    expect(neighborId(ids, 'zzz')).toBeNull()
  })
})

describe('focusFirstAvailable', () => {
  it('focuses the target when attached, else the fallback', () => {
    const target = document.createElement('button')
    const fallback = document.createElement('h2')
    fallback.tabIndex = -1
    document.body.append(fallback)

    focusFirstAvailable(target, fallback) // target detached
    expect(document.activeElement).toBe(fallback)

    document.body.append(target)
    focusFirstAvailable(target, fallback)
    expect(document.activeElement).toBe(target)

    target.remove()
    fallback.remove()
  })
})

describe('restoreFocusOnClose', () => {
  it('takes over Reka\'s close-auto-focus and focuses the target', () => {
    const target = document.createElement('button')
    document.body.append(target)
    const event = new Event('closeAutoFocus', { cancelable: true })

    restoreFocusOnClose(event, () => target)

    expect(event.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(target)
    target.remove()
  })

  it('leaves the default behavior alone without a (connected) target', () => {
    const event = new Event('closeAutoFocus', { cancelable: true })
    restoreFocusOnClose(event, undefined)
    restoreFocusOnClose(event, () => null)
    restoreFocusOnClose(event, () => document.createElement('button')) // detached
    expect(event.defaultPrevented).toBe(false)
  })
})
