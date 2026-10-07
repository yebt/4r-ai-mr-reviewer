<script lang="ts">
/**
 * Pure case-insensitive substring filter over `items[].label`. Exported from
 * a plain `<script>` block since `<script setup>` cannot export (mirrors the
 * old app's `ReleaseModal.vue` twin-block pattern) — kept as a standalone
 * function so it's testable independent of Reka's own combobox internals,
 * rather than relying on Reka's built-in (harder to reason about) filter.
 */
import type { SelectItemOption } from './Select.vue'

export function filterComboboxItems(items: SelectItemOption[], query: string): SelectItemOption[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  return items.filter((item) => item.label.toLowerCase().includes(q))
}
</script>

<script setup lang="ts">
/**
 * Searchable single-select combobox — built on Reka's Combobox primitive
 * family (`ComboboxRoot`/`Anchor`/`Input`/`Trigger`/`Content`/`Viewport`/
 * `Item`/`Empty`), styled with the same tokens as `Select.vue` (this is the
 * text-input-driven sibling: pick from `items` by typing to filter, instead
 * of a fixed dropdown list — used by `ReleaseDialog`'s branch pickers, where
 * a repo can have far more branches than fit a plain `Select`).
 *
 * `modelValue` is the selected item's plain string `value` (like `Select`),
 * never a Reka-internal item object. Filtering is our own
 * `filterComboboxItems` (via `ignore-filter` on `ComboboxRoot`, which turns
 * off Reka's built-in filter) driven by `ComboboxInput`'s own `v-model`
 * (the typed search text) — kept separate from the root's `v-model`
 * (the selected value) since Reka models them independently.
 */
import { computed, ref, watch } from 'vue'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxTrigger,
  ComboboxViewport,
} from 'reka-ui'
import Icon from './Icon.vue'
// `SelectItemOption` is already imported above in the plain `<script>` block
// — both blocks share one module scope, so importing it again here would be
// a duplicate identifier.

const {
  id,
  items,
  placeholder = 'Search…',
  disabled,
  'aria-label': ariaLabel,
  'aria-invalid': ariaInvalid,
  'aria-describedby': ariaDescribedby,
} = defineProps<{
  id?: string
  items: SelectItemOption[]
  placeholder?: string
  disabled?: boolean
  'aria-label'?: string
  'aria-invalid'?: boolean
  'aria-describedby'?: string
}>()

const modelValue = defineModel<string>()

// The typed search text — Reka's own ComboboxInput v-model, distinct from
// `modelValue` (the selected value). Cleared whenever the dialog's selection
// changes from elsewhere (e.g. a default fill), so the input never keeps
// showing stale search text once a value settles.
const searchTerm = ref('')
watch(
  () => modelValue.value,
  () => {
    searchTerm.value = ''
  },
)

const filteredItems = computed(() => filterComboboxItems(items, searchTerm.value))

function displayValue(value: unknown): string {
  return items.find((item) => item.value === value)?.label ?? ''
}
</script>

<template>
  <ComboboxRoot v-model="modelValue" :disabled="disabled" ignore-filter class="relative">
    <ComboboxAnchor
      class="flex h-8 w-full items-center gap-1.5 rounded-md border border-line-control bg-bg-panel px-2.5 transition-colors focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-focus-ring has-[[aria-invalid=true]]:border-danger-solid"
    >
      <ComboboxInput
        :id="id"
        v-model="searchTerm"
        :placeholder="placeholder"
        :display-value="displayValue"
        :disabled="disabled"
        :aria-label="ariaLabel"
        :aria-invalid="ariaInvalid"
        :aria-describedby="ariaDescribedby"
        class="h-full min-w-0 flex-1 border-none bg-transparent text-sm text-text outline-none placeholder:text-text-placeholder disabled:cursor-not-allowed"
      />
      <ComboboxTrigger class="shrink-0" :aria-label="ariaLabel ? `${ariaLabel} options` : 'Show options'">
        <Icon name="chevron-down" size="sm" class="text-text-muted" />
      </ComboboxTrigger>
    </ComboboxAnchor>

    <ComboboxPortal>
      <ComboboxContent
        :side-offset="4"
        position="popper"
        class="z-50 min-w-[var(--reka-combobox-trigger-width)] overflow-hidden rounded-lg border border-line bg-bg-panel-raised shadow-token-md"
      >
        <ComboboxViewport class="max-h-60 overflow-y-auto p-1">
          <ComboboxEmpty class="px-2 py-1.5 text-sm text-text-muted">No results</ComboboxEmpty>
          <ComboboxItem
            v-for="item in filteredItems"
            :key="item.value"
            :value="item.value"
            :text-value="item.label"
            :disabled="item.disabled"
            class="flex cursor-pointer items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
          >
            <span>{{ item.label }}</span>
            <ComboboxItemIndicator>
              <Icon name="check" size="sm" class="text-accent-text" />
            </ComboboxItemIndicator>
          </ComboboxItem>
        </ComboboxViewport>
      </ComboboxContent>
    </ComboboxPortal>
  </ComboboxRoot>
</template>
