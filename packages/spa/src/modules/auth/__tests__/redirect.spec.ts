import { describe, expect, it } from 'vitest'
import { safeRedirect } from '@modules/auth/redirect'

describe('safeRedirect', () => {
  it('returns an in-app absolute path unchanged', () => {
    expect(safeRedirect('/repos')).toBe('/repos')
    expect(safeRedirect('/reviews/42?tab=findings')).toBe('/reviews/42?tab=findings')
  })

  it('falls back to home for missing or empty values', () => {
    expect(safeRedirect(undefined)).toBe('/')
    expect(safeRedirect('')).toBe('/')
    expect(safeRedirect(null)).toBe('/')
  })

  it('rejects external and protocol-relative URLs', () => {
    expect(safeRedirect('https://evil.com')).toBe('/')
    expect(safeRedirect('//evil.com')).toBe('/')
    expect(safeRedirect('relative/path')).toBe('/')
  })

  it('never redirects back to the login page', () => {
    expect(safeRedirect('/login')).toBe('/')
    expect(safeRedirect('/login?redirect=/repos')).toBe('/')
  })

  it('uses the first value when given an array (repeated query param)', () => {
    expect(safeRedirect(['/repos', '/reviews'])).toBe('/repos')
  })

  it('honors a custom fallback', () => {
    expect(safeRedirect(undefined, '/home')).toBe('/home')
  })
})
