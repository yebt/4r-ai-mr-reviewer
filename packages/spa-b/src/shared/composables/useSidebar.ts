import { createSharedComposable, useLocalStorage } from '@vueuse/core'

function useSidebarState() {
  // Persisted so a reload/new tab keeps the user's last choice — same
  // pattern as useColorScheme's theme persistence.
  const collapsed = useLocalStorage('4r-sidebar-collapsed', false)

  function toggle() {
    collapsed.value = !collapsed.value
  }

  return { collapsed, toggle }
}

/**
 * Desktop sidebar collapse state — shared across every call site (the
 * sidebar's own toggle button, and AppShell.vue, which owns the desktop
 * layout and derives the sidebar's rendered width from the same state) via
 * VueUse's `createSharedComposable`, so there's exactly one source of truth
 * regardless of how many components read it.
 */
export const useSidebar = createSharedComposable(useSidebarState)
