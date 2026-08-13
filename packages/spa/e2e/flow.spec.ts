import { test, expect } from '@playwright/test'
import { mockApi, preloadFlowRepo } from './support'

// Render tests for the Flow workspace against mocked API data. Chromium only and
// headless so it runs without a display. The route now splits in two: /flow is a
// repo selector and /flow/:repoId is the per-repo workspace.
test.use({ headless: true })

test.beforeEach(async ({ page }) => {
  await mockApi(page)
  await preloadFlowRepo(page)
})

test('the selector lists repos and opens a workspace', async ({ page }) => {
  await page.goto('/flow')

  // Both tracked repos appear as pickable rows; there is no auto-redirect.
  const webApp = page.getByRole('button', { name: /web-app/ })
  await expect(webApp).toBeVisible()
  await expect(page.getByRole('button', { name: /api-service/ })).toBeVisible()

  // Search filters the list by name.
  await page.getByRole('textbox', { name: 'Search repositories' }).fill('web')
  await expect(page.getByRole('button', { name: /api-service/ })).toBeHidden()

  // Picking a repo routes to its workspace URL.
  await webApp.click()
  await expect(page).toHaveURL(/\/flow\/repoA/)
  await expect(page.getByText('https://gitlab.com/acme/web-app')).toBeVisible()
})

test('the workspace shows the sticky tabs and the open MRs', async ({ page }) => {
  await page.goto('/flow/repoA')

  // The three-tab bar is present and MRs is the default tab.
  const tablist = page.getByRole('tablist', { name: 'Workspace' })
  await expect(tablist.getByRole('tab', { name: /MRs/ })).toBeVisible()
  await expect(tablist.getByRole('tab', { name: /Reviews/ })).toBeVisible()
  await expect(tablist.getByRole('tab', { name: /Actions/ })).toBeVisible()

  // The open-MR list renders and the reviewed one is cross-linked.
  await expect(
    page.getByText('Add OpenRouter provider with live model browser'),
  ).toBeVisible()
  await expect(page.getByRole('button', { name: 'Release to main' })).toBeVisible()
})

test('the Review button opens the launch modal with provider/model/mode', async ({ page }) => {
  await page.goto('/flow/repoA')

  // The MR row now carries a single Review button (no inline selects); clicking
  // it opens the launch modal where provider/model/mode are chosen.
  await page.getByRole('button', { name: 'Review !128' }).click()

  const dialog = page.getByRole('dialog', { name: 'Start a review' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText('Add OpenRouter provider with live model browser')).toBeVisible()
  await expect(dialog.getByText('Provider', { exact: true })).toBeVisible()
  await expect(dialog.getByText('Context mode', { exact: true })).toBeVisible()
  // Provider, Model and Context mode selects are present (the fixture provider
  // declares models, so the model field renders as a select too).
  await expect(dialog.getByRole('combobox')).toHaveCount(3)
  await expect(dialog.getByRole('button', { name: 'Start review' })).toBeVisible()

  // Cancel closes it without launching a review (still on the workspace).
  await dialog.getByRole('button', { name: 'Cancel' }).click()
  await expect(dialog).toBeHidden()
  await expect(page).toHaveURL(/\/flow\/repoA/)
})

test('the settings modal shows the full webhook config with a reveal', async ({ page }) => {
  await page.goto('/flow/repoA')
  await page.getByRole('button', { name: 'Repository settings' }).click()

  const dialog = page.getByRole('dialog', { name: 'Repository settings' })
  await expect(dialog.getByText('Webhook URL')).toBeVisible()
  await expect(dialog.getByText('Secret token')).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'Rotate secret token' })).toBeVisible()

  // The secret is masked until revealed.
  await expect(page.getByText('whsec_9f3c1a7b2e4d8065')).toBeHidden()
  await dialog.getByRole('button', { name: 'Show secret token' }).click()
  await expect(page.getByText('whsec_9f3c1a7b2e4d8065')).toBeVisible()
})

test('Release to main previews the exact next tag before launch', async ({ page }) => {
  await page.goto('/flow/repoA')
  await page.getByRole('button', { name: 'Release to main' }).click()

  // Scope to the release dialog — the hidden Actions tab also lists a run titled
  // "Release v1.0.0", so an unscoped v1.0.0 match would resolve there.
  const dialog = page.getByRole('dialog', { name: 'Release to main' })

  // The dry-run preview endpoint is mocked to v1.0.0.
  await expect(dialog.getByText('Next tag')).toBeVisible()
  await expect(dialog.getByText('v1.0.0').first()).toBeVisible()

  // Branch pickers are populated from the repo's branches; the fixture repo is
  // missing 'development', so the modal warns and offers a picker.
  await expect(dialog.getByText(/branch not found/)).toBeVisible()
  await expect(dialog.getByRole('option', { name: 'develop', exact: true }).first()).toBeAttached()
})
