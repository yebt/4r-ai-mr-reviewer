import { describe, expect, it } from 'vitest'
import { projectToDraft } from './projectSearch'
import type { AccountProject } from './types'

describe('projectToDraft', () => {
  it('maps webUrl to url and name to name', () => {
    const project: AccountProject = {
      id: 42,
      name: 'my-project',
      pathWithNamespace: 'group/my-project',
      webUrl: 'https://gitlab.com/group/my-project',
    }

    expect(projectToDraft(project)).toEqual({
      url: 'https://gitlab.com/group/my-project',
      name: 'my-project',
    })
  })
})
