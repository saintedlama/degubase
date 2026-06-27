import { test, expect } from "@playwright/test";
import { createWorkspace, createTable, seedRow, cleanup } from "../fixtures";

/**
 * Record terminology (#94, #95) — detail header shows actual DB id; all
 * user-visible text uses "record" not "row".
 *
 * Tests are independent (each navigates fresh); no serial mode needed.
 */
test.describe("Record terminology (#94, #95)", () => {
  let workspaceCode = "";
  let tableCode = "";
  let tableViewUrl = "";
  let rowId = 0;

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "TermTest WS");
    const tbl = await createTable(request, workspaceCode, "TermTest Table");
    tableCode = tbl.code;
    tableViewUrl = tbl.url;
    rowId = await seedRow(request, workspaceCode, tableCode);
    await seedRow(request, workspaceCode, tableCode);
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("#94: record detail header shows actual DB id matching URL", async ({
    page,
  }) => {
    const detailUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rowId}`;
    await page.goto(detailUrl);
    await page.waitForURL(new RegExp(`/rows/${rowId}`));

    await expect(page.getByText(`Record #${rowId}`)).toBeVisible();
    await expect(page.getByText(/^Row \d+/)).not.toBeVisible();
  });

  test('#95: record detail header and tooltips use "record" not "row"', async ({
    page,
  }) => {
    const detailUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rowId}`;
    await page.goto(detailUrl);
    await page.waitForURL(new RegExp(`/rows/${rowId}`));

    await expect(page.locator('button[title="Previous record"]')).toBeVisible();
    await expect(page.locator('button[title="Next record"]')).toBeVisible();
    await expect(
      page.locator('button[title="Copy link to this record"]'),
    ).toBeVisible();
  });

  test('#95: tabular "delete" button tooltip says "Delete record"', async ({
    page,
  }) => {
    await page.goto(tableViewUrl);
    await expect(page.locator("tbody tr").first()).toBeVisible();

    const firstRow = page.locator("tbody tr.group").first();
    // Open the row actions flapout (dropdown is teleported to body)
    await firstRow.getByTitle("Row actions").click();

    const deleteBtn = page.getByTestId("delete-row-btn");
    await expect(deleteBtn).toBeVisible();
    await expect(deleteBtn).toHaveAttribute("title", "Delete record");
  });
});

/**
 * Loading indicators (#75) — top progress bar during navigation, spinner on
 * button during form submission.
 *
 * Tests are independent; no serial mode needed.
 */
test.describe("Loading indicators (#75)", () => {
  let workspaceCode = "";
  let tableViewUrl = "";

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "ProgressTest");
    const tbl = await createTable(request, workspaceCode, "SpinnerTable");
    tableViewUrl = tbl.url;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("#75 progress bar appears during slow navigation", async ({ page }) => {
    await page.route("/api/workspaces/", async (route) => {
      await new Promise((r) => setTimeout(r, 300));
      await route.continue();
    });

    const barSeen = page.waitForSelector("div.z-\\[9999\\], div.z-9999", {
      timeout: 2000,
    });

    await page.goto("/");
    await barSeen;
  });

  test("#75 spinner appears on Save Column button while submitting", async ({
    page,
  }) => {
    await page.goto(tableViewUrl);

    let resolveRoute!: () => void;
    const blocker = new Promise<void>((r) => {
      resolveRoute = r;
    });

    await page.route(
      `/api/workspaces/${workspaceCode}/*/columns/`,
      async (route) => {
        if (route.request().method() === "POST") {
          await blocker;
          await route.continue();
        } else {
          await route.continue();
        }
      },
    );

    await page.getByTestId("add-column-btn").click();
    await page.getByTestId("column-name-input").fill("SpinnerCol");
    await page.getByTestId("save-column-btn").click();

    const spinner = page.locator(
      '[data-testid="save-column-btn"] svg.animate-spin',
    );
    await expect(spinner).toBeVisible({ timeout: 2000 });

    resolveRoute();
    await expect(spinner).not.toBeVisible({ timeout: 3000 });
  });
});
