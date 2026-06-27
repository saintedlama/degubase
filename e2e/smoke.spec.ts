import { test } from "./test";
import { expect } from "@playwright/test";
import { cleanup } from "./fixtures";

/**
 * Smoke test suite — covers the core happy path end-to-end:
 *   home → create workspace → create table → add column →
 *   add row → edit cell → open record detail → delete row → cleanup
 *
 * Interaction strategy:
 *   - Actions  → page.getByTestId('…')   (stable, decoupled from copy/styling)
 *   - Assertions → getByRole / getByText  (validates what the user actually sees)
 *
 * Tests run serially and share state through the closure variables below.
 * If any step fails the remaining tests are skipped automatically.
 */
test.describe("DeguBase smoke", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let tableViewUrl = "";

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  // ── 1. Home page ──────────────────────────────────────────────────────────

  test("home page loads with branding", async ({ page }) => {
    await page.goto("/");
    await expect(
      page.getByRole("heading", { name: "Workspaces" }),
    ).toBeVisible();
    await expect(page.getByText("DeguBase").first()).toBeVisible();
    await expect(
      page.getByRole("button", { name: "New workspace" }),
    ).toBeVisible();
  });

  // ── 2. Workspace ──────────────────────────────────────────────────────────

  test("create workspace", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "New workspace" }).click();
    await expect(
      page.getByRole("heading", { name: "New Workspace" }),
    ).toBeVisible();

    await page.getByTestId("ws-name").fill("Smoke Workspace");
    await page.getByRole("button", { name: "Create Workspace" }).click();

    await page.waitForURL(/\/workspaces\/[^/]+$/);
    workspaceCode = new URL(page.url()).pathname.replace(/^\/workspaces\//, "");
    expect(workspaceCode).toBeTruthy();

    await expect(
      page.getByRole("heading", { name: "Smoke Workspace" }),
    ).toBeVisible();
  });

  // ── 3. Table ──────────────────────────────────────────────────────────────

  test("create table", async ({ page }) => {
    await page.goto(`/workspaces/${workspaceCode}`);

    await page.getByTestId("new-table-btn").click();
    await expect(page.getByTestId("new-table-dialog")).toBeVisible();

    await page.getByTestId("table-name-input").fill("Contacts");
    await page.getByTestId("create-table-btn").click();

    await page.waitForURL(/\/views\//);
    tableViewUrl = new URL(page.url()).pathname;
    expect(tableViewUrl).toContain("/views/");

    await expect(page.getByText("Contacts").first()).toBeVisible();
  });

  // ── 4. Column ─────────────────────────────────────────────────────────────

  test("add a text column", async ({ page }) => {
    await page.goto(tableViewUrl);

    await page.getByTestId("add-column-btn").click();
    await expect(page.getByTestId("add-column-dialog")).toBeVisible();

    await page.getByTestId("column-name-input").fill("Notes");
    // Type defaults to "text" — no change needed
    await page.getByTestId("save-column-btn").click();

    await expect(page.locator("thead").getByText("Notes")).toBeVisible();
  });

  // ── 5. Row — add & edit ───────────────────────────────────────────────────

  test("add a row", async ({ page }) => {
    await page.goto(tableViewUrl);
    await page.getByTestId("add-row-btn").click();

    await expect(page.locator("tbody tr:first-child")).toBeVisible();
  });

  test("edit a cell inline", async ({ page }) => {
    await page.goto(tableViewUrl);

    // td:nth-child(2) is the first data column (td:nth-child(1) is the row-number cell)
    const cell = page.locator("tbody tr:first-child td:nth-child(2)");
    await cell.click();

    const input = cell.locator("input");
    await expect(input).toBeVisible();
    await input.fill("Hello DeguBase");
    await input.press("Enter");

    await expect(cell).toContainText("Hello DeguBase");
  });

  // ── 6. Record detail ──────────────────────────────────────────────────────

  test("open row detail view", async ({ page }) => {
    await page.goto(tableViewUrl);

    // Open the row actions flapout (… button), then click Open record
    const firstRow = page.locator("tbody tr.group:first-child");
    await expect(firstRow).toBeVisible();
    await firstRow.getByTitle("Row actions").click();
    await page.getByTestId("open-row-btn").click();

    await page.waitForURL(/\/rows\/\d+/);

    await expect(page.getByText("Notes").first()).toBeVisible();
    await expect(page.getByText("Hello DeguBase").first()).toBeVisible();
  });

  // ── 7. Delete row ─────────────────────────────────────────────────────────

  test("delete row", async ({ page }) => {
    await page.goto(tableViewUrl);

    // Count only data rows (tr.group), not the always-present "Add record" row.
    const dataRows = page.locator("tbody tr.group");
    await expect(dataRows.first()).toBeVisible();
    const rowsBefore = await dataRows.count();

    // Open the row actions flapout, click Delete, then confirm via the dialog.
    const firstRow = page.locator("tbody tr.group:first-child");
    await firstRow.getByTitle("Row actions").click();
    await page.getByTestId("delete-row-btn").click();

    await page.getByRole("button", { name: "Delete", exact: true }).click();

    await expect(dataRows).toHaveCount(rowsBefore - 1);
  });

  // ── 8. Workspace list after editing ───────────────────────────────────────

  test("workspace appears on home page", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByText("Smoke Workspace")).toBeVisible();
  });
});
