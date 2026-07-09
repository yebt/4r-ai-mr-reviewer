<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'

const props = defineProps<{ open: boolean; title?: string }>()
const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)

watch(
  () => props.open,
  async (open) => {
    if (open) {
      await nextTick()
      panel.value?.focus()
    }
  },
)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-4"
        @keydown="onKeydown"
      >
        <div class="absolute inset-0 bg-canvas/80 backdrop-blur-sm" @click="emit('close')" />

        <div
          ref="panel"
          tabindex="-1"
          class="relative max-h-[90vh] w-full overflow-y-auto border border-line bg-surface p-6 outline-none sm:max-w-md"
          role="dialog"
          aria-modal="true"
        >
          <div class="mb-5 flex items-start justify-between gap-4">
            <h2 v-if="title" class="section-title">{{ title }}</h2>
            <span v-else aria-hidden="true" />
            <button type="button" class="btn-ghost" aria-label="Close" @click="emit('close')">
              <span class="i-lucide-x text-sm" aria-hidden="true" />
            </button>
          </div>
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
