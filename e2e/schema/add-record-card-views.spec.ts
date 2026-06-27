import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Bug #136 — "Add record does not work in card based views"
 *
 * Root cause: RecordDetailView called api.listRows (default page size = 100) and
 * searched for the current row by ID in the result. Two problems:
 *   1. Any record beyond page 1 was never found ("Row not found").
 *   2. The displayed record should never depend on list pagination at all.
 *
 * Fix: RecordDetailView now has a separate `currentRow` ref fetched directly via
 * GET /rows/{id}. listRows is still called but only to populate the prev/next
 * navigation arrows — the displayed record is always fetched by ID.
 *
 * Tests:
 *   1. Card view   — Add record opens the record detail without "not found"
 *   2. Kanban view — same
 *   3. Regression  — row beyond default page size (>100 rows) is still found
 */
test.describe('Add record opens record detail (card-based views)', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let cardViewCode = ''
  let kanbanViewCode = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'AddRecord E2E')
    const tbl = await createTable(request, workspaceCode, 'Items', [
      { name: 'Title', type: 'text' },
      { name: 'Status', type: 'single-select', options: { choices: ['Todo', 'Done'] } },
    ])
    tableCode = tbl.code

    const cardView = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`,
      { data: { name: 'Cards', type: 'card' } },
    )
    expect(cardView.status()).toBe(201)
    cardViewCode = (await cardView.json()).code

    const kanbanView = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`,
      { data: { name: 'Board', type: 'kanban' } },
    )
    expect(kanbanView.status()).toBe(201)
    kanbanViewCode = (await kanbanView.json()).code
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('card view: Add record opens record detail without "not found"', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${cardViewCode}`)
    await expect(page.getByRole('heading', { name: 'Items' })).toBeVisible()

    await page.getByRole('button', { name: 'Add record' }).first().click()

    // Should navigate to a /rows/ URL and show the record editor, not a "not found" message
    await expect(page).toHaveURL(/\/rows\/\d+/)
    await expect(page.getByText(/not found/i)).not.toBeVisible()
  })

  test('kanban view: Add record opens record detail without "not found"', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`)
    await expect(page.getByRole('heading', { name: 'Items' })).toBeVisible()

    await page.getByRole('button', { name: 'Add record' }).first().click()

    await expect(page).toHaveURL(/\/rows\/\d+/)
    await expect(page.getByText(/not found/i)).not.toBeVisible()
  })

  test('regression: record beyond page 1 (>100 rows) is found in record detail', async ({ request, page }) => {
    // Seed 101 rows so the next new record is row 102 — beyond the default page size
    const colsResp = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`)
    const cols = await colsResp.json()
    const titleCol = cols.find((c: { name: string }) => c.name === 'Title')
    expect(titleCol).toBeTruthy()

    for (let i = 1; i <= 101; i++) {
      await seedRow(request, workspaceCode, tableCode, { [titleCol.code]: `Seed ${i}` })
    }

    // Create one more row via the card view "Add record" button
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${cardViewCode}`)
    await expect(page.getByRole('heading', { name: 'Items' })).toBeVisible()

    await page.getByRole('button', { name: 'Add record' }).first().click()

    await expect(page).toHaveURL(/\/rows\/\d+/)
    await expect(page.getByText(/not found/i)).not.toBeVisible()
  })
})
