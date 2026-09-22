import { ref } from 'vue'
import { createSharedComposable, useMagicKeys, whenever } from '@vueuse/core'

function useCommandPaletteState() {
  const isOpen = ref(false)

  function open() {
    isOpen.value = true
  }

  function close() {
    isOpen.value = false
  }

  function toggle() {
    isOpen.value = !isOpen.value
  }

  // `passive: false` + preventDefault keeps the browser's own Ctrl+K
  // (location bar focus) and any OS binding from firing alongside ours, per
  // docs/ui-patterns.md #2: "must not conflict with browser/OS bindings".
  const keys = useMagicKeys({
    passive: false,
    onEventFired(event) {
      const isCommandKey = event.metaKey || event.ctrlKey
      if (event.type === 'keydown' && isCommandKey && event.key.toLowerCase() === 'k') {
        event.preventDefault()
      }
    },
  })

  // `noUncheckedIndexedAccess` types combo lookups as possibly `undefined`;
  // a getter + optional chaining keeps this reactive without a non-null
  // assertion (the combo ref is always defined once `useMagicKeys` runs).
  whenever(() => keys['meta+k']?.value, toggle)
  whenever(() => keys['ctrl+k']?.value, toggle)

  return { isOpen, open, close, toggle }
}

/**
 * Global ⌘K / Ctrl+K command-palette state. Shared across every call site
 * (sidebar trigger, mobile search trigger, CommandPalette itself) via
 * VueUse's `createSharedComposable` — one instance, one keybinding, no
 * duplicate `keydown` listeners regardless of how many components use it.
 */
export const useCommandPalette = createSharedComposable(useCommandPaletteState)
