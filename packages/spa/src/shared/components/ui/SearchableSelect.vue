<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'

// A searchable single-select styled like a `field-underline` input: the trigger
// shows the current value (or a placeholder), and opening it reveals a search
// box plus a filtered, scrollable option list. Built for long lists where a
// native <select> is painful to scan — e.g. a repository's branches. Mirrors the
// Flow repo-switcher pattern (search + click-away), kept generic and reusable.
const props = defineProps<{
  modelValue: string
  options: string[]
  placeholder?: string
  disabled?: boolean
  ariaLabel?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const open = ref(false)
const query = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return q ? props.options.filter((o) => o.toLowerCase().includes(q)) : props.options
})

function toggle() {
  if (props.disabled) return
  if (open.value) close()
  else openList()
}
function openList() {
  open.value = true
  query.value = ''
  nextTick(() => searchInput.value?.focus())
}
function close() {
  open.value = false
}
function pick(value: string) {
  emit('update:modelValue', value)
  close()
}
function onEnter() {
  const first = filtered.value[0]
  if (first !== undefined) pick(first)
}
</script>

<template>
  <div class="relative">
    <button
      type="button"
      class="field-underline flex items-center gap-2 text-left"
      :class="modelValue ? 'text-ink' : 'text-muted/50'"
      :disabled="disabled"
      aria-haspopup="listbox"
      :aria-expanded="open"
      :aria-label="ariaLabel"
      @click="toggle"
    >
      <span class="min-w-0 flex-1 truncate">{{ modelValue || placeholder || 'Select…' }}</span>
      <span class="i-lucide-chevron-down text-muted shrink-0 text-sm" aria-hidden="true" />
    </button>

    <div
      v-if="open"
      class="border-line bg-surface absolute left-0 right-0 z-30 mt-1 flex max-h-64 flex-col overflow-hidden border shadow-lg shadow-black/40"
    >
      <div class="border-line/60 flex items-center gap-2 border-b px-3">
        <span class="i-lucide-search text-muted shrink-0 text-sm" aria-hidden="true" />
        <input
          ref="searchInput"
          v-model="query"
          type="text"
          class="text-ink placeholder:text-muted/60 w-full bg-transparent py-2.5 text-sm outline-none"
          placeholder="Search…"
          autocomplete="off"
          @keydown.enter.prevent="onEnter"
          @keydown.escape.stop="close"
        />
      </div>
      <ul class="min-h-0 flex-1 overflow-y-auto py-1" role="listbox">
        <li
          v-for="o in filtered"
          :key="o"
          role="option"
          :aria-selected="o === modelValue"
          class="flex cursor-pointer items-center gap-2 border-l-2 px-3 py-2 font-mono text-xs"
          :class="
            o === modelValue
              ? 'border-accent bg-accent/10 text-ink'
              : 'text-muted hover:text-ink border-transparent'
          "
          @click="pick(o)"
        >
          <span class="min-w-0 flex-1 truncate">{{ o }}</span>
        </li>
        <li v-if="filtered.length === 0" class="text-muted px-3 py-3 text-xs">No branches match.</li>
      </ul>
    </div>

    <!-- click-away -->
    <div v-if="open" class="fixed inset-0 z-20" @click="close" />
  </div>
</template>
