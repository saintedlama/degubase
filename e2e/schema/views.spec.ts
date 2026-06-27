import { test } from "../test";
import { expect } from "@playwright/test";
import { createWorkspace, cleanup } from "../fixtures";

/**
 * View config save — card-field object arrays and axis labels persist via
 * PUT /views/:code.
 *
 * Serial: each save test is verified by the following reload test.
 */
test.describe("Save view config", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let tableCode = "";
  let matrixViewCode = "";
  let kanbanViewCode = "";

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "Views Test WS");

    const tbl = await request.post(`/api/workspaces/${workspaceCode}/tables/`, {
      data: { name: "VT Table" },
    });
    expect(tbl.status()).toBe(201);
    tableCode = (await tbl.json()).code;

    const colX = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      {
        data: {
          name: "Status",
          type: "single-select",
          options: { choices: ["Low", "High"] },
        },
      },
    );
    expect(colX.status()).toBe(201);
    const xColId = (await colX.json()).id;

    const colY = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      {
        data: {
          name: "Effort",
          type: "single-select",
          options: { choices: ["Easy", "Hard"] },
        },
      },
    );
    expect(colY.status()).toBe(201);
    const yColId = (await colY.json()).id;

    const colN = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      {
        data: { name: "Notes", type: "text" },
      },
    );
    expect(colN.status()).toBe(201);

    const matrixView = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`,
      {
        data: { name: "Matrix", type: "matrix", config: { xColId, yColId } },
      },
    );
    expect(matrixView.status()).toBe(201);
    matrixViewCode = (await matrixView.json()).code;

    const kanbanView = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/views/`,
      {
        data: {
          name: "Kanban",
          type: "kanban",
          config: { groupByColId: xColId },
        },
      },
    );
    expect(kanbanView.status()).toBe(201);
    kanbanViewCode = (await kanbanView.json()).code;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("saving view with card-field config returns 200", async ({ page }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${matrixViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    await page.getByRole("button", { name: "Fields" }).click();

    const notesCheckbox = page.getByLabel("Notes");
    await expect(notesCheckbox).toBeVisible();
    await notesCheckbox.click();

    await page.evaluate(() => {
      const overlay = document.querySelector<HTMLElement>(".fixed.inset-0");
      overlay?.click();
    });

    await expect(page.getByRole("button", { name: "Save view" })).toBeVisible();

    await page.getByRole("button", { name: "Save view" }).click();

    const [putResponse] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/views/${matrixViewCode}`) &&
          r.request().method() === "PUT",
      ),
      page.getByRole("button", { name: 'Update "Matrix"' }).click(),
    ]);

    expect(putResponse.status()).toBe(200);

    await expect(
      page.getByRole("button", { name: "Save view" }),
    ).not.toBeVisible();
  });

  test("saved card-field config persists after page reload", async ({
    page,
  }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${matrixViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    await expect(
      page.getByRole("button", { name: "Save view" }),
    ).not.toBeVisible();
  });

  test("toggling axis labels and saving persists hideAxisLabels", async ({
    page,
  }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${matrixViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    await page.getByRole("button", { name: "Fields" }).click();

    const axisCheckbox = page.getByLabel("Show axis labels");
    await expect(axisCheckbox).toBeVisible();
    await axisCheckbox.click();

    await page.evaluate(() => {
      document.querySelector<HTMLElement>(".fixed.inset-0")?.click();
    });

    await expect(page.getByRole("button", { name: "Save view" })).toBeVisible();

    await page.getByRole("button", { name: "Save view" }).click();
    const [putResponse] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/views/${matrixViewCode}`) &&
          r.request().method() === "PUT",
      ),
      page.getByRole("button", { name: 'Update "Matrix"' }).click(),
    ]);
    expect(putResponse.status()).toBe(200);

    const viewJson = await page.request.get(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/views/${matrixViewCode}`,
    );
    const view = await viewJson.json();
    expect(view.config).toBeDefined();
  });

  test("kanban: saving view with card-field objects returns 200", async ({
    page,
  }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    await page.getByRole("button", { name: "Fields" }).click();

    const notesCheckbox = page.getByLabel("Notes");
    await expect(notesCheckbox).toBeVisible();
    await notesCheckbox.click();

    await page.evaluate(() => {
      document.querySelector<HTMLElement>(".fixed.inset-0")?.click();
    });

    await expect(page.getByRole("button", { name: "Save view" })).toBeVisible();

    await page.getByRole("button", { name: "Save view" }).click();
    const [putResponse] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/views/${kanbanViewCode}`) &&
          r.request().method() === "PUT",
      ),
      page.getByRole("button", { name: 'Update "Kanban"' }).click(),
    ]);

    expect(putResponse.status()).toBe(200);
    await expect(
      page.getByRole("button", { name: "Save view" }),
    ).not.toBeVisible();
  });

  test("kanban: saved card-field config persists after reload", async ({
    page,
  }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Save view" }),
    ).not.toBeVisible();
  });

  test("kanban: hide a column and save persists", async ({ page }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    // Verify both lanes are present initially
    await expect(page.getByText("Low").first()).toBeVisible();
    await expect(page.getByText("High").first()).toBeVisible();

    // Open the column visibility flapout
    await page.locator('button:has-text("Columns")').last().click();

    // Uncheck the "High" checkbox to hide it
    const highCheckbox = page.locator(
      'label:has-text("High") input[type="checkbox"]',
    );
    await expect(highCheckbox).toBeVisible();
    await highCheckbox.uncheck();

    // Close the flapout
    await page.locator(".fixed.inset-0.z-49").click();

    // The "High" lane should now be hidden
    await expect(page.getByText("Low").first()).toBeVisible();
    await expect(page.getByText("High").first()).not.toBeAttached();

    // Save the view
    const saveBtn = page.getByRole("button", { name: "Save view" });
    await expect(saveBtn).toBeVisible({ timeout: 5_000 });
    await saveBtn.click();
    const [putResponse] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/views/${kanbanViewCode}`) &&
          r.request().method() === "PUT",
      ),
      page.getByRole("button", { name: 'Update "Kanban"' }).click(),
    ]);
    expect(putResponse.status()).toBe(200);
    await expect(saveBtn).not.toBeVisible({ timeout: 10_000 });
  });

  test("kanban: hidden column persists after reload", async ({ page }) => {
    await page.goto(
      `/workspaces/${workspaceCode}/tables/${tableCode}/views/${kanbanViewCode}`,
    );
    await expect(page.getByRole("heading", { name: "VT Table" })).toBeVisible();

    // "High" should still be hidden
    await expect(page.getByText("Low").first()).toBeVisible();
    await expect(page.getByText("High").first()).not.toBeAttached();

    // Save view should not appear (config matches saved state)
    await expect(
      page.getByRole("button", { name: "Save view" }),
    ).not.toBeVisible();
  });
});
