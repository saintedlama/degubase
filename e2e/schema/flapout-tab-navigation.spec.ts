import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Tab key navigation through flapout-based cell editors in the tabular view.
 *
 * Root cause of the original bug: Tab events only fired on focused elements. When
 * a flapout opened, nothing inside it had focus, so Tab fired outside the panel
 * and was never caught. Fix: floatingEl gets tabindex="-1" and is auto-focused
 * once positioned (visibility:visible), so Tab events bubble to its @keydown.tab handler.
 *
 * Column layout under test:  Title (text) | Tags (multi-select) | Tasks (checklist)
 * Editable order:            Title → Tags → Tasks
 */
test.describe('Flapout tab navigation', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let viewUrl = ''
  let colIdText = 0
  let colIdMulti = 0
  let colIdChecklist = 0
  let rowId = 0

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'FlapoutTab E2E')
    const tbl = await createTable(request, workspaceCode, 'Items', [
      { name: 'Title',     type: 'text' },
      { name: 'Tags',      type: 'multi-select', options: { choices: ['alpha', 'beta', 'gamma'] } },
      { name: 'Tasks',     type: 'checklist' },
    ])
    tableCode = tbl.code
    viewUrl   = tbl.url

    const cols: Array<{ id: number; name: string }> = await (
      await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`)
    ).json()
    colIdText      = cols.find(c => c.name === 'Title')?.id     ?? 0
    colIdMulti     = cols.find(c => c.name === 'Tags')?.id      ?? 0
    colIdChecklist = cols.find(c => c.name === 'Tasks')?.id     ?? 0

    rowId = await seedRow(request, workspaceCode, tableCode)
    await seedRow(request, workspaceCode, tableCode)  // second row for wrap test
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('Tab from multi-select flapout opens next flapout (checklist)', async ({ page }) => {
    await page.goto(viewUrl)
    await expect(page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdMulti}"]`)).toBeVisible()

    // Open multi-select flapout
    await page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdMulti}"]`).click()
    const panel = page.locator('.fixed.z-200')
    await expect(panel).toBeVisible()
    await expect(panel).toBeFocused()

    // Tab → checklist flapout on same row
    await page.keyboard.press('Tab')

    await expect(panel).toBeVisible()
    // Tasks (checklist) td should now be the active flapout cell
    const checklistTd = page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdChecklist}"]`)
    await expect(checklistTd).toHaveClass(/outline-brand-500/)
  })

  test('Tab from checklist flapout wraps to first editable column on next row', async ({ page }) => {
    await page.goto(viewUrl)
    await expect(page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdChecklist}"]`)).toBeVisible()

    // Open checklist flapout (last editable column on rowId)
    await page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdChecklist}"]`).click()
    const panel = page.locator('.fixed.z-200')
    await expect(panel).toBeVisible()
    await expect(panel).toBeFocused()

    // Tab → wraps to Title (text, inline editor) on the next row
    await page.keyboard.press('Tab')

    // Flapout should be closed; an inline text editor should be active
    await expect(panel).not.toBeVisible()
    const nextTitleTd = page.locator(`td[data-col-id="${colIdText}"]`).nth(1)
    await expect(nextTitleTd).toHaveClass(/outline-brand-500/)
  })

  test('Shift+Tab from checklist flapout opens previous flapout (multi-select)', async ({ page }) => {
    await page.goto(viewUrl)
    await expect(page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdChecklist}"]`)).toBeVisible()

    // Open checklist flapout
    await page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdChecklist}"]`).click()
    const panel = page.locator('.fixed.z-200')
    await expect(panel).toBeVisible()
    await expect(panel).toBeFocused()

    // Shift+Tab → multi-select flapout
    await page.keyboard.press('Shift+Tab')

    await expect(panel).toBeVisible()
    const multiTd = page.locator(`td[data-row-id="${rowId}"][data-col-id="${colIdMulti}"]`)
    await expect(multiTd).toHaveClass(/outline-brand-500/)
  })
})
