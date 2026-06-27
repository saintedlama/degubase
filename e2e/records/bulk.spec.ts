import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Bulk actions (#71) — bulk edit dialog, preview, update, and direct API.
 *
 * Serial: the UI bulk-update test changes all open rows to closed; the
 * subsequent API preview test asserts 0 open rows remain.
 */
test.describe('Bulk actions (#71)', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let tableUrl = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'BulkTest WS')
    const tbl = await createTable(request, workspaceCode, 'BulkTest Table', [
      { name: 'status', type: 'single-select', options: { choices: ['open', 'closed'] } },
    ])
    tableCode = tbl.code
    tableUrl = tbl.url

    for (let i = 0; i < 3; i++) {
      await seedRow(request, workspaceCode, tableCode, { status: 'open' })
    }
    await seedRow(request, workspaceCode, tableCode, { status: 'closed' })
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('Bulk edit button is visible in toolbar', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await expect(page.getByRole('button', { name: 'Bulk edit' })).toBeVisible()
  })

  test('opens bulk edit dialog', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Bulk edit' }).click()
    await expect(page.getByRole('heading', { name: 'Bulk edit records' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Preview' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Update records' })).toBeDisabled()
  })

  test('preview shows total row count with no filters', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Bulk edit' }).click()
    await page.getByRole('button', { name: 'Preview' }).click()
    await expect(page.getByText('4 records will be updated')).toBeVisible()
  })

  test('preview counts filtered rows correctly', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Bulk edit' }).click()

    await page.getByRole('button', { name: 'Add filter' }).click()
    const filterValueSel = page.locator('section').first().locator('select').last()
    await filterValueSel.selectOption('open')

    await page.getByRole('button', { name: 'Preview' }).click()
    await expect(page.getByText('3 records will be updated')).toBeVisible()
  })

  test('bulk update patches matching rows and reloads table', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Bulk edit' }).click()

    await page.getByRole('button', { name: 'Add filter' }).click()
    const filterValueSel = page.locator('section').first().locator('select').last()
    await filterValueSel.selectOption('open')

    await page.getByRole('button', { name: 'Add field' }).click()
    const updateValueSel = page.locator('section').nth(1).locator('select').last()
    await updateValueSel.selectOption('closed')

    await page.getByRole('button', { name: 'Preview' }).click()
    await expect(page.getByText('3 records will be updated')).toBeVisible()

    await page.getByRole('button', { name: 'Update records' }).click()

    await expect(page.getByRole('heading', { name: 'Bulk edit records' })).not.toBeVisible()
    await expect(page.getByText('3 records updated')).toBeVisible()
  })

  test('bulk API: preview returns count without modifying rows', async ({ request }) => {
    const res = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/bulk`,
      { data: { filters: [{ col: 'status', op: 'is', value: 'open' }], data: {}, preview: true } },
    )
    expect(res.status()).toBe(200)
    const body = await res.json()
    expect(body).toHaveProperty('count')
    expect(body.count).toBe(0)
  })

  test('bulk API: patches multiple rows in one call', async ({ request }) => {
    const listRes = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`)
    const rows = (await listRes.json()).data
    for (const row of rows) {
      await request.patch(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${row.id}`, {
        data: { data: { status: 'open' } },
      })
    }

    const bulkRes = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/bulk`,
      { data: { filters: [], data: { status: 'closed' }, preview: false } },
    )
    expect(bulkRes.status()).toBe(200)
    const body = await bulkRes.json()
    expect(body.updated).toBe(4)

    const verify = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`, {
      params: { filter: 'status:is:closed' },
    })
    expect((await verify.json()).total).toBe(4)
  })
})
