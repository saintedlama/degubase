import { test, expect } from "@playwright/test";
import { createWorkspace, cleanup } from "../fixtures";

/**
 * Row-link column type — picker, selection, persistence, clear, and blocked
 * deletes with referential integrity.
 *
 * Serial: tests build shared state (select link → persist → clear → re-link →
 * verify blocked delete).
 */
test.describe("Row link column type", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let projTblCode = "";
  let projNameColCode = "";
  let taskTblCode = "";
  let rlColCode = "";
  let taskViewUrl = "";
  let projViewUrl = "";
  let projectRowId = 0;
  let taskRowId = 0;

  test.beforeAll(async ({ request }) => {
    const ws = await request.post("/api/workspaces/", {
      data: { name: "RowLink E2E" },
    });
    expect(ws.status()).toBe(201);
    workspaceCode = (await ws.json()).code;

    const projTbl = await request.post(
      `/api/workspaces/${workspaceCode}/tables/`,
      { data: { name: "Projects" } },
    );
    expect(projTbl.status()).toBe(201);
    projTblCode = (await projTbl.json()).code;

    const projNameCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${projTblCode}/columns/`,
      {
        data: { name: "name", type: "text" },
      },
    );
    expect(projNameCol.status()).toBe(201);
    projNameColCode = (await projNameCol.json()).code;

    const projRow = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${projTblCode}/rows/`,
      {
        data: { data: { [projNameColCode]: "Alpha Project" } },
      },
    );
    expect(projRow.status()).toBe(201);
    projectRowId = (await projRow.json()).id;

    const taskTbl = await request.post(
      `/api/workspaces/${workspaceCode}/tables/`,
      { data: { name: "Tasks" } },
    );
    expect(taskTbl.status()).toBe(201);
    taskTblCode = (await taskTbl.json()).code;

    const rlCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/columns/`,
      {
        data: {
          name: "Project",
          type: "row-link",
          options: { targetTableCode: projTblCode },
        },
      },
    );
    expect(rlCol.status()).toBe(201);
    rlColCode = (await rlCol.json()).code;

    const taskRow = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/rows/`,
      {
        data: { data: {} },
      },
    );
    expect(taskRow.status()).toBe(201);
    taskRowId = (await taskRow.json()).id;

    const views = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/views/`,
    );
    const viewList: Array<{ type: string; code: string }> = await views.json();
    const viewCode = viewList.find((v) => v.type === "tabular")?.code;
    expect(viewCode).toBeTruthy();
    taskViewUrl = `/workspaces/${workspaceCode}/tables/${taskTblCode}/views/${viewCode}`;

    const projViews = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${projTblCode}/views/`,
    );
    const projViewList: Array<{ type: string; code: string }> =
      await projViews.json();
    const projViewCode = projViewList.find((v) => v.type === "tabular")?.code;
    expect(projViewCode).toBeTruthy();
    projViewUrl = `/workspaces/${workspaceCode}/tables/${projTblCode}/views/${projViewCode}`;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("row-link column header appears in the table", async ({ page }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();
    await expect(
      page.getByRole("columnheader", { name: /Project/ }),
    ).toBeVisible();
  });

  test("clicking the row-link cell opens the picker with project rows listed", async ({
    page,
  }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();

    const projectHeader = page.getByRole("columnheader", { name: /Project/ });
    const box = await projectHeader.boundingBox();
    expect(box).not.toBeNull();
    await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height + 20);

    await expect(page.getByPlaceholder("Search rows…")).toBeVisible();
    await expect(
      page.getByRole("button", { name: /Alpha Project/ }),
    ).toBeVisible();
  });

  test("picker search filters the row list", async ({ page }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();

    const projectHeader = page.getByRole("columnheader", { name: /Project/ });
    const box = await projectHeader.boundingBox();
    await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height + 20);

    const search = page.getByPlaceholder("Search rows…");
    await expect(search).toBeVisible();

    await search.fill("Alpha");
    await expect(
      page.getByRole("button", { name: /Alpha Project/ }),
    ).toBeVisible();

    await search.fill("zzznomatch");
    await expect(page.getByText("No rows found")).toBeVisible();

    await page.keyboard.press("Escape");
  });

  test("selecting a row stores the link and shows the label in the cell", async ({
    page,
  }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();

    const projectHeader = page.getByRole("columnheader", { name: /Project/ });
    const box = await projectHeader.boundingBox();
    await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height + 20);

    await expect(
      page.getByRole("button", { name: /Alpha Project/ }),
    ).toBeVisible();
    await page.getByRole("button", { name: /Alpha Project/ }).click();

    await page.getByRole("button", { name: "Close (Esc)" }).click();

    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).toBeVisible();
  });

  test("row-link label persists after page reload", async ({ page }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();
    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).toBeVisible();
  });

  test("API confirms the link value was stored correctly", async ({
    request,
  }) => {
    const resp = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/rows/${taskRowId}`,
    );
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    const link = body.data[rlColCode];
    expect(link).toBeTruthy();
    expect(link.id).toBe(projectRowId);
    expect(link.label).toBe("Alpha Project");
  });

  test('picker shows "Clear link" when a link is set; clearing removes the label', async ({
    page,
  }) => {
    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();

    const projectHeader = page.getByRole("columnheader", { name: /Project/ });
    const box = await projectHeader.boundingBox();
    await page.mouse.click(box!.x + box!.width / 2, box!.y + box!.height + 20);

    await expect(
      page.getByRole("button", { name: /Clear link/ }),
    ).toBeVisible();
    await page.getByRole("button", { name: /Clear link/ }).click();

    await page.getByRole("button", { name: "Close (Esc)" }).click();

    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).not.toBeVisible();
  });

  test("deleting a referenced row is blocked with 409 and the link remains intact", async ({
    page,
    request,
  }) => {
    await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/rows/${taskRowId}`,
      {
        data: {
          data: { [rlColCode]: { id: projectRowId, label: "Alpha Project" } },
        },
      },
    );

    await page.goto(taskViewUrl);
    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).toBeVisible();

    const del = await request.delete(
      `/api/workspaces/${workspaceCode}/tables/${projTblCode}/rows/${projectRowId}`,
    );
    expect(del.status()).toBe(409);
    const delBody = await del.json();
    expect(delBody.code).toBe("referenced");
    expect(Array.isArray(delBody.references)).toBe(true);
    expect(delBody.references.length).toBeGreaterThan(0);

    await page.goto(taskViewUrl);
    await expect(page.getByRole("heading", { name: "Tasks" })).toBeVisible();
    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).toBeVisible();
  });

  test("API confirms the link cell is still set after the blocked delete", async ({
    request,
  }) => {
    const resp = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${taskTblCode}/rows/${taskRowId}`,
    );
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.data[rlColCode]).toBeTruthy();
    expect(body.data[rlColCode].id).toBe(projectRowId);
  });

  test("UI: deleting a referenced row shows the blocking dialog with a link to the referencing record", async ({
    page,
  }) => {
    await page.goto(projViewUrl);
    await expect(page.getByRole("heading", { name: "Projects" })).toBeVisible();

    // Open the row actions flapout, then click Delete
    const row = page.locator("tbody tr").first();
    await row.getByTitle("Row actions").click();
    await page.getByTestId("delete-row-btn").click();

    await expect(
      page.getByRole("button", { name: "Delete", exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Delete", exact: true }).click();

    await expect(page.getByText("Cannot delete record")).toBeVisible();
    await expect(page.getByText(/Tasks #/)).toBeVisible();

    const link = page.getByRole("link", { name: /Tasks #/ });
    await expect(link).toHaveAttribute(
      "href",
      new RegExp(`/tables/${taskTblCode}/rows/${taskRowId}`),
    );

    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByText("Cannot delete record")).not.toBeVisible();
    await expect(
      page.locator("tbody").getByText("Alpha Project"),
    ).toBeVisible();
  });
});

/**
 * Row link – display column selection.
 *
 * Serial: changing display column via UI in one test, then verifying the
 * updated label in subsequent tests.
 */
test.describe("Row link – display column selection", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let tgtTblCode = "";
  let titleColCode = "";
  let srcTblCode = "";
  let refColCode = "";
  let tgtRow1Id = 0;
  let srcRow1Id = 0;
  let srcViewUrl = "";

  test.beforeAll(async ({ request }) => {
    const ws = await request.post("/api/workspaces/", {
      data: { name: "DisplayCol E2E" },
    });
    expect(ws.status()).toBe(201);
    workspaceCode = (await ws.json()).code;

    const tgtTbl = await request.post(
      `/api/workspaces/${workspaceCode}/tables/`,
      { data: { name: "Categories" } },
    );
    expect(tgtTbl.status()).toBe(201);
    tgtTblCode = (await tgtTbl.json()).code;

    const codeCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tgtTblCode}/columns/`,
      {
        data: { name: "code", type: "text" },
      },
    );
    expect(codeCol.status()).toBe(201);
    const codeColCode = (await codeCol.json()).code;

    const titleCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tgtTblCode}/columns/`,
      {
        data: { name: "title", type: "text" },
      },
    );
    expect(titleCol.status()).toBe(201);
    titleColCode = (await titleCol.json()).code;

    const r1 = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tgtTblCode}/rows/`,
      {
        data: {
          data: { [codeColCode]: "CAT-1", [titleColCode]: "Category Alpha" },
        },
      },
    );
    expect(r1.status()).toBe(201);
    tgtRow1Id = (await r1.json()).id;

    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tgtTblCode}/rows/`,
      {
        data: {
          data: { [codeColCode]: "CAT-2", [titleColCode]: "Category Beta" },
        },
      },
    );

    const srcTbl = await request.post(
      `/api/workspaces/${workspaceCode}/tables/`,
      { data: { name: "Items" } },
    );
    expect(srcTbl.status()).toBe(201);
    srcTblCode = (await srcTbl.json()).code;

    const refCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${srcTblCode}/columns/`,
      {
        data: {
          name: "Category",
          type: "row-link",
          options: { targetTableCode: tgtTblCode },
        },
      },
    );
    expect(refCol.status()).toBe(201);
    refColCode = (await refCol.json()).code;

    const sr1 = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${srcTblCode}/rows/`,
      {
        data: { data: { [refColCode]: { id: tgtRow1Id, label: "CAT-1" } } },
      },
    );
    expect(sr1.status()).toBe(201);
    srcRow1Id = (await sr1.json()).id;

    const views = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${srcTblCode}/views/`,
    );
    const viewList: Array<{ type: string; code: string }> = await views.json();
    const viewCode = viewList.find((v) => v.type === "tabular")?.code;
    expect(viewCode).toBeTruthy();
    srcViewUrl = `/workspaces/${workspaceCode}/tables/${srcTblCode}/views/${viewCode}`;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  test("auto mode: cell shows the first text column (code) as label", async ({
    page,
  }) => {
    await page.goto(srcViewUrl);
    await expect(page.getByRole("heading", { name: "Items" })).toBeVisible();
    await expect(page.locator("tbody").getByText("CAT-1")).toBeVisible();
  });

  test('changing display column to "title" saves successfully', async ({
    page,
  }) => {
    await page.goto(srcViewUrl);
    await expect(page.getByRole("heading", { name: "Items" })).toBeVisible();

    await page.getByTitle("Edit columns").click();
    await expect(
      page.getByRole("button", { name: "Add column" }),
    ).toBeVisible();

    await page
      .locator("span")
      .filter({ hasText: /^Category$/ })
      .first()
      .locator("..")
      .getByTitle("Edit column")
      .click();

    await expect(
      page.locator("label", { hasText: "Display column" }),
    ).toBeVisible();

    const displaySelect = page
      .locator("label", { hasText: "Display column" })
      .locator("..")
      .locator("select");
    await expect(displaySelect).toBeEnabled();

    await displaySelect.selectOption(titleColCode);
    await expect(displaySelect).toHaveValue(titleColCode);

    const [putResp] = await Promise.all([
      page.waitForResponse(
        (r) => r.url().includes("/columns/") && r.request().method() === "PUT",
      ),
      page.getByRole("button", { name: "Save", exact: true }).click(),
    ]);
    expect(putResp.status()).toBe(200);

    await expect(
      page.locator("label", { hasText: "Display column" }),
    ).not.toBeVisible();
  });

  test("API check: label is persisted to the database right after save", async ({
    request,
  }) => {
    const resp = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${srcTblCode}/rows/${srcRow1Id}`,
    );
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.data?.[refColCode]?.label).toBe("Category Alpha");
  });

  test("cell shows updated label after display column change", async ({
    page,
  }) => {
    const [getResp] = await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes(`/tables/${srcTblCode}/rows/`) &&
          r.request().method() === "GET",
      ),
      page.goto(srcViewUrl),
    ]);
    await expect(page.getByRole("heading", { name: "Items" })).toBeVisible();
    const getBody = await getResp.json();
    const rowData = getBody?.data?.find(
      (r: { id: number }) => r.id === srcRow1Id,
    );
    expect(rowData?.data?.[refColCode]?.label).toBe("Category Alpha");
    await expect(
      page.locator("tbody").getByText("Category Alpha"),
    ).toBeVisible();
    await expect(page.locator("tbody").getByText("CAT-1")).not.toBeVisible();
  });

  test("API confirms updated label is persisted on the row", async ({
    request,
  }) => {
    const resp = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${srcTblCode}/rows/${srcRow1Id}`,
    );
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    const link = body.data[refColCode];
    expect(link).toBeTruthy();
    expect(link.id).toBe(tgtRow1Id);
    expect(link.label).toBe("Category Alpha");
  });
});
