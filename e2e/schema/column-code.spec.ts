import { test, expect } from "@playwright/test";
import { createWorkspace, createTable, cleanup } from "../fixtures";

/**
 * Immutable column code feature.
 *
 * Serial: tests 2-6 build shared state (colCode set in test 2, row created in
 * test 3, column renamed in test 5, UI asserts on the row in test 6).
 */
test.describe("Column codes", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let tableCode = "";
  let colCode = "";
  let viewCode = "";

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "ColCode Test");
    const tbl = await createTable(request, workspaceCode, "Items");
    tableCode = tbl.code;
    viewCode = tbl.viewCode;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("creating a column returns a code field", async ({ request }) => {
    const col = await (
      await request.post(
        `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
        { data: { name: "My Field", type: "text" } },
      )
    ).json();

    expect(col.code).toBeDefined();
    expect(col.code).toBe("my-field");
    colCode = col.code;
  });

  test("row data uses column code as key", async ({ request }) => {
    const row = await (
      await request.post(
        `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
        { data: { data: { [colCode]: "hello" } } },
      )
    ).json();

    expect(row.data[colCode]).toBe("hello");
    expect(row.data["my-field"]).toBe("hello");
  });

  test("listing columns returns code field on each column", async ({
    request,
  }) => {
    const cols = await (
      await request.get(
        `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      )
    ).json();
    const userCols = (cols as any[]).filter((c: any) => c.id > 0);
    expect(userCols.length).toBeGreaterThan(0);
    for (const col of userCols) {
      expect(col.code).toBeDefined();
    }
  });

  test("renaming column preserves its code", async ({ request }) => {
    const updated = await (
      await request.put(
        `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/${colCode}`,
        { data: { name: "Renamed Field", type: "text" } },
      )
    ).json();

    expect(updated.code).toBe(colCode);
    expect(updated.name).toBe("Renamed Field");
  });

  test("record detail view renders column values via code", async ({
    page,
  }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${viewCode}`,
    );

    await expect(page.getByRole("cell", { name: "hello" })).toBeVisible({
      timeout: 10_000,
    });

    // Open the row actions flapout, then click Open record
    const row = page.getByRole("row", { name: /hello/ });
    await row.getByTitle("Row actions").click();
    await page.getByTestId("open-row-btn").click();

    await page.waitForURL(/\/rows\/\d+/);
    await expect(
      page.locator("input.form-input, textarea").filter({ hasValue: "hello" }),
    ).toBeVisible();
  });
});
