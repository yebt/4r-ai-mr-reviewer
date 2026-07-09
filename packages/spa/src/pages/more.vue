<script setup lang="ts">
import PageHeader from '@shared/components/ui/PageHeader.vue'

const items = [
  { to: '/accounts', label: 'GitLab accounts', icon: 'i-lucide-users', hint: 'Manage GitLab connections' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu', hint: 'Configure models and keys' },
  { to: '/skills', label: '4R review skills', icon: 'i-lucide-book-open', hint: 'Rule sets used by the engine' },
]
</script>

<template>
  <div>
    <PageHeader title="More" />

    <ul class="border-t border-line/50">
      <li v-for="it in items" :key="it.to">
        <RouterLink :to="it.to" class="group flex items-center justify-between gap-4 border-b border-line/50 py-4">
          <div class="flex items-center gap-3">
            <span :class="it.icon" class="text-lg text-muted group-hover:text-ink" aria-hidden="true" />
            <div>
              <div class="text-sm text-ink">{{ it.label }}</div>
              <div class="label-mono mt-0.5">{{ it.hint }}</div>
            </div>
          </div>
          <span class="i-lucide-chevron-right text-muted group-hover:text-accent" aria-hidden="true" />
        </RouterLink>
      </li>
    </ul>
  </div>
</template>
