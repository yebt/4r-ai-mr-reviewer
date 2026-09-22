<script setup lang="ts">
/**
 * Reusable section-shell for a "unified settings" screen: a Tabs-driven
 * section nav (Reka UI `TabsRoot`/`TabsList`/`TabsTrigger`) over a caller-
 * supplied set of `TabsContent` panels (default slot). One instance serves
 * both desktop and mobile — the tab strip becomes horizontally scrollable
 * under the primary 768px breakpoint (CSS reflow, not a structural swap;
 * unlike app-level nav, a tab strip isn't in docs/ui-patterns.md #8's
 * structural-swap list). See DESIGN.md's atomic-design layering — this is
 * an organism: a distinct, self-contained section of a screen, no route
 * awareness, consumed by pages (src/pages/settings.vue).
 */
import { TabsRoot, TabsList, TabsTrigger } from 'reka-ui'
import Icon from '../atoms/Icon.vue'

export interface SettingsSection {
  /** Stable id — the Tabs `value` and the section's `TabsContent value`. */
  id: string
  /** Human-readable section name shown in the tab strip. */
  label: string
  /** kebab-case lucide-vue-next icon name shown next to the label. */
  icon: string
}

const props = defineProps<{
  sections: SettingsSection[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function onUpdateModelValue(value: string | number) {
  emit('update:modelValue', String(value))
}
</script>

<template>
  <TabsRoot
    :model-value="props.modelValue"
    orientation="horizontal"
    class="flex flex-col gap-6"
    @update:model-value="onUpdateModelValue"
  >
    <TabsList
      aria-label="Settings sections"
      class="scrollbar-none flex gap-1 overflow-x-auto border-b border-line"
    >
      <TabsTrigger
        v-for="section in props.sections"
        :key="section.id"
        :value="section.id"
        class="flex shrink-0 items-center gap-1.5 whitespace-nowrap border-b-2 border-transparent px-3 py-2.5 text-sm font-medium text-text-muted transition-colors hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring data-[state=active]:border-accent data-[state=active]:text-text"
      >
        <Icon :name="section.icon" size="sm" />
        {{ section.label }}
      </TabsTrigger>
    </TabsList>

    <slot />
  </TabsRoot>
</template>
