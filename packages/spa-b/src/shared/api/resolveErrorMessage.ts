/**
 * Resolves a user-facing error message from an unknown caught value, falling
 * back to a caller-supplied default.
 *
 * Shared across the `modules/*` stores and forms so every mutation's error
 * handler formats errors the same way. The server's raw text is only passed
 * through when it reads like a deliberate domain message (a clear 4xx such as
 * "A provider with that name already exists"). Server internals — any 5xx,
 * stack traces, file:line locations, "nil pointer", "upstream … error", SQL
 * state codes — are replaced by a plain sentence built on `fallback`, so
 * users never see implementation detail in a toast or banner.
 */

// Text that only makes sense to the people who run the server.
const INTERNAL_PATTERNS: RegExp[] = [
  /nil pointer|invalid memory address|\bpanic\b|goroutine|stack ?trace|segmentation fault/i,
  /\.(go|ts|js|vue|py|rs):\d+/i,
  /\bat \S+ \(.*:\d+(:\d+)?\)/,
  /upstream .*(error|timeout|reset|refused)/i,
  /SQLSTATE|\bsqlite\b|\bpq:|syscall|ECONN\w*|ENOTFOUND|ETIMEDOUT/i,
  /Unexpected non-JSON response/i,
]

function looksInternal(message: string): boolean {
  return INTERNAL_PATTERNS.some((pattern) => pattern.test(message))
}

function statusOf(err: Error): number | undefined {
  const status = (err as { status?: unknown }).status
  return typeof status === 'number' ? status : undefined
}

function isNetworkFailure(err: Error): boolean {
  return err instanceof TypeError && /failed to fetch|networkerror|load failed|network request failed/i.test(err.message)
}

export function resolveErrorMessage(err: unknown, fallback: string): string {
  if (!(err instanceof Error)) return fallback

  if (isNetworkFailure(err)) {
    return `${trimEnd(fallback)}. Can't reach the server — check your connection and try again.`
  }

  const status = statusOf(err)
  const message = err.message.trim()
  if ((status !== undefined && status >= 500) || message === '' || looksInternal(message)) {
    return `${trimEnd(fallback)}. Something went wrong on our side — try again in a moment.`
  }
  return message
}

function trimEnd(text: string): string {
  return text.replace(/[.\s]+$/, '')
}
