// Sanitize a `?redirect=` query value before navigating to it after login.
// Only same-origin, absolute in-app paths are allowed; anything else (external
// URLs, protocol-relative `//evil.com`, the login page itself, or a missing
// value) falls back to home. This prevents an open-redirect via the login link.
export function safeRedirect(target: unknown, fallback = '/'): string {
  const value = Array.isArray(target) ? target[0] : target
  if (typeof value !== 'string' || value === '') return fallback
  // Must be an app-internal absolute path, not a protocol-relative `//host` URL.
  if (!value.startsWith('/') || value.startsWith('//')) return fallback
  // Avoid bouncing straight back to the login page.
  if (value === '/login' || value.startsWith('/login?') || value.startsWith('/login/')) {
    return fallback
  }
  return value
}
