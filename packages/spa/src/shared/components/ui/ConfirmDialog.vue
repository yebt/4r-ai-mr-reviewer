<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useConfirmDialog } from '@shared/composables/useConfirm'

const { state, resolveConfirm } = useConfirmDialog()
const confirmButton = ref<HTMLButtonElement | null>(null)

watch(
  () => state.open,
  async (open) => {
    if (open) {
      await nextTick()
      confirmButton.value?.focus()
    }
  },
)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') resolveConfirm(false)
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="state.open"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @keydown="onKeydown"
      >
        <div class="absolute inset-0 bg-canvas/80 backdrop-blur-sm" @click="resolveConfirm(false)" />

        <div
          class="relative w-full max-w-sm border border-line bg-surface p-6"
          role="alertdialog"
          aria-modal="true"
        >
          <h2 v-if="state.title" class="section-title flex items-center gap-2">
            <span
              class="inline-block h-3.5 w-0.5"
              :class="state.danger ? 'bg-danger' : 'bg-accent'"
              aria-hidden="true"
            />
            {{ state.title }}
          </h2>
          <p class="mt-3 text-sm text-muted">{{ state.message }}</p>

          <div class="mt-6 flex justify-end gap-3">
            <button class="btn-line" @click="resolveConfirm(false)">{{ state.cancelText }}</button>
            <button
              ref="confirmButton"
              :class="state.danger ? 'btn-danger-solid' : 'btn-accent'"
              @click="resolveConfirm(true)"
            >
              {{ state.confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
