/**
 * Telegram feature — DTO and payload types, verified against the real
 * backend contract (`GET /telegram` curled against localhost:8082).
 * `botToken` is write-only: the server never returns it on `TelegramTarget`.
 */

export interface TelegramTarget {
  id: string
  name: string
  chatId: string
  threadId: string
  isDefault: boolean
  isBot: boolean
  createdAt: string
}

/** `POST /telegram` body. */
export interface CreateTelegramTargetPayload {
  name: string
  chatId: string
  threadId: string
  botToken?: string
  isBot?: boolean
}

/** `PUT /telegram/{id}` body — blank `botToken` keeps the stored token. */
export interface UpdateTelegramTargetPayload {
  name: string
  chatId: string
  threadId: string
  botToken?: string
  isBot?: boolean
}

/** Never throws for a reachable-but-rejecting target — always `{ ok, error? }`. */
export interface TestTelegramTargetResult {
  ok: boolean
  error?: string
}
