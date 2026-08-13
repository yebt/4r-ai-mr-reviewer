<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { ActiveFilter, FilterField } from '@shared/components/ui/filter-builder'

// Reusable, presentational Linear-style filter builder. A "+ Filter" button adds
// one chip per field; each chip reads `<field> is <value>` and opens a small
// popover to edit its values (checkbox list when multi, single-select otherwise).
// No business logic: the active filters are a v-model of { key, values } — the
// consumer decides what the keys/values mean.
const props = defineProps<{
  fields: FilterField[]
  modelValue: ActiveFilter[]
}>()

const emit = defineEmits<{ 'update:modelValue': [ActiveFilter[]] }>()

// Which popover is open: '__add' for the field picker, a field key for a chip's
// value editor, or null for none. Only one is ever open at a time.
const open = ref<string | null>(null)
const query = ref('')
// Root element — used to focus the first interactive control when a popover opens
// (there is only ever one open, so a single [data-autofocus] target is safe).
const root = ref<HTMLElement | null>(null)

const fieldByKey = computed(() => new Map(props.fields.map((f) => [f.key, f])))
function fieldOf(key: string): FilterField | undefined {
  return fieldByKey.value.get(key)
}
function labelOf(key: string): string {
  return fieldOf(key)?.label ?? key
}
function activeFor(key: string): ActiveFilter | undefined {
  return props.modelValue.find((f) => f.key === key)
}

// Fields not already added — one chip per field, no duplicates.
const availableFields = computed(() =>
  props.fields.filter((f) => !props.modelValue.some((a) => a.key === f.key)),
)

function toggle(key: string) {
  open.value = open.value === key ? null : key
  query.value = ''
}
function close() {
  open.value = null
}

// Focus the search box (or first option/item) when a popover opens, so keyboard
// users land inside it and Escape/arrow keys work immediately.
watch(open, async (v) => {
  if (!v) return
  await nextTick()
  root.value?.querySelector<HTMLElement>('[data-autofocus]')?.focus()
})

function emitModel(next: ActiveFilter[]) {
  emit('update:modelValue', next)
}

function addField(key: string) {
  const field = fieldOf(key)
  if (!field) return
  // A single-select field with one option (e.g. a boolean toggle) is fully
  // decided on add — set it and skip the value popover.
  const preset = !field.multi && field.options.length === 1 ? [field.options[0]!.value] : []
  emitModel([...props.modelValue, { key, values: preset }])
  open.value = preset.length ? null : key
  query.value = ''
}

function removeField(key: string) {
  emitModel(props.modelValue.filter((f) => f.key !== key))
  if (open.value === key) open.value = null
}

function setValues(key: string, values: string[]) {
  emitModel(props.modelValue.map((f) => (f.key === key ? { key, values } : f)))
}

function selectValue(field: FilterField, value: string) {
  const current = activeFor(field.key)
  if (!current) return
  if (field.multi) {
    const has = current.values.includes(value)
    setValues(field.key, has ? current.values.filter((v) => v !== value) : [...current.values, value])
  } else {
    setValues(field.key, [value])
    open.value = null
  }
}

function reset() {
  emitModel([])
  open.value = null
}

function isSelected(key: string, value: string): boolean {
  return activeFor(key)?.values.includes(value) ?? false
}

// Chip value summary: "any" (no value yet), the single option's label, or
// "any of N" for a multi field with more than one value selected.
function valueText(filter: ActiveFilter): string {
  const field = fieldOf(filter.key)
  if (!field || filter.values.length === 0) return 'any'
  if (field.multi && filter.values.length > 1) return `any of ${filter.values.length}`
  const first = filter.values[0]!
  return field.options.find((o) => o.value === first)?.label ?? first
}

function filteredOptions(field: FilterField) {
  const q = query.value.trim().toLowerCase()
  if (!q) return field.options
  return field.options.filter((o) => o.label.toLowerCase().includes(q))
}

// Only large option lists get a search box; small ones focus the first option.
function hasSearch(field: FilterField): boolean {
  return field.options.length > 6
}

const popoverClass =
  'border-line bg-surface absolute left-0 top-full z-30 mt-1.5 flex max-h-80 w-56 flex-col overflow-hidden border shadow-lg shadow-black/40'
</script>

<template>
  <div ref="root" class="flex flex-wrap items-center gap-2">
    <!-- Active filter chips -->
    <div v-for="f in props.modelValue" :key="f.key" class="relative">
      <div class="border-line bg-surface inline-flex items-stretch border text-xs">
        <button
          type="button"
          class="text-ink hover:bg-muted/10 inline-flex items-center gap-1.5 px-2 py-1 transition-colors"
          aria-haspopup="listbox"
          :aria-expanded="open === f.key"
          :aria-label="`Edit ${labelOf(f.key)} filter`"
          @click="toggle(f.key)"
        >
          <span class="text-muted font-medium">{{ labelOf(f.key) }}</span>
          <span class="text-muted/60">is</span>
          <span class="text-accent font-mono">{{ valueText(f) }}</span>
        </button>
        <button
          type="button"
          class="border-line text-muted hover:text-ink hover:bg-muted/10 border-l px-1.5 transition-colors"
          :aria-label="`Remove ${labelOf(f.key)} filter`"
          @click="removeField(f.key)"
        >
          <span class="i-lucide-x text-sm" aria-hidden="true" />
        </button>
      </div>

      <!-- Value editor popover -->
      <template v-if="open === f.key && fieldOf(f.key)">
        <div
          :class="popoverClass"
          role="listbox"
          :aria-label="`${labelOf(f.key)} values`"
          :aria-multiselectable="fieldOf(f.key)!.multi ? 'true' : 'false'"
          @keydown.escape="close"
        >
          <div v-if="hasSearch(fieldOf(f.key)!)" class="border-line/60 flex items-center gap-2 border-b px-3">
            <span class="i-lucide-search text-muted shrink-0 text-sm" aria-hidden="true" />
            <input
              v-model="query"
              data-autofocus
              type="text"
              class="text-ink placeholder:text-muted/60 w-full bg-transparent py-2.5 text-sm outline-none"
              :placeholder="`Filter ${labelOf(f.key).toLowerCase()}…`"
              autocomplete="off"
              @keydown.escape="close"
            />
          </div>
          <ul class="min-h-0 flex-1 overflow-y-auto py-1">
            <li v-for="(opt, i) in filteredOptions(fieldOf(f.key)!)" :key="opt.value">
              <button
                type="button"
                role="option"
                :aria-selected="isSelected(f.key, opt.value)"
                :data-autofocus="!hasSearch(fieldOf(f.key)!) && i === 0 ? '' : undefined"
                class="flex w-full items-center gap-2 border-l-2 px-3 py-2 text-left text-sm transition-colors"
                :class="
                  isSelected(f.key, opt.value)
                    ? 'border-accent bg-accent/10 text-ink'
                    : 'text-muted hover:text-ink border-transparent'
                "
                @click="selectValue(fieldOf(f.key)!, opt.value)"
              >
                <span
                  v-if="fieldOf(f.key)!.multi"
                  class="shrink-0 text-sm"
                  :class="
                    isSelected(f.key, opt.value)
                      ? 'i-lucide-check-square text-accent'
                      : 'i-lucide-square text-muted/60'
                  "
                  aria-hidden="true"
                />
                <span class="min-w-0 flex-1 truncate">{{ opt.label }}</span>
                <span
                  v-if="!fieldOf(f.key)!.multi && isSelected(f.key, opt.value)"
                  class="i-lucide-check text-accent shrink-0 text-sm"
                  aria-hidden="true"
                />
              </button>
            </li>
            <li v-if="filteredOptions(fieldOf(f.key)!).length === 0" class="text-muted px-3 py-3 text-xs">
              No matches.
            </li>
          </ul>
        </div>
        <div class="fixed inset-0 z-20" @click="close" />
      </template>
    </div>

    <!-- + Filter (field picker) -->
    <div v-if="availableFields.length" class="relative">
      <button
        type="button"
        class="btn-line text-xs"
        aria-haspopup="menu"
        :aria-expanded="open === '__add'"
        aria-label="Add filter"
        @click="toggle('__add')"
      >
        <span class="i-lucide-plus text-sm" aria-hidden="true" />
        Filter
      </button>
      <template v-if="open === '__add'">
        <div :class="popoverClass" role="menu" aria-label="Add a filter" @keydown.escape="close">
          <ul class="min-h-0 flex-1 overflow-y-auto py-1">
            <li v-for="(field, i) in availableFields" :key="field.key">
              <button
                type="button"
                role="menuitem"
                :data-autofocus="i === 0 ? '' : undefined"
                class="text-muted hover:text-ink hover:bg-muted/10 flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors"
                @click="addField(field.key)"
              >
                <span class="i-lucide-filter text-muted shrink-0 text-sm" aria-hidden="true" />
                {{ field.label }}
              </button>
            </li>
          </ul>
        </div>
        <div class="fixed inset-0 z-20" @click="close" />
      </template>
    </div>

    <button
      v-if="props.modelValue.length"
      type="button"
      class="btn-ghost text-xs"
      @click="reset"
    >
      <span class="i-lucide-x text-sm" aria-hidden="true" />
      Reset
    </button>

    <div class="ml-auto flex items-center">
      <slot name="trailing" />
    </div>
  </div>
</template>
