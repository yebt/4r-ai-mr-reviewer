/**
 * Accounts feature — DTO and payload types, verified against the real
 * backend contract (`curl -s http://localhost:8082/accounts`). `token` is
 * write-only: the server never returns it on `Account`.
 */

export interface Account {
  id: string
  name: string
  baseUrl: string
  createdAt: string
}

/** `POST /accounts` body. */
export interface CreateAccountPayload {
  name: string
  baseUrl: string
  token: string
}

/** `PATCH /accounts/{id}` body — blank `token` keeps the stored token. */
export interface UpdateAccountPayload {
  name: string
  baseUrl: string
  token: string
}
