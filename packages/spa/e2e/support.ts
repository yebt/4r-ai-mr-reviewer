import type { Page } from '@playwright/test'
import { fixtureFor } from './mock-fixtures.mjs'

// Intercept the SPA's real API calls (pathname starts with /api/) and serve the
// deterministic fixtures, so render tests need no backend or GitLab. Matching by
// pathname prefix (not a '**/api/**' glob) avoids swallowing Vite's own module
// URLs like /src/shared/api/client.ts.
export async function mockApi(page: Page): Promise<void> {
  await page.route(
    (url) => url.pathname.startsWith('/api/'),
    async (route) => {
      const url = new URL(route.request().url())
      const f = fixtureFor(route.request().method(), url.pathname.replace(/^\/api/, ''), url.search)
      await route.fulfill({
        status: f?.status ?? 200,
        contentType: 'application/json',
        body: JSON.stringify(f ? f.json : {}),
      })
    },
  )
}

// Preload the Flow workspace with a repo so /flow renders loaded, not the picker.
export async function preloadFlowRepo(page: Page, repoId = 'repoA'): Promise<void> {
  await page.addInitScript((id) => localStorage.setItem('flow.repoId', id), repoId)
}
