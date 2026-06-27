import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, cleanup } from '../fixtures'

/**
 * Modal migration regression tests (#90) — edit/delete flows use ModalDialog /
 * ConfirmDialog, never native confirm().
 */
test.describe('Modal flows (#90)', () => {
  let workspaceCode = ''
  let tableCode = ''
  let tableViewUrl = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'Modal Test WS')
    const tbl = await createTable(request, workspaceCode, 'Modal Table')
    tableCode = tbl.code
    tableViewUrl = tbl.url
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  // ── Edit table dialog ──────────────────────────────────────────────────────

  test('edit table: ModalDialog opens with current name pre-filled', async ({ page }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTitle('Edit table').click()
    await expect(page.getByRole('heading', { name: 'Edit Table' })).toBeVisible()

    const nameInput = page.locator('input[placeholder="Table name"]')
    await expect(nameInput).toHaveValue('Modal Table')

    await page.getByRole('button', { name: 'Cancel' }).click()
    await expect(page.getByRole('heading', { name: 'Edit Table' })).not.toBeVisible()
  })

  test('edit table: saving new name updates the toolbar heading', async ({ page }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTitle('Edit table').click()
    await page.locator('input[placeholder="Table name"]').fill('Renamed Table')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByRole('heading', { name: 'Edit Table' })).not.toBeVisible()
    await expect(page.getByRole('heading', { name: 'Renamed Table' })).toBeVisible()

    // Restore name for subsequent tests
    await page.getByTitle('Edit table').click()
    await page.locator('input[placeholder="Table name"]').fill('Modal Table')
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()
  })

  // ── Create view dialog ─────────────────────────────────────────────────────

  test('create view: ModalDialog opens with pre-filled name and closes on cancel', async ({ page }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTestId('view-switcher-btn').click()
    await page.getByTestId('new-view-btn-tabular').click()

    await expect(page.getByRole('heading', { name: 'New table view' })).toBeVisible()
    await expect(page.locator('input[placeholder="View name"]')).toHaveValue('Table')

    await page.getByRole('button', { name: 'Cancel' }).click()
    await expect(page.getByRole('heading', { name: 'New table view' })).not.toBeVisible()
  })

  test('create view: submitting the dialog creates a new view', async ({ page, request }) => {
    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTestId('view-switcher-btn').click()
    await page.getByTestId('new-view-btn-tabular').click()
    await expect(page.getByRole('heading', { name: 'New table view' })).toBeVisible()

    await page.locator('input[placeholder="View name"]').fill('Extra View')
    await page.getByRole('button', { name: 'Create view' }).click()

    await expect(page.getByRole('heading', { name: 'New table view' })).not.toBeVisible()

    const resp = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`)
    const views = await resp.json()
    expect(views.some((v: { name: string }) => v.name === 'Extra View')).toBe(true)
  })

  // ── Delete table confirm dialog ────────────────────────────────────────────

  test('delete table: ConfirmDialog renders (not native confirm) and cancel aborts', async ({ page }) => {
    await page.addInitScript(() => {
      window.confirm = () => { throw new Error('native confirm() must not be used') }
    })

    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTitle('Delete table').click()

    await expect(page.getByRole('heading', { name: 'Delete table' })).toBeVisible()
    await expect(page.getByText(/Delete "Modal Table"/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Delete', exact: true })).toBeVisible()

    await page.getByRole('button', { name: 'Cancel', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Delete table' })).not.toBeVisible()
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()
  })

  // ── Delete column confirm dialog ───────────────────────────────────────────

  test('delete column: ConfirmDialog renders and cancel leaves column intact', async ({ page, request }) => {
    const col = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`, {
      data: { name: 'TempCol', type: 'text' },
    })
    expect(col.status()).toBe(201)
    const colCode = (await col.json()).code

    await page.addInitScript(() => {
      window.confirm = () => { throw new Error('native confirm() must not be used') }
    })

    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTitle('Edit columns').click()
    await expect(page.getByText('Columns')).toBeVisible()

    const colRow = page.locator('div').filter({ hasText: 'TempCol' }).first()
    await colRow.getByTitle('Delete column').click()

    await expect(page.getByRole('heading', { name: 'Delete column?' })).toBeVisible()
    await expect(page.getByText(/Deleting "TempCol"/)).toBeVisible()

    await page.getByRole('button', { name: 'Cancel', exact: true }).click()
    await expect(page.getByRole('heading', { name: 'Delete column?' })).not.toBeVisible()
    await expect(page.getByText('TempCol', { exact: true })).toBeVisible()

    await request.delete(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/${colCode}`).catch(() => {})
  })

  test('delete column: confirming removes the column', async ({ page, request }) => {
    const col = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`, {
      data: { name: 'DisposeCol', type: 'text' },
    })
    expect(col.status()).toBe(201)

    await page.goto(tableViewUrl)
    await expect(page.getByRole('heading', { name: 'Modal Table' })).toBeVisible()

    await page.getByTitle('Edit columns').click()
    await expect(page.getByText('Columns')).toBeVisible()

    const colRow = page.locator('div').filter({ hasText: 'DisposeCol' }).first()
    await colRow.getByTitle('Delete column').click()

    await expect(page.getByRole('heading', { name: 'Delete column?' })).toBeVisible()

    const [deleteResp] = await Promise.all([
      page.waitForResponse(r => r.url().includes('/columns/') && r.request().method() === 'DELETE'),
      page.getByRole('button', { name: 'Delete', exact: true }).click(),
    ])
    expect(deleteResp.status()).toBe(204)

    await expect(page.getByRole('heading', { name: 'Delete column?' })).not.toBeVisible()
    await expect(page.getByText('DisposeCol')).not.toBeVisible()
  })
})
