import { nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import { useSidebar } from './useSidebar'

// This sandbox's Node/jsdom combo leaves `window.localStorage` undefined
// (a pre-existing environment gap, unrelated to useSidebar.ts — reproduces
// with a bare `window.localStorage.setItem(...)` in any spec here). Give
// the test a minimal in-memory Storage-compatible polyfill so the
// composable's real persistence path (VueUse's useLocalStorage) is still
// exercised for real, rather than skipping the assertion.
function installMemoryStorage(): Storage {
  const store = new Map<string, string>()
  const storage: Storage = {
    get length() {
      return store.size
    },
    clear: () => store.clear(),
    getItem: (key) => store.get(key) ?? null,
    key: (index) => Array.from(store.keys())[index] ?? null,
    removeItem: (key) => {
      store.delete(key)
    },
    setItem: (key, value) => {
      store.set(key, value)
    },
  }
  Object.defineProperty(window, 'localStorage', { value: storage, configurable: true })
  return storage
}

// Installed once, at module load — useSidebar() is a shared composable
// (createSharedComposable), so the first call anywhere captures whichever
// `window.localStorage` exists at that moment via VueUse's useStorage.
// Replacing the object per-test (rather than clearing this same instance)
// would orphan that first capture.
const storage = installMemoryStorage()

// useSidebar() backs onto a module-level shared instance persisted to
// localStorage, so each test resets both before asserting — same approach
// as useCommandPalette.spec.ts.
beforeEach(() => {
  storage.clear()
  useSidebar().collapsed.value = false
})

describe('useSidebar', () => {
  it('starts expanded (not collapsed)', () => {
    expect(useSidebar().collapsed.value).toBe(false)
  })

  it('toggle() flips collapsed', () => {
    const sidebar = useSidebar()
    sidebar.toggle()
    expect(sidebar.collapsed.value).toBe(true)
    sidebar.toggle()
    expect(sidebar.collapsed.value).toBe(false)
  })

  it('is shared across every call site (singleton state)', () => {
    const a = useSidebar()
    const b = useSidebar()
    a.toggle()
    expect(b.collapsed.value).toBe(true)
  })

  it('persists collapsed state to localStorage under the "4r-sidebar-collapsed" key', async () => {
    useSidebar().toggle()
    // VueUse's useLocalStorage writes through a watcher, flushed on the
    // next reactivity tick rather than synchronously on assignment.
    await nextTick()
    expect(window.localStorage.getItem('4r-sidebar-collapsed')).toBe('true')
  })
})
