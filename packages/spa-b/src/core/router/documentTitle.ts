/**
 * Pure formatting for the dynamic `document.title`, set from
 * `router.afterEach` in index.ts. Split into its own module so it's
 * unit-testable (documentTitle.spec.ts) without importing index.ts itself,
 * which eagerly bootstraps the real router (auth guards, NProgress, HMR
 * wiring) as an import-time side effect.
 */
export function formatDocumentTitle(title: string | undefined): string {
  return title ? `${title} · 4R` : '4R'
}
