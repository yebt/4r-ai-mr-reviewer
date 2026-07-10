<script setup lang="ts">
const links = [
  { to: '/', label: 'Home', icon: 'i-lucide-zap', exact: true },
  { to: '/repos', label: 'Repos', icon: 'i-lucide-folder-git-2' },
  { to: '/reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { to: '/more', label: 'More', icon: 'i-lucide-menu' },
]
</script>

<template>
  <nav
    class="border-line/60 bg-canvas text-muted fixed inset-x-0 bottom-0 z-30 flex border-t md:hidden"
  >
    <RouterLink
      v-for="l in links"
      :key="l.to"
      :to="l.to"
      class="flex flex-1 flex-col items-center gap-1 py-2.5 transition-colors"
      active-class="text-accent"
      :exact-active-class="l.exact ? 'text-accent' : ''"
    >
      <span :class="l.icon" class="text-lg" aria-hidden="true" />
      <span class="text-[0.6rem] tracking-wider uppercase">{{ l.label }}</span>
    </RouterLink>
  </nav>
</template>
