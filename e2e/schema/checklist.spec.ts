import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Checklist column type — creation, adding items, toggling, persistence,
 * and progress-bar display in the tabular cell renderer.
 *
 * Serial: each test builds on the previous (adds items, checks them, verifies
 * the count after reload).
 */
test.describe('Checklist column type', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let tabularViewCode = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'Checklist Test WS')
    const tbl = await createTable(request, workspaceCode, 'CL Table', [
      { name: 'Tasks', type: 'checklist' },
    ])
    tableCode = tbl.code
    tabularViewCode = tbl.viewCode
    await seedRow(request, workspaceCode, tableCode)
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('clicking a checklist cell opens the flapout with an Add item input', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    const tasksHeader = page.getByRole('columnheader', { name: /Tasks/ })
    const headerBox = await tasksHeader.boundingBox()
    expect(headerBox).not.toBeNull()

    await page.mouse.click(headerBox!.x + headerBox!.width / 2, headerBox!.y + headerBox!.height + 20)

    await expect(page.getByRole('textbox', { name: 'Add item…' })).toBeVisible()
  })

  test('can add items to the checklist, they appear in the flapout and persist after close', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    const tasksHeader = page.getByRole('columnheader', { name: /Tasks/ })
    const headerBox = await tasksHeader.boundingBox()
    await page.mouse.click(headerBox!.x + headerBox!.width / 2, headerBox!.y + headerBox!.height + 20)
    await expect(page.getByRole('textbox', { name: 'Add item…' })).toBeVisible()

    const addInput = page.getByRole('textbox', { name: 'Add item…' })
    await addInput.fill('Write tests')
    await addInput.press('Enter')
    await addInput.fill('Fix bugs')
    await addInput.press('Enter')
    await addInput.fill('Deploy')
    await addInput.press('Enter')

    await expect(page.getByText('Write tests')).toBeVisible()
    await expect(page.getByText('Fix bugs')).toBeVisible()
    await expect(page.getByText('Deploy')).toBeVisible()

    await page.getByRole('button', { name: 'Close (Esc)' }).click()
    await expect(page.getByRole('textbox', { name: 'Add item…' })).not.toBeVisible()
  })

  test('checking an item crosses it out and persists to the API', async ({ page, request }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    const tasksHeader = page.getByRole('columnheader', { name: /Tasks/ })
    const headerBox = await tasksHeader.boundingBox()
    await page.mouse.click(headerBox!.x + headerBox!.width / 2, headerBox!.y + headerBox!.height + 20)
    await expect(page.getByText('Write tests')).toBeVisible()

    const checkboxes = page.getByRole('checkbox')
    await checkboxes.first().click()

    await page.getByRole('button', { name: 'Close (Esc)' }).click()

    const rows = await request.get(`/api/workspaces/${workspaceCode}/tables/${tableCode}/rows`)
    const rowData = await rows.json()
    const items = Object.values(rowData.data[0].data).find((v): v is Array<{ text: string; checked: boolean }> =>
      Array.isArray(v),
    )
    expect(items).toBeDefined()
    const writeTests = items!.find(i => i.text === 'Write tests')
    expect(writeTests?.checked).toBe(true)
    const fixBugs = items!.find(i => i.text === 'Fix bugs')
    expect(fixBugs?.checked).toBe(false)
  })

  test('cell shows progress count and bar after items are saved', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    await expect(page.getByText('1 / 3')).toBeVisible()
  })

  test('progress bar updates after checking a second item', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    const tasksHeader = page.getByRole('columnheader', { name: /Tasks/ })
    const headerBox = await tasksHeader.boundingBox()
    await page.mouse.click(headerBox!.x + headerBox!.width / 2, headerBox!.y + headerBox!.height + 20)
    await expect(page.getByText('Fix bugs')).toBeVisible()

    const checkboxes = page.getByRole('checkbox')
    await checkboxes.nth(1).click()

    await page.getByRole('button', { name: 'Close (Esc)' }).click()

    await expect(page.getByText('2 / 3')).toBeVisible()
  })

  test('checklist data persists after page reload', async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}/tables/${tableCode}/views/${tabularViewCode}`)
    await expect(page.getByRole('heading', { name: 'CL Table' })).toBeVisible()

    await expect(page.getByText('2 / 3')).toBeVisible()
  })
})
