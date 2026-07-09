import { reactive } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

// Module-level singleton state, read by the single <ConfirmDialog> host mounted
// in the app shell and driven by confirm() from any call site.
const state = reactive({
  open: false,
  title: '',
  message: '',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  danger: false,
})

let resolver: ((ok: boolean) => void) | null = null

/** Opens the confirm dialog and resolves true (confirmed) or false (cancelled). */
export function confirm(options: ConfirmOptions): Promise<boolean> {
  state.title = options.title ?? ''
  state.message = options.message
  state.danger = options.danger ?? false
  state.confirmText = options.confirmText ?? (state.danger ? 'Delete' : 'Confirm')
  state.cancelText = options.cancelText ?? 'Cancel'
  state.open = true
  return new Promise((resolve) => {
    resolver = resolve
  })
}

/** Used by the dialog host to answer and close. */
export function resolveConfirm(ok: boolean) {
  if (!state.open) return
  state.open = false
  resolver?.(ok)
  resolver = null
}

export function useConfirmDialog() {
  return { state, resolveConfirm }
}
