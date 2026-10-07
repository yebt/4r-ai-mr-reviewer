import { describe, expect, it } from 'vitest'
import { focusFirstAvailable, neighborId } from './focusAfterRemoval'

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
