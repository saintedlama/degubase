import { test, expect } from '@playwright/test'
import { createWorkspace, createTable, seedRow, cleanup } from '../fixtures'

/**
 * Activity panel annotations (#96) on the record detail view.
 *
 * Serial: tests build shared state — annotation created in test 3 is edited in
 * test 5, and the change-entry note from test 9 is verified in test 10.
 */
test.describe('Annotations (#96)', () => {
  test.describe.configure({ mode: 'serial' })

  let workspaceCode = ''
  let tableCode = ''
  let rowId = 0
  let detailUrl = ''

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, 'AnnTest WS')
    const tbl = await createTable(request, workspaceCode, 'AnnTest Table')
    tableCode = tbl.code
    rowId = await seedRow(request, workspaceCode, tableCode)
    detailUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rowId}`
  })

  test.afterAll(async ({ request }) => { await cleanup(request, workspaceCode) })

  test('Activity panel opens and shows the markdown editor and Add button', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))

    await expect(page.locator('[contenteditable="true"]')).not.toBeVisible()

    await page.getByRole('button', { name: /Activity/ }).click()

    await expect(page.locator('[contenteditable="true"]').first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add', exact: true })).toBeVisible()
  })

  test('Activity panel shows the initial change entry from row creation', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    await expect(page.getByRole('button', { name: '+ Add note' }).first()).toBeVisible()
  })

  test('can create a standalone annotation', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    await page.locator('[contenteditable="true"]').first().fill('First annotation')
    await page.getByRole('button', { name: 'Add', exact: true }).click()

    await expect(page.getByText('First annotation')).toBeVisible()
  })

  test('annotation persists after page reload', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    await expect(page.getByText('First annotation')).toBeVisible()
  })

  test('Ctrl+Enter submits the annotation', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    const editor = page.locator('[contenteditable="true"]').first()
    await editor.fill('Keyboard annotation')
    await editor.press('Control+Enter')

    await expect(page.locator('.annotation-md').getByText('Keyboard annotation')).toBeVisible()
  })

  test('can edit an annotation inline', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    const annotationCards = page.locator('.border-amber-200')
    await expect(annotationCards.first()).toBeVisible()
    const firstCard = annotationCards.first()

    await firstCard.getByRole('button', { name: 'Edit', exact: true }).click()

    const editEditor = firstCard.locator('[contenteditable="true"]')
    await expect(editEditor).toBeVisible()
    await editEditor.fill('Edited annotation text')
    await firstCard.getByRole('button', { name: 'Save', exact: true }).click()

    await expect(page.getByText('Edited annotation text')).toBeVisible()
  })

  test('Cancel discards the edit', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    const annotationCards = page.locator('.border-amber-200')
    await expect(annotationCards.first()).toBeVisible()
    const firstCard = annotationCards.first()

    const originalText = (await firstCard.locator('.annotation-md').first().textContent())?.trim() ?? ''

    await firstCard.getByRole('button', { name: 'Edit', exact: true }).click()
    await firstCard.locator('[contenteditable="true"]').fill('Should be discarded')
    await firstCard.getByRole('button', { name: 'Cancel', exact: true }).click()

    await expect(page.getByText('Should be discarded')).not.toBeVisible()
    await expect(firstCard.locator('.annotation-md').first()).toContainText(originalText)
  })

  test('can delete an annotation', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    const annotationCards = page.locator('.border-amber-200')
    await expect(annotationCards.first()).toBeVisible()
    const countBefore = await annotationCards.count()

    const lastCard = annotationCards.last()
    const deletedText = (await lastCard.locator('.annotation-md').first().textContent())?.trim() ?? ''
    await lastCard.getByRole('button', { name: 'Delete', exact: true }).click()

    await expect(annotationCards).toHaveCount(countBefore - 1)
    if (deletedText) await expect(page.getByText(deletedText, { exact: true })).not.toBeVisible()
  })

  test('can add a note to a change entry', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    const addNoteBtn = page.getByRole('button', { name: '+ Add note' }).first()
    await expect(addNoteBtn).toBeVisible()
    await addNoteBtn.click()

    const noteEditor = page.locator('[contenteditable="true"]').last()
    await expect(noteEditor).toBeVisible()
    await noteEditor.fill('Why this record was created')
    await page.getByRole('button', { name: 'Save', exact: true }).click()

    await expect(page.getByText('Why this record was created')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit note' })).toBeVisible()
  })

  test('change entry note persists after page reload', async ({ page }) => {
    await page.goto(detailUrl)
    await page.waitForURL(new RegExp(`/rows/${rowId}`))
    await page.getByRole('button', { name: /Activity/ }).click()

    await expect(page.getByText('Why this record was created')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Edit note' })).toBeVisible()
  })
})
