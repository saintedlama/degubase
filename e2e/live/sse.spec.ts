import { test, expect } from '@playwright/test'
import { createWorkspace, cleanup } from '../fixtures'

/**
 * Kanban SSE live-update (#97) — board refreshes automatically via SSE when
 * rows are created or updated from an external source, without a page reload.
 *
 * Serial: the update test reads the first row created in the new-row test.
 */
test.describe('Kanban SSE live-update (#97)', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let statusColCode = ''
  let kanbanViewCode = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'SSE Test WS')

    const tbl = await request.post(`/api/workspaces/${workspaceCode}/tables/`, { data: { name: 'SSE Table' } })
    expect(tbl.status()).toBe(201)
    tableCode = (await tbl.json()).code

    const col = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`, {
      data: { name: 'Status', type: 'single-select', options: { choices: ['Todo', 'Done'] } },
    })
    expect(col.status()).toBe(201)
    statusColCode = (await col.json()).code

    const kv = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`, {
      data: { name: 'Board', type: 'kanban' },
    })
    expect(kv.status()).toBe(201)
    kanbanViewCode = (await kv.json()).code

    const row = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`, {
      data: { data: { [statusColCode]: 'Todo' } },
    })
    expect(row.status()).toBe(201)
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('#97: new row created via API appears in kanban without page reload', async ({ page, request }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`)

    await expect(page.getByText('Todo')).toBeVisible()
    const todoBefore = await page.locator('[class*="flex-col"]').filter({ hasText: 'Todo' }).locator('[draggable]').count()

    const newRow = await request.post(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`, {
      data: { data: { [statusColCode]: 'Todo' } },
    })
    expect(newRow.status()).toBe(201)

    await expect(async () => {
      const todoAfter = await page.locator('[class*="flex-col"]').filter({ hasText: 'Todo' }).locator('[draggable]').count()
      expect(todoAfter).toBe(todoBefore + 1)
    }).toPass({ timeout: 5000 })
  })

  test('#97: row updated via API moves card to correct lane without reload', async ({ page, request }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`)
    await expect(page.getByText('Todo')).toBeVisible()

    const rowsResp = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`)
    const rows = (await rowsResp.json()).data ?? (await rowsResp.json()).items
    const firstRow = rows[0]

    await request.patch(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${firstRow.id}`, {
      data: { data: { [statusColCode]: 'Done' } },
    })

    await expect(async () => {
      const doneCount = await page.locator('[class*="flex-col"]').filter({ hasText: 'Done' }).locator('[draggable]').count()
      expect(doneCount).toBeGreaterThan(0)
    }).toPass({ timeout: 5000 })
  })
})
