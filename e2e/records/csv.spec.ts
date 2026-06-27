import { test, expect } from '@playwright/test'
import { createWorkspace, cleanup } from '../fixtures'

/**
 * CSV import/export (#67) — API and UI flows for downloading and uploading CSV.
 *
 * Serial: the import API test measures row count before/after; a previous API
 * test imports rows that would affect the count if tests ran in parallel.
 */
test.describe('CSV import/export (#67)', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let tableUrl = ''
  let nameColCode = ''
  let noteColCode = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'CsvTest WS')

    const tbl = await request.post(`/api/workspaces/${workspaceCode}/tables/`, { data: { name: 'CsvTest Table' } })
    expect(tbl.status()).toBe(201)
    tableCode = (await tbl.json()).code

    const views = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`)
    const defaultView = (await views.json())[0]
    tableUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/views/${defaultView.code}`

    const nameCol = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`, {
      data: { name: 'name', type: 'text' },
    })
    expect(nameCol.status()).toBe(201)
    nameColCode = (await nameCol.json()).code

    const noteCol = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`, {
      data: { name: 'note', type: 'text' },
    })
    expect(noteCol.status()).toBe(201)
    noteColCode = (await noteCol.json()).code

    await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`, {
      data: { data: { [nameColCode]: 'Alice', [noteColCode]: 'First note' } },
    })
    await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`, {
      data: { data: { [nameColCode]: 'Bob', [noteColCode]: 'Second note' } },
    })
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  // ── API: Export ───────────────────────────────────────────────────────────

  test('export API: returns CSV with correct headers and row data', async ({ request }) => {
    const res = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/export.csv`,
    )
    expect(res.status()).toBe(200)
    expect(res.headers()['content-type']).toContain('text/csv')

    const body = await res.text()
    const lines = body.trim().split('\n')
    expect(lines[0]).toContain('name')
    expect(lines[0]).toContain('note')
    expect(lines.some(l => l.includes('Alice'))).toBe(true)
    expect(lines.some(l => l.includes('Bob'))).toBe(true)
  })

  test('export API: respects shorthand filter', async ({ request }) => {
    const res = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/export.csv?filter=${nameColCode}:is:Alice`,
    )
    expect(res.status()).toBe(200)
    const body = await res.text()
    expect(body).toContain('Alice')
    expect(body).not.toContain('Bob')
  })

  // ── API: Import ───────────────────────────────────────────────────────────

  test('import API: preview returns headers, sample and suggested mapping', async ({ request }) => {
    const csv = 'name,note\nCharlie,Third note\nDana,Fourth note'
    const res = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/import`,
      {
        multipart: {
          file: { name: 'test.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) },
        },
      },
    )
    expect(res.status()).toBe(200)
    const body = await res.json()
    expect(body.headers).toEqual(expect.arrayContaining(['name', 'note']))
    expect(Array.isArray(body.sample)).toBe(true)
    expect(body.suggested_mapping).toMatchObject({ name: nameColCode, note: noteColCode })
  })

  test('import API: imports rows with explicit mapping', async ({ request }) => {
    const countBefore = (await (await request.get(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
    )).json()).total

    const csv = 'name,note\nCharlie,Third note\nDana,Fourth note'
    const mapping = JSON.stringify({ name: nameColCode, note: noteColCode })
    const res = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/import`,
      {
        multipart: {
          file: { name: 'test.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) },
          mapping,
        },
      },
    )
    expect(res.status()).toBe(201)
    expect((await res.json()).imported).toBe(2)

    const countAfter = (await (await request.get(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
    )).json()).total
    expect(countAfter).toBe(countBefore + 2)
  })

  // ── UI: Export ────────────────────────────────────────────────────────────

  test('export button triggers CSV download', async ({ page }) => {
    await page.goto(tableUrl)
    const downloadPromise = page.waitForEvent('download')
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Export CSV' }).click()
    const download = await downloadPromise
    expect(download.suggestedFilename()).toMatch(/\.csv$/)
  })

  // ── UI: Import dialog ─────────────────────────────────────────────────────

  test('import dialog: opens and Next → is disabled without a file', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Import CSV' }).click()
    await expect(page.getByRole('heading', { name: 'Import CSV' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Next →' })).toBeDisabled()
  })

  test('import dialog: full flow — upload, mapping, import, rows in table', async ({ page }) => {
    await page.goto(tableUrl)
    await page.getByTitle('More actions').click()
    await page.getByRole('button', { name: 'Import CSV' }).click()

    const fileChooserPromise = page.waitForEvent('filechooser')
    await page.getByText('Drop a CSV file or click to browse').click()
    const fileChooser = await fileChooserPromise
    await fileChooser.setFiles({
      name: 'import.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from('name,note\nEve,Fifth note\nFrank,Sixth note'),
    })

    await expect(page.getByRole('button', { name: 'Next →' })).toBeEnabled()
    await page.getByRole('button', { name: 'Next →' }).click()

    await expect(page.getByText('Map CSV columns to table fields')).toBeVisible()
    await page.locator('button').filter({ hasText: /^Import$/ }).last().click()

    await expect(page.getByText('Import complete')).toBeVisible()
    await expect(page.getByText('2 records imported successfully.')).toBeVisible()
    await page.getByRole('button', { name: 'Close' }).click()

    await expect(page.getByRole('cell', { name: 'Eve' })).toBeVisible()
    await expect(page.getByRole('cell', { name: 'Frank' })).toBeVisible()
  })
})
