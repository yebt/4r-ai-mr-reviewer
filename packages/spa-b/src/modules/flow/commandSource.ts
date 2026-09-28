/**
 * Registers the ⌘K command palette's "Repositories" group: lets the user
 * jump straight into a repo's Flow workspace (`/flow/:repoId`) by typing its
 * name, without first opening /flow and picking it from the repo picker.
 *
 * Kept in `modules/flow` (not the shared design system) per the layering
 * rule in CommandPalette.vue — the palette only reads the shared
 * `useCommandSources` registry, never a feature module directly.
 */
import { computed } from 'vue'
import { registerCommandSource } from '@shared/composables/useCommandSources'
import { useReposStore } from '@modules/repos/store'
import type { Repo } from '@modules/repos/types'

/** Strips the `https://gitlab.com/` prefix so the palette shows the short
 * `group/project` path as the item's hint, e.g. "acme/widgets". */
function toRepoHint(url: string): string {
  return url.replace(/^https?:\/\/gitlab\.com\//, '')
}

/**
 * Registers the "Repositories" command source once (call from an app-level
 * setup location, e.g. App.vue's root setup — not per-component, so the
 * repo list is fetched exactly once and stays registered for the app's
 * lifetime). Reading `useReposStore()` here triggers its `['repos']` query
 * even if the user never visits a repos page, so the palette can search
 * repos immediately.
 */
export function useRepoCommandSource() {
  const reposStore = useReposStore()

  const repoItems = computed(() =>
    reposStore.repos.map((repo: Repo) => ({
      id: `repo-${repo.id}`,
      label: repo.name,
      hint: toRepoHint(repo.url),
      icon: 'workflow',
      to: `/flow/${repo.id}`,
      keywords: [repo.url],
    })),
  )

  return registerCommandSource({
    id: 'flow-repos',
    heading: 'Repositories',
    items: () => repoItems.value,
  })
}
