/**
 * Vault (security/master-key) feature — DTO and payload types, verified
 * against the real backend contract (`GET /vault/status` curled against
 * localhost:8082, auth off).
 */

export interface VaultStatus {
  initialized: boolean
  passwordProtected: boolean
}

/**
 * Sentinel the store's `['vault-status']` query resolves to when the
 * backend answers 501 — vault management is unavailable on this build. Kept
 * as a successful query result (not a thrown error) so the UI treats it as
 * an ordinary "hide the card" state rather than a retry-worthy failure.
 */
export interface VaultUnavailable {
  unavailable: true
}

export type VaultStatusResult = VaultStatus | VaultUnavailable

/** `POST /vault/password` body. Empty `newPassword` switches to key-file mode. */
export interface ChangeVaultPasswordPayload {
  oldPassword: string
  newPassword: string
}

export interface ChangeVaultPasswordResult {
  passwordProtected: boolean
  /** e.g. "update AIR_PASSWORD before the next restart" — surface as a persistent alert, not just a toast. */
  warning?: string
}
