import { describe, expect, it } from 'vitest'
import { safeRedirect } from './redirect'

describe('safeRedirect', () => {
  it('accepts an in-app absolute path', () => {
    expect(safeRedirect('/flow')).toBe('/flow')
  })

  it('rejects a protocol-relative path', () => {
    expect(safeRedirect('//evil.example.com')).toBe('/')
  })

  it('rejects the login page itself', () => {
    expect(safeRedirect('/login')).toBe('/')
  })

  it('rejects a relative path', () => {
    expect(safeRedirect('relative/path')).toBe('/')
  })

  it('rejects an empty value', () => {
    expect(safeRedirect('')).toBe('/')
  })
})
