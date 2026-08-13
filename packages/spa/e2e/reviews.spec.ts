import { test, expect } from '@playwright/test'
import { mockApi } from './support'

test.use({ headless: true })

test.beforeEach(async ({ page }) => {
  await mockApi(page)
})

test('reviews list filters by status and offers retry on failures', async ({ page }) => {
  await page.goto('/reviews')

  // Status filter chips with counts (3 reviews: 1 awaiting, 1 failed, 1 done).
  await expect(page.getByRole('button', { name: /Awaiting/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /Failed/ })).toBeVisible()

  // A failed review is recoverable, not a dead end.
  await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

  // Filtering to Done narrows to the finished review and hides the failed one.
  await page.getByRole('button', { name: /Done/ }).click()
  await expect(page.getByText('!120').first()).toBeVisible()
  await expect(page.getByText('!126')).toBeHidden()
})

test('groups retry/webhook clones of the same MR under the latest', async ({ page }) => {
  await page.goto('/reviews')

  // MR !120 has a done review and an earlier failed attempt: the older one is
  // collapsed behind a toggle, not cluttering the list.
  const toggle = page.getByRole('button', { name: /earlier attempt.* for !120/ })
  await expect(toggle).toBeVisible()
  await expect(page.getByText('timeout cloning repo')).toBeHidden()
  await expect(page.getByRole('link', { name: /!120/ })).toHaveCount(1)
  await toggle.click()
  // Expanding reveals the older attempt's row (a second !120 review link appears).
  await expect(page.getByRole('link', { name: /!120/ })).toHaveCount(2)
})

test('review detail: the + Filter builder adds a chip and narrows the findings', async ({ page }) => {
  await page.goto('/reviews/rev3')
  // Triage view counts readout — rev3 has 6 findings across dimensions/severities.
  await expect(page.getByText('Showing 6 of 6')).toBeVisible()

  // The progressive builder offers Lens / Severity / Status fields.
  await page.getByRole('button', { name: 'Add filter' }).click()
  await expect(page.getByRole('menuitem', { name: 'Lens' })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: 'Severity' })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: 'Status' })).toBeVisible()

  // Adding a Lens filter creates a chip (removable) and opens its value popover.
  await page.getByRole('menuitem', { name: 'Lens' }).click()
  await expect(page.getByRole('button', { name: 'Remove Lens filter' })).toBeVisible()

  // Selecting Risk narrows the list to the 2 risk findings; counts stay honest.
  await page.getByRole('option', { name: 'Risk' }).click()
  await expect(page.getByText('Showing 2 of 6')).toBeVisible()

  // No stray humanize tabs (regression: a bad response rendered dozens).
  await expect(page.getByRole('button', { name: 'V9' })).toBeHidden()
})
