<script setup lang="ts">
import type { ToastKind } from '@shared/composables/useToast'
import { useToasts } from '@shared/composables/useToast'

const { toasts, dismiss } = useToasts()

const borderClass: Record<ToastKind, string> = {
  success: 'border-ok',
  error: 'border-danger',
  info: 'border-accent',
}
const icon: Record<ToastKind, string> = {
  success: 'i-lucide-check',
  error: 'i-lucide-triangle-alert',
  info: 'i-lucide-info',
}
const iconClass: Record<ToastKind, string> = {
  success: 'text-ok',
  error: 'text-danger',
  info: 'text-accent',
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed right-4 bottom-4 z-50 flex w-72 flex-col gap-2">
      <TransitionGroup name="toast">
        <button
          v-for="t in toasts"
          :key="t.id"
          type="button"
          class="bg-surface flex items-start gap-3 border-l-2 px-4 py-3 text-left text-sm shadow-lg shadow-black/40"
          :class="borderClass[t.kind]"
          @click="dismiss(t.id)"
        >
          <span
            :class="[icon[t.kind], iconClass[t.kind]]"
            class="mt-0.5 shrink-0"
            aria-hidden="true"
          />
          <span class="text-ink">{{ t.message }}</span>
        </button>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
