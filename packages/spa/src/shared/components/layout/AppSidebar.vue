<script setup lang="ts">
import { useCommandPalette } from '@shared/composables/useCommandPalette'

const emit = defineEmits<{ navigate: [] }>()

const palette = useCommandPalette()

// Show the platform-correct hotkey hint (⌘ on Apple, Ctrl elsewhere).
const isMac =
  typeof navigator !== 'undefined' && /Mac|iP(hone|ad|od)/.test(navigator.platform || navigator.userAgent)
const hotkeyLabel = isMac ? '⌘K' : 'Ctrl K'

const links = [
  { to: '/', label: 'Quick access', icon: 'i-lucide-zap', exact: true },
  { to: '/flow', label: 'Flow', icon: 'i-lucide-layout-panel-left' },
  { to: '/repos', label: 'Repositories', icon: 'i-lucide-folder-git-2' },
  { to: '/reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { to: '/actions', label: 'Actions', icon: 'i-lucide-git-merge' },
  { to: '/accounts', label: 'Accounts', icon: 'i-lucide-users' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu' },
  { to: '/telegram', label: 'Telegram', icon: 'i-lucide-send' },
  { to: '/profiles', label: 'Profiles', icon: 'i-lucide-feather' },
  { to: '/skills', label: 'Skills', icon: 'i-lucide-book-open' },
  { to: '/settings', label: 'Settings', icon: 'i-lucide-settings' },
]
</script>

<template>
  <aside class="border-line/60 bg-canvas flex w-52 shrink-0 flex-col border-r">
    <div class="px-5 py-5">
      <div class="text-ink font-mono text-sm font-semibold tracking-tight">ai&#8209;reviewer</div>
      <div class="label-mono mt-1">4R quality gate</div>
    </div>
    <div class="px-3 pb-3">
      <button
        type="button"
        class="border-line text-muted hover:text-ink hover:border-ink flex w-full items-center gap-2 border px-3 py-2 text-sm transition-colors"
        aria-keyshortcuts="Meta+K Control+K"
        @click="palette.show()"
      >
        <span class="i-lucide-search shrink-0 text-sm" aria-hidden="true" />
        <span class="flex-1 text-left">Search</span>
        <kbd class="border-line text-muted border px-1 py-0.5 font-mono text-[0.6rem]">{{
          hotkeyLabel
        }}</kbd>
      </button>
    </div>
    <nav class="flex flex-col">
      <RouterLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="text-muted hover:text-ink flex items-center gap-3 border-l-2 border-transparent px-5 py-2.5 text-sm transition-colors"
        active-class="border-accent! text-ink!"
        :exact-active-class="link.exact ? 'border-accent! text-ink!' : ''"
        @click="emit('navigate')"
      >
        <span :class="link.icon" class="text-[0.95rem] opacity-70" aria-hidden="true" />
        {{ link.label }}
      </RouterLink>
    </nav>
  </aside>
</template>
