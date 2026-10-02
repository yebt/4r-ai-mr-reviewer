import { ref } from 'vue'

/** Visual/semantic category — drives ToastHost's icon and color. */
export type ToastKind = 'success' | 'error' | 'info'

export interface ToastOptions {
  /** Auto-dismiss delay in ms. `0` disables auto-dismiss. Defaults to 5000. */
  duration?: number
}

export interface ToastRecord {
  id: string
  kind: ToastKind
  message: string
  duration: number
}

const DEFAULT_DURATION = 5000

// Module-level state: one shared queue for the whole app, regardless of how
// many components call useToast(). ToastHost.vue (mounted once in App.vue)
// renders this queue; every other call site only ever pushes/dismisses.
const toasts = ref<ToastRecord[]>([])
const timers = new Map<string, ReturnType<typeof setTimeout>>()

let nextId = 0
function generateId(): string {
  nextId += 1
  return `toast-${nextId}`
}

function clearTimer(id: string) {
  const timer = timers.get(id)
  if (timer !== undefined) {
    clearTimeout(timer)
    timers.delete(id)
  }
}

/** Remove a toast immediately (manual close, swipe dismiss, or timeout). */
function dismiss(id: string) {
  clearTimer(id)
  toasts.value = toasts.value.filter((toast) => toast.id !== id)
}

function push(kind: ToastKind, message: string, opts?: ToastOptions): string {
  const id = generateId()
  const duration = opts?.duration ?? DEFAULT_DURATION

  toasts.value = [...toasts.value, { id, kind, message, duration }]

  if (duration > 0) {
    timers.set(
      id,
      setTimeout(() => dismiss(id), duration),
    )
  }

  return id
}

/**
 * Global toast queue. `success`/`error`/`info` push a toast and return its
 * id; `dismiss(id)` removes one immediately (used by ToastHost for the
 * close button and swipe-to-dismiss). Toasts auto-dismiss after `duration`
 * ms (default 5000) unless `duration: 0` is passed.
 */
export function useToast() {
  return {
    toasts,
    success: (message: string, opts?: ToastOptions) => push('success', message, opts),
    error: (message: string, opts?: ToastOptions) => push('error', message, opts),
    info: (message: string, opts?: ToastOptions) => push('info', message, opts),
    dismiss,
  }
}
