import type { AccountProject } from './types'

/**
 * Maps a searched GitLab project to what `RepoForm`'s picker fills on
 * selection: `webUrl` becomes the repo URL, `name` becomes the repo name
 * (only when the form's Name field is still empty — enforced by the caller).
 */
export function projectToDraft(project: AccountProject): { url: string; name: string } {
  return { url: project.webUrl, name: project.name }
}
