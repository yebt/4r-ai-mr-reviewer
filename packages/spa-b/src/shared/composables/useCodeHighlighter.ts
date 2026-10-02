import { ref, watchEffect, type Ref } from 'vue'
import { createHighlighterCore, type HighlighterCore } from 'shiki/core'
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript'
import githubLight from 'shiki/themes/github-light.mjs'
import githubDark from 'shiki/themes/github-dark.mjs'
import diffLang from 'shiki/langs/diff.mjs'
import typescriptLang from 'shiki/langs/typescript.mjs'
import jsonLang from 'shiki/langs/json.mjs'
import bashLang from 'shiki/langs/bash.mjs'
import vueLang from 'shiki/langs/vue.mjs'
import htmlLang from 'shiki/langs/html.mjs'
import cssLang from 'shiki/langs/css.mjs'

/**
 * Shiki highlighting, dual-theme, truly fine-grained.
 *
 * `createHighlighterCore` + a small set of STATICALLY imported lang/theme
 * modules + the JS regex engine (`shiki/engine/javascript`, no WASM). The
 * naive `import { createHighlighter } from 'shiki'` entry resolves langs by
 * name through a barrel that contains a dynamic `import()` for every
 * bundled language, so a bundler pulls in ~every language (and the ~600KB
 * oniguruma WASM engine) as separate chunks regardless of which ones are
 * actually requested. Static per-file imports below avoid that: only the
 * modules listed here ever reach the output.
 *
 * Each snippet is highlighted ONCE with both themes (`defaultColor: false`);
 * the result carries `--shiki-light`/`--shiki-dark` CSS variables per token,
 * and the `.shiki` rules in main.css pick the active one off the `.dark`
 * class on <html> — so switching theme never re-highlights.
 */
const THEME_NAMES = { light: 'github-light', dark: 'github-dark' } as const

const LANG_MODULES = {
  diff: diffLang,
  typescript: typescriptLang,
  json: jsonLang,
  bash: bashLang,
  vue: vueLang,
  html: htmlLang,
  css: cssLang,
} as const

export type HighlightLang = keyof typeof LANG_MODULES

let highlighterPromise: Promise<HighlighterCore> | null = null

function getHighlighter() {
  highlighterPromise ??= createHighlighterCore({
    themes: [githubLight, githubDark],
    langs: Object.values(LANG_MODULES),
    engine: createJavaScriptRegexEngine(),
  })
  return highlighterPromise
}

export async function highlightCode(code: string, lang: HighlightLang = 'typescript') {
  const highlighter = await getHighlighter()
  return highlighter.codeToHtml(code, {
    lang,
    themes: THEME_NAMES,
    defaultColor: false,
  })
}

/**
 * Reactive wrapper: re-highlights when `code`/`lang` change, but NOT on
 * theme change (the rendered HTML is theme-agnostic; see above).
 */
export function useHighlightedCode(code: Ref<string>, lang: Ref<HighlightLang> | HighlightLang) {
  const html = ref('')
  const isLoading = ref(true)

  watchEffect(async () => {
    isLoading.value = true
    const resolvedLang = typeof lang === 'string' ? lang : lang.value
    const result = await highlightCode(code.value, resolvedLang)
    html.value = result
    isLoading.value = false
  })

  return { html, isLoading }
}
