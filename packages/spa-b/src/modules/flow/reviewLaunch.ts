/**
 * Flow module — pure default-resolution helpers for `ReviewLaunchDialog`.
 * Provider mirrors the old app's `ReviewLaunchModal` default
 * (`repo.providerId || defaultProvider?.id || ''`, see
 * `packages/spa/src/pages/flow/[repoId].vue`), except it also falls back to
 * the account-wide default provider when the repo's saved `providerId` no
 * longer matches a configured provider (e.g. that provider was deleted).
 *
 * Model is a refinement the old modal didn't have (it always started blank):
 * prefer the repo's own saved model when the repo's own provider is the one
 * selected, else fall back to that provider's own default model. Both
 * resolve to `''` ("use the provider's default") when nothing else applies —
 * a valid, meaningful value server-side (see `types.ts#CreateReviewInput`).
 */
import type { Provider } from '@modules/providers/types'
import type { Repo } from '@modules/repos/types'

/** The repo's own provider if it's still configured, else the account-wide default provider, else `''` (no providers at all). */
export function resolveDefaultProviderId(repo: Repo, providers: Provider[]): string {
  if (repo.providerId && providers.some((provider) => provider.id === repo.providerId)) {
    return repo.providerId
  }
  return providers.find((provider) => provider.isDefault)?.id ?? ''
}

/** `repo.model` when `providerId` is the repo's own provider and it has a saved model, else `providerId`'s own default model, else `''`. */
export function resolveDefaultModel(repo: Repo, providerId: string, providers: Provider[]): string {
  if (providerId && providerId === repo.providerId && repo.model) {
    return repo.model
  }
  return providers.find((provider) => provider.id === providerId)?.model ?? ''
}
