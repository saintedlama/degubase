import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, cleanup } from '../fixtures'

/**
 * Table icon picker (#99) — a visual flapout grid instead of a raw text input.
 */
test.describe('Table icon picker (#99)', () => {
  let workspaceCode = ''
  let tableCode = ''
  let tableViewUrl = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'Icon Test WS')
    const tbl = await createTable(request, workspaceCode, 'Icon Table')
    tableCode = tbl.code
    tableViewUrl = tbl.url
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('icon picker shows a flapout grid instead of a text input', async ({ page }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Icon Table' })).toBeVisible()

    await page.getByTitle('Edit table').click()
    await expect(page.getByRole('heading', { name: 'Edit Table' })).toBeVisible()

    await expect(page.locator('input[placeholder="ri-database-2-line"]')).not.toBeVisible()
    await expect(page.locator('button').filter({ hasText: 'No icon' })).toBeVisible()

    await page.getByRole('button', { name: 'Cancel' }).click()
  })

  test('selecting an icon from the flapout saves it on the table', async ({ page, request }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Icon Table' })).toBeVisible()

    await page.getByTitle('Edit table').click()
    await expect(page.getByRole('heading', { name: 'Edit Table' })).toBeVisible()

    await page.locator('button').filter({ hasText: 'No icon' }).click()

    const grid = page.locator('.grid.grid-cols-6')
    await expect(grid).toBeVisible()

    await page.locator('button[title="Database"]').click()
    await expect(grid).not.toBeVisible()

    await page.getByRole('button', { name: 'Save' }).click()

    const tblResp = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/`)
    const tbl = await tblResp.json()
    expect(tbl.icon).toBe('database')
  })
})
