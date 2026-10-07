<script setup lang="ts">
/**
 * Confirmation-dialog molecule — wraps Reka's AlertDialog primitive family
 * (Root/Trigger/Portal/Overlay/Content/Title/Description/Cancel/Action),
 * token-styled and built on the Button atom. Replaces the inline
 * AlertDialogRoot/.../AlertDialogAction boilerplate duplicated across every
 * module's delete-confirmation flow (see e.g. ProvidersSection.vue) and
 * fixes the hardcoded `text-white` those inline copies used on the confirm
 * button.
 *
 * `pending` wires straight into the confirm Button's `:loading`. A loading
 * Button stays focusable (`aria-disabled`, not native `disabled`) but never
 * emits `click`, so while `pending` is true neither `confirm` fires again nor
 * does Reka's built-in close-on-Action behavior re-run.
 *
 * Focus rescue: callers usually delete the row that owned the trigger, which
 * would strand focus on <body>. When `pending` settles and focus is on
 * <body>, focus moves to the page's <main> landmark. Callers that know a
 * better target (next row, section heading) focus it themselves first.
 *
 * Usage:
 * ```html
 * <ConfirmDialog
 *   title='Delete "demo"?'
 *   description="This removes the item. This cannot be undone."
 *   :pending="store.deleting"
 *   @confirm="handleDelete(item)"
 * >
 *   <template #trigger>
 *     <Button variant="ghost" size="sm" class="text-danger-text hover:bg-danger-bg">Delete</Button>
 *   </template>
 * </ConfirmDialog>
 * ```
 */
import { nextTick, watch } from 'vue'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  AlertDialogTrigger,
} from 'reka-ui'
import Button from '../atoms/Button.vue'

const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    confirmLabel?: string
    /** Confirm action reads as destructive (Button variant `danger`) when true. */
    danger?: boolean
    /** Wired to the confirm Button's `:loading` — disables it and shows a spinner. */
    pending?: boolean
  }>(),
  {
    confirmLabel: 'Delete',
    danger: true,
    pending: false,
  },
)

const emit = defineEmits<{
  confirm: []
}>()

watch(
  () => props.pending,
  (pending, wasPending) => {
    if (!wasPending || pending) return
    void nextTick(() => {
      const active = document.activeElement
      if (active && active !== document.body) return
      const main = document.querySelector<HTMLElement>('main')
      if (!main) return
      if (!main.hasAttribute('tabindex')) main.setAttribute('tabindex', '-1')
      main.focus({ preventScroll: true })
    })
  },
)
</script>

<template>
  <AlertDialogRoot>
    <AlertDialogTrigger as-child>
      <slot name="trigger" />
    </AlertDialogTrigger>
    <AlertDialogPortal>
      <AlertDialogOverlay class="overlay z-20" />
      <AlertDialogContent
        class="fixed left-1/2 top-1/2 z-30 w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <AlertDialogTitle class="text-md font-semibold text-text">{{ title }}</AlertDialogTitle>
        <AlertDialogDescription v-if="description" class="mt-1 text-sm text-text-muted">
          {{ description }}
        </AlertDialogDescription>
        <div class="mt-5 flex justify-end gap-2">
          <AlertDialogCancel as-child>
            <Button variant="ghost" size="sm">Cancel</Button>
          </AlertDialogCancel>
          <AlertDialogAction as-child>
            <Button
              :variant="danger ? 'danger' : 'accent'"
              size="sm"
              :loading="pending"
              @click="emit('confirm')"
            >
              {{ confirmLabel }}
            </Button>
          </AlertDialogAction>
        </div>
      </AlertDialogContent>
    </AlertDialogPortal>
  </AlertDialogRoot>
</template>
