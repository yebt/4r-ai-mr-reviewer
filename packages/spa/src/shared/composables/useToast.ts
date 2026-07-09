import { reactive } from 'vue'

export type ToastKind = 'success' | 'error' | 'info'

export interface Toast {
  id: number
  kind: ToastKind
  message: string
}

const toasts = reactive<Toast[]>([])
let seq = 0

function push(kind: ToastKind, message: string, timeout = 3500) {
  const id = ++seq
  toasts.push({ id, kind, message })
  window.setTimeout(() => dismiss(id), timeout)
}

function dismiss(id: number) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}

/** Fire a toast from anywhere: `toast.success('Saved')`. */
export const toast = {
  success: (message: string) => push('success', message),
  error: (message: string) => push('error', message),
  info: (message: string) => push('info', message),
}

export function useToasts() {
  return { toasts, dismiss }
}
