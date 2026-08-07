import { test, expect } from '@playwright/test'
import { mockApi, preloadFlowRepo } from './support'

// Render tests for the Flow workspace against mocked API data. Chromium only and
// headless so it runs without a display.
test.use({ headless: true })

test.beforeEach(async ({ page }) => {
  await mockApi(page)
  await preloadFlowRepo(page)
})

test('loads a repo with the Needs-you band and open MRs', async ({ page }) => {
  await page.goto('/flow')
  await expect(page.getByRole('button', { name: /web-app/ })).toBeVisible()

  // Attention-first band with its three actionable item types (scoped to the
  // band region — the reviews list below also has an Approve on the held review).
  const band = page.getByRole('region', { name: /Needs you/ })
  await expect(band).toBeVisible()
  await expect(band.getByRole('button', { name: 'Approve' }).first()).toBeVisible()
  await expect(band.getByRole('button', { name: 'Retry' }).first()).toBeVisible()
  await expect(band.getByRole('button', { name: 'Skip' }).first()).toBeVisible()

  // Open MRs are launchable and the reviewed one is cross-linked. (v-show keeps
  // both tabs' MR lists in the DOM, so scope to the first match.)
  await expect(
    page.getByText('Add OpenRouter provider with live model browser').first(),
  ).toBeVisible()
  await expect(page.getByText('reviewed', { exact: false }).first()).toBeVisible()
})

test('Settings tab shows the full webhook config with a reveal', async ({ page }) => {
  await page.goto('/flow')
  await page.getByRole('tab', { name: 'Settings' }).click()

  await expect(page.getByText('Webhook URL')).toBeVisible()
  await expect(page.getByText('Secret token')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Rotate' })).toBeVisible()

  // The secret is masked until revealed.
  await expect(page.getByText('whsec_9f3c1a7b2e4d8065')).toBeHidden()
  await page.getByRole('button', { name: 'Show' }).click()
  await expect(page.getByText('whsec_9f3c1a7b2e4d8065')).toBeVisible()
})

test('Release tab previews the exact next tag before launch', async ({ page }) => {
  await page.goto('/flow')
  await page.getByRole('tab', { name: 'Release' }).click()

  await page.getByRole('button', { name: 'Release to main' }).click()
  // The dry-run preview endpoint is mocked to v1.0.0.
  await expect(page.getByText('Next tag')).toBeVisible()
  await expect(page.getByText('v1.0.0').first()).toBeVisible()
})
