<script setup lang="ts">
/**
 * Global ⌘K / Ctrl+K command palette (docs/ui-patterns.md #2): a Reka
 * Combobox (locale-aware `useFilter` filtering, full keyboard nav) inside a
 * Dialog for the modal overlay/focus trap/Escape-to-close. Desktop renders
 * a centered modal; mobile is a full-screen takeover — the same
 * interaction model, CSS-reflowed per the breakpoint (no separate
 * component needed here, unlike the nav/shell structural swap).
 */
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import {
  ComboboxContent,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxRoot,
  ComboboxViewport,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  VisuallyHidden,
  useFilter,
} from 'reka-ui'
import { useColorScheme } from '@shared/composables/useColorScheme'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import { type CommandItem, filterCommandItems, useCommandSources } from '@shared/composables/useCommandSources'
import { type NavItem, navItems } from '@core/nav'
import Icon from '../atoms/Icon.vue'
import Kbd from '../atoms/Kbd.vue'

interface PaletteAction {
  id: string
  label: string
  icon: string
  run: () => void
}

const router = useRouter()
const { colorMode } = useColorScheme()
const { isOpen, close } = useCommandPalette()

const query = ref('')
const inputRef = ref<InstanceType<typeof ComboboxInput>>()

const actions = computed<PaletteAction[]>(() => [
  {
    id: 'toggle-theme',
    label: colorMode.value === 'dark' ? 'Switch to light theme' : 'Switch to dark theme',
    icon: colorMode.value === 'dark' ? 'sun' : 'moon',
    run: () => {
      colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
    },
  },
])

const { contains } = useFilter({ sensitivity: 'base' })

const filteredNavItems = computed<NavItem[]>(() => {
  const term = query.value.trim()
  return term ? navItems.filter((item) => contains(item.label, term)) : navItems
})

const filteredActions = computed<PaletteAction[]>(() => {
  const term = query.value.trim()
  return term ? actions.value.filter((action) => contains(action.label, term)) : actions.value
})

// Feature-contributed groups (e.g. Flow's "Repositories"), registered via
// `registerCommandSource` — kept out of the design system's own imports so
// this component never depends on `@modules/*` (see useCommandSources.ts).
const commandSources = useCommandSources()

const filteredSourceGroups = computed(() =>
  Array.from(commandSources.values())
    .map((source) => ({ source, items: filterCommandItems(source.items(), query.value) }))
    .filter((group) => group.items.length > 0),
)

const resultCount = computed(
  () =>
    filteredNavItems.value.length +
    filteredSourceGroups.value.reduce((total, group) => total + group.items.length, 0) +
    filteredActions.value.length,
)

function goToCommandItem(item: CommandItem) {
  router.push(item.to)
  handleClose()
}

function handleClose() {
  query.value = ''
  close()
}

function goTo(item: NavItem) {
  router.push(item.to)
  handleClose()
}

function runAction(action: PaletteAction) {
  action.run()
  handleClose()
}

watch(isOpen, async (open) => {
  if (!open) return
  query.value = ''
  await nextTick()
  ;(inputRef.value?.$el as HTMLInputElement | undefined)?.focus()
})
</script>

<template>
  <DialogRoot :open="isOpen" @update:open="(value) => (value ? undefined : handleClose())">
    <DialogPortal>
      <DialogOverlay class="overlay z-40" />
      <DialogContent
        class="fixed inset-0 z-50 flex flex-col bg-bg-panel-raised focus:outline-none md:inset-x-auto md:top-[12%] md:bottom-auto md:left-1/2 md:h-auto md:max-h-[70vh] md:w-[min(34rem,calc(100vw-2rem))] md:-translate-x-1/2 md:rounded-xl md:border md:border-line md:shadow-token-lg"
      >
        <VisuallyHidden as-child>
          <DialogTitle>Command palette</DialogTitle>
        </VisuallyHidden>
        <VisuallyHidden as-child>
          <DialogDescription>Search destinations and actions</DialogDescription>
        </VisuallyHidden>

        <ComboboxRoot :ignore-filter="true" class="flex min-h-0 flex-1 flex-col">
          <div class="flex items-center gap-2 border-b border-line px-4 py-3">
            <Icon name="search" size="sm" class="shrink-0 text-text-muted" />
            <ComboboxInput
              ref="inputRef"
              v-model="query"
              placeholder="Search or jump to…"
              class="h-6 min-w-0 flex-1 bg-transparent text-[1rem] text-text outline-none md:text-base placeholder:text-text-placeholder"
            />
            <Kbd class="shrink-0">Esc</Kbd>
          </div>

          <span class="sr-only" role="status" aria-live="polite">{{ resultCount }} results</span>

          <ComboboxContent
            force-mount
            class="min-h-0 flex-1 overflow-y-auto p-2"
            @escape-key-down="handleClose"
          >
            <ComboboxViewport class="flex flex-col gap-3">
              <p v-if="resultCount === 0" class="px-2 py-6 text-center text-sm text-text-muted">
                No results for “{{ query }}”
              </p>

              <ComboboxGroup v-if="filteredNavItems.length">
                <ComboboxLabel class="px-2 py-1 text-xs text-text-muted">Navigate</ComboboxLabel>
                <ComboboxItem
                  v-for="item in filteredNavItems"
                  :key="item.to"
                  :value="item.to"
                  :text-value="item.label"
                  class="flex min-h-11 cursor-pointer items-center gap-2.5 rounded-md px-2.5 text-sm text-text outline-none data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
                  @select="goTo(item)"
                >
                  <Icon :name="item.icon" size="sm" />
                  {{ item.label }}
                </ComboboxItem>
              </ComboboxGroup>

              <ComboboxGroup
                v-for="group in filteredSourceGroups"
                :key="group.source.id"
              >
                <ComboboxLabel class="px-2 py-1 text-xs text-text-muted">{{ group.source.heading }}</ComboboxLabel>
                <ComboboxItem
                  v-for="item in group.items"
                  :key="item.id"
                  :value="item.id"
                  :text-value="item.label"
                  class="flex min-h-11 cursor-pointer items-center gap-2.5 rounded-md px-2.5 text-sm text-text outline-none data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
                  @select="goToCommandItem(item)"
                >
                  <Icon v-if="item.icon" :name="item.icon" size="sm" class="shrink-0" />
                  <span class="min-w-0 flex-1 truncate">{{ item.label }}</span>
                  <span v-if="item.hint" class="max-w-[40%] shrink-0 truncate text-xs text-text-muted">{{
                    item.hint
                  }}</span>
                </ComboboxItem>
              </ComboboxGroup>

              <ComboboxGroup v-if="filteredActions.length">
                <ComboboxLabel class="px-2 py-1 text-xs text-text-muted">Actions</ComboboxLabel>
                <ComboboxItem
                  v-for="action in filteredActions"
                  :key="action.id"
                  :value="action.id"
                  :text-value="action.label"
                  class="flex min-h-11 cursor-pointer items-center gap-2.5 rounded-md px-2.5 text-sm text-text outline-none data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
                  @select="runAction(action)"
                >
                  <Icon :name="action.icon" size="sm" />
                  {{ action.label }}
                </ComboboxItem>
              </ComboboxGroup>
            </ComboboxViewport>
          </ComboboxContent>
        </ComboboxRoot>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
