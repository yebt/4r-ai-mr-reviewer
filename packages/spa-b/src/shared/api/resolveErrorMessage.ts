/**
 * Resolves a human-readable error message from an unknown caught value,
 * falling back to a caller-supplied default when the value isn't an `Error`.
 *
 * Shared across the `modules/*` stores (providers, accounts, telegram,
 * profiles) so every mutation's `onError` handler formats errors the same
 * way instead of re-declaring this one-liner per store.
 */
export function resolveErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback
}
