// Helpers to derive repository fields from a pasted GitLab project URL, so the
// user types the URL once and we fill the rest.

export interface ParsedRepoUrl {
  valid: boolean
  origin: string // e.g. https://gitlab.com
  path: string // e.g. group/subgroup/project
  name: string // last path segment, e.g. project
}

/** Parses a GitLab project URL into its origin, namespaced path, and name. */
export function parseRepoUrl(raw: string): ParsedRepoUrl {
  const empty: ParsedRepoUrl = { valid: false, origin: '', path: '', name: '' }
  const trimmed = raw.trim()
  if (!trimmed) return empty
  try {
    const u = new URL(trimmed)
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return empty
    const path = u.pathname.replace(/^\/+|\/+$/g, '').replace(/\.git$/, '')
    if (!path) return empty
    const segments = path.split('/')
    return {
      valid: true,
      origin: u.origin,
      path,
      name: segments[segments.length - 1] ?? '',
    }
  } catch {
    return empty
  }
}

function hostOf(raw: string): string {
  try {
    return new URL(raw).host.toLowerCase()
  } catch {
    return ''
  }
}

/** Returns the id of the account whose base URL host matches, or '' if none. */
export function matchAccountId(
  originOrUrl: string,
  accounts: ReadonlyArray<{ id: string; baseUrl: string }>,
): string {
  const host = hostOf(originOrUrl)
  if (!host) return ''
  return accounts.find((a) => hostOf(a.baseUrl) === host)?.id ?? ''
}
