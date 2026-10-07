<script setup lang="ts">
/**
 * Wraps Reka UI's `Select` primitive family (SelectRoot/Trigger/Content/
 * Item/...), token-styled. Consumers pass a flat `items` list; for grouped
 * or custom-rendered options, compose Reka's primitives directly instead.
 */
import {
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectIcon,
  SelectPortal,
  SelectContent,
  SelectViewport,
  SelectItem,
  SelectItemText,
  SelectItemIndicator,
} from 'reka-ui'
import Icon from './Icon.vue'

export interface SelectItemOption {
  label: string
  value: string
  disabled?: boolean
}

const {
  id,
  items,
  placeholder = 'Select…',
  disabled,
  'aria-invalid': ariaInvalid,
  'aria-describedby': ariaDescribedby,
} = defineProps<{
  id?: string
  items: SelectItemOption[]
  placeholder?: string
  disabled?: boolean
  'aria-invalid'?: boolean
  'aria-describedby'?: string
}>()

const modelValue = defineModel<string>()
</script>

<template>
  <SelectRoot v-model="modelValue" :disabled="disabled">
    <SelectTrigger
      :id="id"
      :aria-invalid="ariaInvalid"
      :aria-describedby="ariaDescribedby"
      class="flex h-8 w-full items-center justify-between gap-2 rounded-md border border-line-control bg-bg-panel px-2.5 text-sm text-text transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring disabled:cursor-not-allowed disabled:opacity-50 aria-[invalid=true]:border-danger-solid data-[placeholder]:text-text-placeholder"
    >
      <SelectValue :placeholder="placeholder" />
      <SelectIcon as-child>
        <Icon name="chevron-down" size="sm" class="text-text-muted" />
      </SelectIcon>
    </SelectTrigger>
    <SelectPortal>
      <SelectContent
        :side-offset="4"
        position="popper"
        class="z-50 min-w-[var(--reka-select-trigger-width)] overflow-hidden rounded-lg border border-line bg-bg-panel-raised shadow-token-md"
      >
        <SelectViewport class="p-1">
          <SelectItem
            v-for="item in items"
            :key="item.value"
            :value="item.value"
            :disabled="item.disabled"
            class="flex cursor-pointer items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm text-text outline-none data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
          >
            <SelectItemText>{{ item.label }}</SelectItemText>
            <SelectItemIndicator>
              <Icon name="check" size="sm" class="text-accent-text" />
            </SelectItemIndicator>
          </SelectItem>
        </SelectViewport>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>
