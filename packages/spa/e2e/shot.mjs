// Render harness: screenshot SPA routes against mocked API data (no backend).
//   Requires the Vite dev server running on :5173.
//   Usage: node e2e/shot.mjs [/route ...]   (defaults to a representative set)
// Writes e2e/shots/<route>__<viewport>.png. Set FLOW_REPO=repoA to preload Flow.
import { chromium } from '@playwright/test'
import { fixtureFor } from './mock-fixtures.mjs'
import { mkdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const BASE = process.env.BASE ?? 'http://localhost:5173'
const shotsDir = fileURLToPath(new URL('./shots/', import.meta.url))
mkdirSync(shotsDir, { recursive: true })

const routes = process.argv.slice(2)
if (routes.length === 0) routes.push('/', '/flow', '/reviews', '/actions')

const viewports = [
  { name: 'desktop', width: 1280, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
]

const browser = await chromium.launch()
try {
  for (const vp of viewports) {
    const ctx = await browser.newContext({
      viewport: { width: vp.width, height: vp.height },
      colorScheme: 'dark',
    })
    // Preload the Flow workspace with a repo and skip any first-run gates.
    await ctx.addInitScript((repo) => {
      localStorage.setItem('flow.repoId', repo)
    }, process.env.FLOW_REPO ?? 'repoA')

    const page = await ctx.newPage()
    // Match only real API calls (pathname starts with /api/), NOT Vite's own
    // module URLs that merely contain "/api/" (e.g. /src/shared/api/client.ts).
    await page.route(
      (url) => url.pathname.startsWith('/api/'),
      async (route) => {
        const url = new URL(route.request().url())
        const pathname = url.pathname.replace(/^\/api/, '')
        const f = fixtureFor(route.request().method(), pathname, url.search)
        await route.fulfill({
          status: f?.status ?? 200,
          contentType: 'application/json',
          body: JSON.stringify(f ? f.json : {}),
        })
      },
    )
    page.on('console', (m) => {
      if (m.type() === 'error') console.log(`  [console.error] ${m.text()}`)
    })

    for (const r of routes) {
      await page.goto(BASE + r, { waitUntil: 'networkidle' }).catch(() => {})
      await page.waitForTimeout(600)
      const slug = r.replace(/[^a-z0-9]+/gi, '_').replace(/^_|_$/g, '') || 'root'
      const out = `${shotsDir}${slug}__${vp.name}.png`
      await page.screenshot({ path: out, fullPage: true })
      console.log('shot', slug, vp.name)
    }
    await ctx.close()
  }
} finally {
  await browser.close()
}
