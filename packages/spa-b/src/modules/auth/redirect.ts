/**
 * Validates a `redirect` query param as an in-app absolute path.
 * Rejects protocol-relative ('//host'), the login page itself, relative
 * paths, and empty values — falling back to '/'.
 */
export function safeRedirect(raw: string | null | undefined): string {
  if (!raw) return '/'
  if (!raw.startsWith('/')) return '/'
  if (raw.startsWith('//')) return '/'
  if (raw.startsWith('/login')) return '/'
  return raw
}
