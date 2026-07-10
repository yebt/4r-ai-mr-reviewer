<script setup lang="ts">
import PageHeader from '@shared/components/ui/PageHeader.vue'

const items = [
  {
    to: '/accounts',
    label: 'GitLab accounts',
    icon: 'i-lucide-users',
    hint: 'Manage GitLab connections',
  },
  {
    to: '/providers',
    label: 'AI providers',
    icon: 'i-lucide-cpu',
    hint: 'Configure models and keys',
  },
  {
    to: '/skills',
    label: '4R review skills',
    icon: 'i-lucide-book-open',
    hint: 'Rule sets used by the engine',
  },
]
</script>

<template>
  <div>
    <PageHeader title="More" />

    <ul class="border-line/50 border-t">
      <li v-for="it in items" :key="it.to">
        <RouterLink
          :to="it.to"
          class="group border-line/50 flex items-center justify-between gap-4 border-b py-4"
        >
          <div class="flex items-center gap-3">
            <span
              :class="it.icon"
              class="text-muted group-hover:text-ink text-lg"
              aria-hidden="true"
            />
            <div>
              <div class="text-ink text-sm">{{ it.label }}</div>
              <div class="label-mono mt-0.5">{{ it.hint }}</div>
            </div>
          </div>
          <span
            class="i-lucide-chevron-right text-muted group-hover:text-accent"
            aria-hidden="true"
          />
        </RouterLink>
      </li>
    </ul>
  </div>
</template>
