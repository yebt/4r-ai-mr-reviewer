import { describe, expect, it } from 'vitest'
import { matchAccountId, parseRepoUrl } from '@modules/repos/url'

describe('parseRepoUrl', () => {
  it('extracts origin, path and name', () => {
    const p = parseRepoUrl('https://gitlab.com/group/project')
    expect(p).toEqual({ valid: true, origin: 'https://gitlab.com', path: 'group/project', name: 'project' })
  })

  it('handles nested subgroups and .git suffix and trailing slash', () => {
    const p = parseRepoUrl('https://gitlab.example.com/a/b/c/repo.git/')
    expect(p.path).toBe('a/b/c/repo')
    expect(p.name).toBe('repo')
    expect(p.origin).toBe('https://gitlab.example.com')
  })

  it('rejects non-http and garbage', () => {
    expect(parseRepoUrl('not a url').valid).toBe(false)
    expect(parseRepoUrl('ssh://git@host/x.git').valid).toBe(false)
    expect(parseRepoUrl('https://gitlab.com').valid).toBe(false) // no project path
    expect(parseRepoUrl('').valid).toBe(false)
  })
})

describe('matchAccountId', () => {
  const accounts = [
    { id: 'a1', baseUrl: 'https://gitlab.com' },
    { id: 'a2', baseUrl: 'https://gitlab.example.com/' },
  ]

  it('matches by host', () => {
    expect(matchAccountId('https://gitlab.com', accounts)).toBe('a1')
    expect(matchAccountId('https://gitlab.example.com/group/x', accounts)).toBe('a2')
  })

  it('returns empty when no host matches', () => {
    expect(matchAccountId('https://other.com/x', accounts)).toBe('')
    expect(matchAccountId('nonsense', accounts)).toBe('')
  })
})
