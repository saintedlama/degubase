import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Markdown editor (#98) — must not swallow characters when typing fast.
 *
 * Bug: watch on `modelValue` called setContent() while editor had focus,
 * resetting content to the last saved snapshot. Fix: skip setContent() when
 * the editor has focus (hasFocus guard).
 */
test.describe('Markdown editor (#98)', () => {
  let workspaceCode = ''
  let tableCode = ''
  let rowId = 0
  let detailUrl = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'MD Test WS')
    const tbl = await createTable(request, workspaceCode, 'MD Table', [
      { name: 'Notes', type: 'markdown' },
    ])
    tableCode = tbl.code
    rowId = await seedRow(request, workspaceCode, tableCode)
    detailUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rowId}`
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('#98: typing quickly into markdown editor retains all characters', async ({ page }) => {
    await page.goto(detailUrl)

    const editorDiv = page.locator('.tiptap[contenteditable="true"]')
    await expect(editorDiv).toBeVisible()

    await editorDiv.click()

    const text = 'Hello World this is a fast typing test 1234567890'
    await editorDiv.type(text, { delay: 0 })

    await page.waitForTimeout(1500)

    await expect(editorDiv).toContainText(text)

    await page.reload()
    await expect(page.locator('.tiptap[contenteditable="true"]')).toContainText(text)
  })
})
