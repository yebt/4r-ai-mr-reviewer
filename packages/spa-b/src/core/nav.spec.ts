import { describe, expect, it } from 'vitest'
import { filterNavItems, isNavItemActive, navItems, primaryNavItems, secondaryNavItems } from './nav'

describe('nav model', () => {
  it('splits into primary and secondary sections', () => {
    expect(primaryNavItems.length + secondaryNavItems.length).toBe(navItems.length)
    expect(primaryNavItems.every((item) => item.section === 'primary')).toBe(true)
    expect(secondaryNavItems.every((item) => item.section === 'secondary')).toBe(true)
  })

  it('keeps the mobile tab bar within the 3-5 primary destination range', () => {
    expect(primaryNavItems.length).toBeGreaterThanOrEqual(3)
    expect(primaryNavItems.length).toBeLessThanOrEqual(5)
  })
})

describe('isNavItemActive', () => {
  const home = navItems.find((item) => item.to === '/')!
  const reviews = navItems.find((item) => item.to === '/reviews')!

  it('matches the home item only on the exact root path', () => {
    expect(isNavItemActive(home, '/')).toBe(true)
    expect(isNavItemActive(home, '/reviews')).toBe(false)
  })

  it('matches a non-root item on its exact path', () => {
    expect(isNavItemActive(reviews, '/reviews')).toBe(true)
  })

  it('matches a non-root item on nested sub-paths', () => {
    expect(isNavItemActive(reviews, '/reviews/123')).toBe(true)
  })

  it('does not match a different top-level destination', () => {
    expect(isNavItemActive(reviews, '/runs')).toBe(false)
    // A path that merely starts with the same string is not a sub-route.
    expect(isNavItemActive(reviews, '/reviews-archive')).toBe(false)
  })
})

describe('filterNavItems', () => {
  it('returns every item for an empty query', () => {
    expect(filterNavItems(navItems, '')).toEqual(navItems)
    expect(filterNavItems(navItems, '   ')).toEqual(navItems)
  })

  it('filters case-insensitively by label', () => {
    expect(filterNavItems(navItems, 'home')).toEqual([expect.objectContaining({ label: 'Home' })])
    expect(filterNavItems(navItems, 'SETTINGS')).toEqual([
      expect.objectContaining({ label: 'Settings' }),
    ])
  })

  it('matches on a partial substring', () => {
    const result = filterNavItems(navItems, 'run')
    expect(result).toEqual([expect.objectContaining({ label: 'Runs' })])
  })

  it('returns an empty list when nothing matches', () => {
    expect(filterNavItems(navItems, 'zzz-nonexistent')).toEqual([])
  })
})
