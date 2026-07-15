<script setup lang="ts">
import { nextTick, onUnmounted, ref, useId, watch } from 'vue'

const props = defineProps<{ open: boolean; title?: string }>()
const emit = defineEmits<{ close: [] }>()

const panel = ref<HTMLElement | null>(null)
const titleId = useId()
// The element focused before opening, so focus returns to it (e.g. the trigger)
// on close — expected screen-reader/keyboard behavior.
let previouslyFocused: HTMLElement | null = null

// Marking the app root inert while the modal is open keeps assistive tech and
// pointer/keyboard focus out of the background. The modal itself is teleported to
// <body> (a sibling of #app), so it stays interactive.
function setBackgroundInert(inert: boolean) {
  const app = document.getElementById('app')
  if (!app) return
  if (inert) app.setAttribute('inert', '')
  else app.removeAttribute('inert')
}

watch(
  () => props.open,
  async (open) => {
    if (open) {
      previouslyFocused = document.activeElement as HTMLElement | null
      setBackgroundInert(true)
      await nextTick()
      panel.value?.focus()
    } else {
      setBackgroundInert(false)
      previouslyFocused?.focus?.()
      previouslyFocused = null
    }
  },
)

// Safety net: never leave the background inert if the modal unmounts while open.
onUnmounted(() => setBackgroundInert(false))

// Visible, focusable descendants of the panel, in DOM order.
function focusable(): HTMLElement[] {
  if (!panel.value) return []
  const selector =
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
  return Array.from(panel.value.querySelectorAll<HTMLElement>(selector)).filter(
    (el) => el.offsetParent !== null,
  )
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close')
    return
  }
  if (e.key !== 'Tab') return

  // Trap Tab within the panel so focus cannot escape into the inert background.
  const items = focusable()
  if (items.length === 0) {
    e.preventDefault()
    panel.value?.focus()
    return
  }
  const first = items[0]!
  const last = items[items.length - 1]!
  const active = document.activeElement
  if (e.shiftKey) {
    if (active === first || !panel.value?.contains(active)) {
      e.preventDefault()
      last.focus()
    }
  } else if (active === last || !panel.value?.contains(active)) {
    e.preventDefault()
    first.focus()
  }
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
        <div class="bg-canvas/80 absolute inset-0 backdrop-blur-sm" @click="emit('close')" />

        <div
          ref="panel"
          tabindex="-1"
          class="border-line bg-surface relative max-h-[90vh] w-full overflow-y-auto border p-6 outline-none sm:max-w-md"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
        >
          <div class="mb-5 flex items-start justify-between gap-4">
            <h2 v-if="title" :id="titleId" class="section-title">{{ title }}</h2>
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
