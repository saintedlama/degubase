/**
 * Demo screenshot suite — navigates the six workspaces seeded by cmd/seed_demo
 * and captures marketing-quality PNGs into screenshots/.
 *
 * Run via:  make screenshots
 * Excluded from the regular `pnpm test` run (see playwright.config.ts testIgnore).
 */
import { test } from '@playwright/test'
import fs from 'fs'
import path from 'path'

const SHOTS = path.join(__dirname, '..', 'screenshots')

interface TableInfo {
  code: string
  views: Record<string, string> // viewType → viewCode
}

interface WorkspaceInfo {
  code: string
  tables: Record<string, TableInfo>
}

// Wait for data to appear based on view type.
// networkidle never resolves on table pages because of the SSE /events stream.
async function waitForView(page: any, viewType: string) {
  switch (viewType) {
    case 'tabular':
      await page.locator('tbody tr.group').first().waitFor({ timeout: 15_000 })
      break
    case 'kanban':
      // Wait for at least one kanban lane to render
      await page.locator('.w-64.shrink-0').first().waitFor({ timeout: 15_000 })
      break
    case 'card':
      // Wait for at least one card item to render
      await page.locator('.bg-surface-1.rounded-lg.border.cursor-pointer.shadow-sm').first().waitFor({ timeout: 15_000 })
      break
    case 'timeline':
      // Today button appears once the timeline grid has rendered
      await page.locator('button.tl-today-btn').waitFor({ timeout: 15_000 })
      break
    default:
      await page.waitForLoadState('load')
  }
}

test.describe('Demo screenshots', () => {
  test.describe.configure({ mode: 'serial' })
  test.use({ viewport: { width: 1440, height: 900 } })

  const wss: Record<string, WorkspaceInfo> = {}

  test.beforeAll(async ({ request }) => {
    fs.mkdirSync(SHOTS, { recursive: true })

    const workspaceList: any[] = await (await request.get('/api/workspaces/')).json()

    for (const ws of workspaceList) {
      const info: WorkspaceInfo = { code: ws.code, tables: {} }
      const tableList: any[] = await (
        await request.get(`/api/workspaces/${ws.code}/tables/`)
      ).json()
      for (const t of tableList) {
        const viewList: any[] = await (
          await request.get(`/api/workspaces/${ws.code}/tables/${t.code}/views/`)
        ).json()
        const views: Record<string, string> = {}
        for (const v of viewList) {
          views[v.type] = v.code
        }
        info.tables[t.name] = { code: t.code, views }
      }
      wss[ws.name] = info
    }
  })

  // Build a navigation URL for a given workspace / table / view type.
  function viewUrl(wsName: string, tableName: string, viewType = 'tabular') {
    const ws = wss[wsName]
    if (!ws) throw new Error(`Workspace not found: ${wsName}`)
    const t = ws.tables[tableName]
    if (!t) throw new Error(`Table not found: ${tableName} in ${wsName}`)
    const viewCode = t.views[viewType] ?? t.views['tabular']
    if (!viewCode) throw new Error(`No ${viewType} view on ${tableName}`)
    return `/workspaces/${ws.code}/tables/${t.code}/views/${viewCode}`
  }

  test('01 home — six workspace cards', async ({ page }) => {
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await page.screenshot({ path: path.join(SHOTS, '01-home.png') })
  })

  test('02 bookmarks gallery — links as cards', async ({ page }) => {
    await page.goto(viewUrl('Bookmarks', 'Links', 'card'))
    await waitForView(page, 'card')
    await page.screenshot({ path: path.join(SHOTS, '02-bookmarks-gallery.png') })
  })

  test('03 task tracker board — tasks kanban', async ({ page }) => {
    await page.goto(viewUrl('Task Tracker', 'Tasks', 'kanban'))
    await waitForView(page, 'kanban')
    await page.screenshot({ path: path.join(SHOTS, '03-task-tracker-board.png') })
  })

  test('04 bug tracker — bugs grid with severity and status', async ({ page }) => {
    await page.goto(viewUrl('Bug Tracker', 'Bugs', 'tabular'))
    await waitForView(page, 'tabular')
    await page.screenshot({ path: path.join(SHOTS, '04-bug-tracker-grid.png') })
  })

  test('05 crm pipeline — contacts kanban by stage', async ({ page }) => {
    await page.goto(viewUrl('CRM', 'Contacts', 'kanban'))
    await waitForView(page, 'kanban')
    await page.screenshot({ path: path.join(SHOTS, '05-crm-pipeline.png') })
  })

  test('06 recipe book gallery — recipes as cards', async ({ page }) => {
    await page.goto(viewUrl('Recipe Book', 'Recipes', 'card'))
    await waitForView(page, 'card')
    await page.screenshot({ path: path.join(SHOTS, '06-recipe-book-gallery.png') })
  })

  test('07 content calendar timeline — posts by publish date', async ({ page }) => {
    await page.goto(viewUrl('Content Calendar', 'Posts', 'timeline'))
    await waitForView(page, 'timeline')
    await page.screenshot({ path: path.join(SHOTS, '07-content-calendar-timeline.png') })
  })

  test('08 row detail — bug with markdown steps', async ({ page, request }) => {
    const ws = wss['Bug Tracker']
    const t = ws.tables['Bugs']
    const res = await request.get(
      `/api/workspaces/${ws.code}/tables/${t.code}/rows/?pageSize=1`
    )
    const { data } = await res.json()
    const firstRowId = data[0].id
    await page.goto(`/workspaces/${ws.code}/tables/${t.code}/rows/${firstRowId}`)
    await page.waitForLoadState('load')
    await page.screenshot({ path: path.join(SHOTS, '08-row-detail.png') })
  })
})
