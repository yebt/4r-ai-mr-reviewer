/**
 * Focus helpers for lists whose rows can disappear (delete / discard).
 *
 * When the element holding focus is removed from the DOM the browser drops
 * focus to <body>, which strands keyboard and screen-reader users at the top
 * of the page. These helpers pick a sensible place to put focus instead.
 */

/**
 * The id that should receive focus once `removedId` is gone: the row that
 * follows it, otherwise the one before it, otherwise `null` (list now empty
 * or `removedId` not present).
 */
export function neighborId(ids: readonly string[], removedId: string): string | null {
  const index = ids.indexOf(removedId)
  if (index === -1) return null
  return ids[index + 1] ?? ids[index - 1] ?? null
}

/**
 * Focus `target` when it is still attached; otherwise `fallback` (e.g. the
 * section heading, which must carry `tabindex="-1"`). `preventScroll` keeps
 * the viewport where the user was working.
 */
export function focusFirstAvailable(
  target: HTMLElement | null | undefined,
  fallback: HTMLElement | null | undefined,
): void {
  const el = target?.isConnected ? target : fallback
  el?.focus({ preventScroll: true })
}
