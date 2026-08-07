import { ref } from 'vue'

// Shared open-state for the global command palette (Cmd/Ctrl+K). Module-level so
// the palette component, the hotkey, and any launcher button all drive one state.
const open = ref(false)

export function useCommandPalette() {
  return {
    open,
    show: () => {
      open.value = true
    },
    hide: () => {
      open.value = false
    },
    toggle: () => {
      open.value = !open.value
    },
  }
}
