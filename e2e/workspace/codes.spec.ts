import { test, expect } from '@playwright/test'
import { cleanup } from '../fixtures'

/**
 * Immutable 4-char workspace/table code feature.
 *
 * Serial: tests chain through wsCode / tableCode / tableViewUrl captured
 * during the UI creation flows — the creation flows ARE what is being tested,
 * so API setup is not appropriate here.
 */
test.describe('Workspace & table codes', () => {
  test.describe.configure({ mode: 'serial' })

  let wsCode = ''
  let tableCode = ''
  let tableViewUrl = ''

  test.afterAll(async ({ request }) => { await cleanup(request, wsCode) })

  test('new workspace dialog auto-generates code from name', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'New workspace' }).click()

    await page.getByTestId('ws-name').fill('Codetest Workspace')
    const codeInput = page.getByTestId('ws-code')
    await expect(codeInput).toHaveValue('CODE')
  })

  test('workspace URL uses the 4-char code', async ({ page }) => {
    await page.goto('/')
    await page.getByRole('button', { name: 'New workspace' }).click()

    await page.getByTestId('ws-name').fill('Codetest Workspace')
    await page.getByRole('button', { name: 'Create Workspace' }).click()

    await page.waitForURL(/\/workspaces\/[A-Z0-9]{4}$/)
    wsCode = new URL(page.url()).pathname.replace(/^\/workspaces\//, '')

    expect(wsCode).toMatch(/^[A-Z0-9]{4}$/)
    expect(wsCode).toBe('CODE')
    await expect(page.getByRole('heading', { name: 'Codetest Workspace' })).toBeVisible()
  })

  test('renaming workspace preserves code in URL', async ({ request }) => {
    const res = await request.put(`/api/workspaces/${wsCode}`, {
      data: { name: 'Renamed Workspace', context: '' },
    })
    const ws = await res.json()
    expect(ws.code).toBe(wsCode)
    expect(ws.name).toBe('Renamed Workspace')
  })

  test('new table dialog auto-generates code from name', async ({ page }) => {
    await page.goto(`/workspaces/${wsCode}`)

    await page.getByTestId('new-table-btn').click()
    await page.getByTestId('table-name-input').fill('Milestones')

    const codeInput = page.getByTestId('table-code-input')
    await expect(codeInput).toHaveValue('MILE')
  })

  test('table URL uses the 4-char code', async ({ page }) => {
    await page.goto(`/workspaces/${wsCode}`)

    await page.getByTestId('new-table-btn').click()
    await page.getByTestId('table-name-input').fill('Milestones')
    await page.getByTestId('create-table-btn').click()

    await page.waitForURL(/\/tables\/[A-Z0-9]{4}\/views\//)
    tableViewUrl = new URL(page.url()).pathname
    tableCode = tableViewUrl.match(/\/tables\/([A-Z0-9]{4})\//)?.[1] ?? ''

    expect(tableCode).toMatch(/^[A-Z0-9]{4}$/)
    expect(tableCode).toBe('MILE')
  })

  test('renaming table preserves code in URL', async ({ request }) => {
    const res = await request.put(`/api/workspaces/${wsCode}/tables/${tableCode}`, {
      data: { name: 'Renamed Table', context: '', icon: '' },
    })
    const tbl = await res.json()
    expect(tbl.code).toBe(tableCode)
    expect(tbl.name).toBe('Renamed Table')
  })
})
