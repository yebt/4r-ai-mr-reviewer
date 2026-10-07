<script setup lang="ts">
/**
 * Global toast surface (organism): renders the useToast() queue as Reka
 * Toast primitives inside one shared ToastProvider/ToastViewport. Mounted
 * once, globally, in App.vue (next to CommandPalette) — every other call
 * site only ever calls useToast() to push/dismiss, never touches this
 * component directly.
 *
 * Positioned bottom-right on desktop, bottom-center above the mobile
 * bottom nav (+ safe-area) on mobile — a CSS reflow (Tailwind `md:`), not a
 * separate render branch, since only the viewport's position changes.
 *
 * Card prominence is status-driven (same `bg-{status}-bg` /
 * `border-{status}-solid` vocabulary as the Alert molecule) plus a left
 * accent bar, so a toast reads as relevant at a glance rather than a
 * neutral panel with only a tinted icon. Entrance is an intentional
 * slide/fade (a local `toast-card-in` keyframe); exit stays a passive fade
 * only. Both durations resolve `var(--duration-base)`, which tokens.css
 * already zeroes under `prefers-reduced-motion: reduce` — no separate
 * motion-reduce styling needed here.
 */
import { ToastClose, ToastDescription, ToastProvider, ToastRoot, ToastTitle, ToastViewport } from 'reka-ui'
import { useToast, type ToastKind } from '@shared/composables/useToast'
import Icon from '../atoms/Icon.vue'

const { toasts, dismiss } = useToast()

const kindTitleMap: Record<ToastKind, string> = {
  success: 'Success',
  error: 'Error',
  info: 'Info',
}

const kindIconMap: Record<ToastKind, string> = {
  success: 'circle-check',
  error: 'circle-x',
  info: 'info',
}

const kindIconClassMap: Record<ToastKind, string> = {
  success: 'text-success-text',
  error: 'text-danger-text',
  info: 'text-info-text',
}

// `error` maps onto the shared `danger` status token — the rest of the
// vocabulary (success/info) already lines up with ToastKind 1:1.
const kindCardClassMap: Record<ToastKind, string> = {
  success: 'border-success-solid/30 bg-success-bg',
  error: 'border-danger-solid/30 bg-danger-bg',
  info: 'border-info-solid/30 bg-info-bg',
}

const kindBarClassMap: Record<ToastKind, string> = {
  success: 'bg-success-solid',
  error: 'bg-danger-solid',
  info: 'bg-info-solid',
}

function handleOpenChange(id: string, open: boolean) {
  // Reka flips `open` to false on Escape, the close button, or a swipe
  // dismiss — any of those should remove the toast from the queue too.
  if (!open) dismiss(id)
}
</script>

<template>
  <ToastProvider>
    <ToastRoot
      v-for="toast in toasts"
      :key="toast.id"
      :type="toast.kind === 'error' ? 'foreground' : 'background'"
      :duration="toast.duration > 0 ? toast.duration : Infinity"
      :class="kindCardClassMap[toast.kind]"
      class="pointer-events-auto relative flex w-full items-start gap-2.5 overflow-hidden rounded-lg border p-3 pl-4 text-sm shadow-token-lg transition-[opacity] duration-[var(--duration-base)] ease-[var(--ease-standard)] data-[state=closed]:opacity-0 data-[state=open]:animate-[toast-card-in_var(--duration-base)_var(--ease-standard)] data-[swipe=move]:transition-none"
      @update:open="(open) => handleOpenChange(toast.id, open)"
    >
      <span aria-hidden="true" :class="kindBarClassMap[toast.kind]" class="absolute inset-y-0 left-0 w-1" />
      <Icon
        :name="kindIconMap[toast.kind]"
        size="sm"
        :class="[kindIconClassMap[toast.kind], 'mt-0.5 shrink-0']"
      />
      <div class="min-w-0 flex-1">
        <ToastTitle class="font-medium text-text">{{ kindTitleMap[toast.kind] }}</ToastTitle>
        <ToastDescription class="text-text-muted">{{ toast.message }}</ToastDescription>
      </div>
      <ToastClose
        class="-m-1 shrink-0 rounded-md p-1 text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        aria-label="Dismiss"
      >
        <Icon name="x" size="sm" />
      </ToastClose>
    </ToastRoot>

    <ToastViewport
      class="fixed inset-x-4 z-50 flex flex-col items-stretch gap-2 outline-none [bottom:calc(5rem+env(safe-area-inset-bottom))] md:inset-x-auto md:bottom-6 md:right-6 md:w-96 md:items-end"
    />
  </ToastProvider>
</template>

<style>
/* Deliberately unscoped: Vue renames keyframes declared inside `scoped`
   styles, which would break the `animate-[toast-card-in_...]` arbitrary
   Tailwind utility above (it references this name literally). */
@keyframes toast-card-in {
  from {
    opacity: 0;
    transform: translateY(0.5rem);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
