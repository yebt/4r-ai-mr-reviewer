import { useLocalStorage } from '@vueuse/core'

// The repo currently loaded in the Flow workspace, persisted and shared module-
// wide so the Flow page, the repo picker and the command palette all drive one
// selection reactively.
const repoId = useLocalStorage('flow.repoId', '')

export function useFlowRepo() {
  return { repoId }
}
