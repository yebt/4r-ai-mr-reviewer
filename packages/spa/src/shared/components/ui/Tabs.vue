<script lang="ts">
// Exported from a normal <script> block because <script setup> cannot export.
export interface TabItem {
  id: string
  label: string
  icon?: string
}
</script>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ tabs: TabItem[]; modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [id: string] }>()

const list = ref<HTMLElement | null>(null)

function select(id: string) {
  emit('update:modelValue', id)
}

// Roving-focus keyboard support (automatic activation: selection follows focus).
function onKeydown(e: KeyboardEvent, index: number) {
  const last = props.tabs.length - 1
  let next = -1
  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') next = index === last ? 0 : index + 1
  else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') next = index === 0 ? last : index - 1
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = last
  else return

  e.preventDefault()
  const tab = props.tabs[next]
  if (!tab) return
  select(tab.id)
  list.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]')[next]?.focus()
}
</script>

<template>
  <div ref="list" role="tablist" class="border-line/50 flex gap-1 border-b">
    <button
      v-for="(tab, i) in tabs"
      :id="`tab-${tab.id}`"
      :key="tab.id"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.id"
      :aria-controls="`panel-${tab.id}`"
      :tabindex="modelValue === tab.id ? 0 : -1"
      class="-mb-px inline-flex cursor-pointer items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors outline-none"
      :class="
        modelValue === tab.id
          ? 'border-accent text-ink'
          : 'text-muted hover:text-ink border-transparent'
      "
      @click="select(tab.id)"
      @keydown="onKeydown($event, i)"
    >
      <span v-if="tab.icon" :class="tab.icon" class="text-sm" aria-hidden="true" />
      {{ tab.label }}
    </button>
  </div>
</template>
