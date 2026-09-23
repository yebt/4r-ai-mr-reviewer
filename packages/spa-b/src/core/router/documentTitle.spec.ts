import { describe, expect, it } from 'vitest'
import { formatDocumentTitle } from './documentTitle'

// The router.afterEach() hook itself just assigns `document.title =
// formatDocumentTitle(to.meta.title)` (see index.ts) — tested here as a
// pure function rather than bootstrapping the full router.
describe('formatDocumentTitle', () => {
  it('suffixes a route title with the app name', () => {
    expect(formatDocumentTitle('Settings')).toBe('Settings · 4R')
  })

  it('falls back to the bare app name when no title is set', () => {
    expect(formatDocumentTitle(undefined)).toBe('4R')
  })
})
