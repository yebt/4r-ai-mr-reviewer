import { describe, expect, it } from 'vitest'
import { type CommandItem, filterCommandItems, registerCommandSource, useCommandSources } from './useCommandSources'

function makeItems(count: number): CommandItem[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `item-${i}`,
    label: `Item ${i}`,
    to: `/item/${i}`,
  }))
}

describe('filterCommandItems', () => {
  it('returns at most 5 items when the query is empty', () => {
    expect(filterCommandItems(makeItems(20), '')).toHaveLength(5)
  })

  it('returns at most 5 items for a whitespace-only query', () => {
    expect(filterCommandItems(makeItems(20), '   ')).toHaveLength(5)
  })

  it('matches case-insensitively against the label', () => {
    const items: CommandItem[] = [
      { id: '1', label: 'Acme Widgets', to: '/repo/1' },
      { id: '2', label: 'Other Thing', to: '/repo/2' },
    ]
    expect(filterCommandItems(items, 'acme')).toEqual([items[0]])
  })

  it('matches against hint', () => {
    const items: CommandItem[] = [
      { id: '1', label: 'Widgets', hint: 'group/widgets-repo', to: '/repo/1' },
      { id: '2', label: 'Other', hint: 'group/other', to: '/repo/2' },
    ]
    expect(filterCommandItems(items, 'widgets-repo')).toEqual([items[0]])
  })

  it('matches against keywords without showing them', () => {
    const items: CommandItem[] = [
      { id: '1', label: 'Widgets', to: '/repo/1', keywords: ['https://gitlab.com/group/widgets'] },
    ]
    expect(filterCommandItems(items, 'gitlab.com/group/widgets')).toEqual(items)
  })

  it('caps matches to 8 while typing a term', () => {
    const items: CommandItem[] = makeItems(20).map((item) => ({ ...item, label: 'Match everything' }))
    expect(filterCommandItems(items, 'match')).toHaveLength(8)
  })

  it('returns no items when nothing matches', () => {
    expect(filterCommandItems(makeItems(5), 'nonexistent-term')).toEqual([])
  })
})

describe('registerCommandSource', () => {
  it('adds the source to the shared registry', () => {
    const unregister = registerCommandSource({
      id: 'test-source',
      heading: 'Test',
      items: () => [],
    })

    expect(useCommandSources().get('test-source')?.heading).toBe('Test')

    unregister()
  })

  it('the returned unregister function removes the source', () => {
    const unregister = registerCommandSource({
      id: 'test-source-2',
      heading: 'Test 2',
      items: () => [],
    })

    unregister()

    expect(useCommandSources().has('test-source-2')).toBe(false)
  })

  it('re-registering the same id replaces the previous source', () => {
    registerCommandSource({ id: 'test-source-3', heading: 'First', items: () => [] })
    registerCommandSource({ id: 'test-source-3', heading: 'Second', items: () => [] })

    expect(useCommandSources().get('test-source-3')?.heading).toBe('Second')
    expect(useCommandSources().size).toBeDefined()

    useCommandSources().delete('test-source-3')
  })
})
