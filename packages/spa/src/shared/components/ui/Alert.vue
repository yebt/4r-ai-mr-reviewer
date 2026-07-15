<script setup lang="ts">
import { computed } from 'vue'

type Variant = 'info' | 'ok' | 'warn' | 'danger'

// Compact inline alert: colored left rule + faint tint + leading icon, built from
// the same palette idioms as the rest of the UI (border-line box, token colors).
const props = withDefaults(defineProps<{ variant?: Variant; icon?: string }>(), {
  variant: 'info',
})

const styles: Record<Variant, { border: string; bg: string; text: string; icon: string }> = {
  info: { border: 'border-l-accent', bg: 'bg-accent/5', text: 'text-muted', icon: 'i-lucide-info' },
  ok: { border: 'border-l-ok', bg: 'bg-ok/5', text: 'text-muted', icon: 'i-lucide-circle-check' },
  warn: {
    border: 'border-l-warn',
    bg: 'bg-warn/5',
    text: 'text-ink',
    icon: 'i-lucide-triangle-alert',
  },
  danger: {
    border: 'border-l-danger',
    bg: 'bg-danger/5',
    text: 'text-danger',
    icon: 'i-lucide-circle-alert',
  },
}

const style = computed(() => styles[props.variant])
const iconClass = computed(() => props.icon ?? style.value.icon)
const role = computed(() => (props.variant === 'danger' ? 'alert' : 'status'))
</script>

<template>
  <div
    class="border-line/50 flex items-start gap-2 border border-l-2 px-3 py-2 text-xs"
    :class="[style.border, style.bg, style.text]"
    :role="role"
  >
    <span :class="iconClass" class="mt-0.5 shrink-0 text-sm" aria-hidden="true" />
    <span class="min-w-0"><slot /></span>
  </div>
</template>
