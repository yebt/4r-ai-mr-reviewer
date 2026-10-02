import { describe, expect, it } from 'vitest'
import { filterComboboxItems } from './Combobox.vue'
import type { SelectItemOption } from './Select.vue'

const items: SelectItemOption[] = [
  { label: 'development', value: 'development' },
  { label: 'main', value: 'main' },
  { label: 'feature/login', value: 'feature/login' },
]

describe('filterComboboxItems', () => {
  it('returns every item for a blank query', () => {
    expect(filterComboboxItems(items, '')).toEqual(items)
  })

  it('returns every item for a whitespace-only query', () => {
    expect(filterComboboxItems(items, '   ')).toEqual(items)
  })

  it('matches case-insensitively on a substring of the label', () => {
    expect(filterComboboxItems(items, 'DEV')).toEqual([items[0]])
  })

  it('matches a substring anywhere in the label, not just a prefix', () => {
    expect(filterComboboxItems(items, 'login')).toEqual([items[2]])
  })

  it('returns an empty array when nothing matches', () => {
    expect(filterComboboxItems(items, 'nonexistent')).toEqual([])
  })
})
