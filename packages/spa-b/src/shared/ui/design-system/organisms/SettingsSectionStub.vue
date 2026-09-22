<script setup lang="ts">
/**
 * "Coming soon" placeholder panel for a settings section that hasn't been
 * built yet. Used by every stub section in the unified Settings view
 * (src/pages/settings.vue) until each one's real UI lands in a later
 * milestone. See DESIGN.md's atomic-design layering — this is an organism:
 * a distinct, self-contained section of a screen, no route awareness.
 */
import Heading from '../atoms/Heading.vue'
import Text from '../atoms/Text.vue'
import Icon from '../atoms/Icon.vue'

withDefaults(
  defineProps<{
    /** Section name, e.g. "Accounts", "Telegram". */
    title: string
    /** One-line explanation of what will live here. */
    description?: string
    /** Lucide icon name (`lucide:*`). */
    icon?: string
  }>(),
  {
    description: 'This section is coming soon.',
    icon: 'lucide:hammer',
  },
)
</script>

<template>
  <div
    class="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-line px-6 py-16 text-center"
  >
    <div
      class="grid size-10 place-items-center rounded-full bg-bg-hover text-text-muted"
      aria-hidden="true"
    >
      <Icon :name="icon" size="lg" />
    </div>
    <Heading :level="2" size="lg">{{ title }}</Heading>
    <Text muted class="max-w-sm">{{ description }}</Text>
  </div>
</template>
