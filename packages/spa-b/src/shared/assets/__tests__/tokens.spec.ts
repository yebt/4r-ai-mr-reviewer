import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC = resolve(process.cwd(), 'src')

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) return sourceFiles(path)
    return /\.(vue|ts)$/.test(entry.name) && !entry.name.includes('.spec.') ? [path] : []
  })
}

// Tailwind silently generates nothing for an unknown color, so a typo'd token
// fails with no error at all: the style is just missing.
const COLOR_UTILITY =
  /(?<![\w/-])(?:bg|text|border(?:-[trblxy])?|ring|outline|fill|divide|placeholder|caret)-((?:bg|text|line|accent|success|warning|danger|info|focus|gray|control)[a-z0-9-]*)(?:\/\d+)?(?![\w-])/g

describe('design tokens', () => {
  it('only uses color utilities whose token is defined in tokens.css', () => {
    const css = readFileSync(join(SRC, 'shared/assets/tokens.css'), 'utf8')
    const defined = new Set([...css.matchAll(/--color-([a-z0-9-]+)\s*:/g)].map((m) => m[1]))

    const unknown = sourceFiles(SRC).flatMap((file) =>
      [...readFileSync(file, 'utf8').matchAll(COLOR_UTILITY)]
        .filter((m) => !defined.has(m[1]))
        .map((m) => `${m[0]} in ${file.slice(SRC.length + 1)}`),
    )

    expect(unknown).toEqual([])
  })
})
