import { test, expect } from "@playwright/test";
import { createWorkspace, createTable, cleanup } from "../fixtures";

/**
 * Referencing — self-referencing (same-table row links) — hierarchical tree rendering and cycle
 * detection.
 *
 * Setup: one table (Tree Test) with a text column (Name) and a self-
 * referential row-link column (Parent). Three rows: Root → Child → Grandchild.
 *
 * Serial: API tests first (cycle detection), then UI tests (tree rendering,
 * collapse/expand). Each UI test does a fresh page.goto so Vue state resets.
 */
test.describe("Self-referencing row links", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let tableCode = "";
  let nameColCode = "";
  let parentColCode = "";
  let viewUrl = "";
  let rootRowId = 0;
  let childRowId = 0;
  let grandchildRowId = 0;

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "Hierarchy E2E");

    const tbl = await createTable(request, workspaceCode, "Tree Test");
    tableCode = tbl.code;
    viewUrl = tbl.url;

    const nameCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      { data: { name: "Name", type: "text" } },
    );
    expect(nameCol.status()).toBe(201);
    nameColCode = (await nameCol.json()).code;

    const parentCol = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns/`,
      {
        data: {
          name: "Parent",
          type: "row-link",
          options: { targetTableCode: tableCode },
        },
      },
    );
    expect(parentCol.status()).toBe(201);
    parentColCode = (await parentCol.json()).code;

    const r1 = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
      { data: { data: { [nameColCode]: "Root" } } },
    );
    expect(r1.status()).toBe(201);
    rootRowId = (await r1.json()).id;

    const r2 = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
      {
        data: {
          data: {
            [nameColCode]: "Child",
            [parentColCode]: { id: rootRowId, label: "Root" },
          },
        },
      },
    );
    expect(r2.status()).toBe(201);
    childRowId = (await r2.json()).id;

    const r3 = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
      {
        data: {
          data: {
            [nameColCode]: "Grandchild",
            [parentColCode]: { id: childRowId, label: "Child" },
          },
        },
      },
    );
    expect(r3.status()).toBe(201);
    grandchildRowId = (await r3.json()).id;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  // ── Backend: cycle detection ───────────────────────────────────────────────

  test("API: self-cycle (row pointing to itself) is rejected with 400", async ({
    request,
  }) => {
    const resp = await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rootRowId}`,
      { data: { data: { [parentColCode]: { id: rootRowId, label: "Root" } } } },
    );
    expect(resp.status()).toBe(400);
    const body = await resp.json();
    expect(body.code).toBe("bad_request");
    expect(body.error).toMatch(/circular/i);
  });

  test("API: indirect cycle (A→B→A) is rejected with 400", async ({
    request,
  }) => {
    // Child's parent is Root; setting Root's parent to Child creates Root→Child→Root
    const resp = await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${rootRowId}`,
      {
        data: { data: { [parentColCode]: { id: childRowId, label: "Child" } } },
      },
    );
    expect(resp.status()).toBe(400);
    const body = await resp.json();
    expect(body.error).toMatch(/circular/i);
  });

  test("API: deep cycle (A→B→C→A) is allowed (graph handles it)", async ({
    request,
  }) => {
    // Root→Child→Grandchild; setting Grandchild's parent to Root creates a 3-cycle. Longer cycles are allowed — the graph traversal handles them safely.
    const resp = await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${grandchildRowId}`,
      {
        data: {
          data: {
            [parentColCode]: { id: rootRowId, label: "Root" },
          },
        },
      },
    );
    expect(resp.status()).toBe(200);
  });

  test("API: valid parent reassignment succeeds (200)", async ({ request }) => {
    // Grandchild currently points to Child. Reparenting it directly to Root is fine —
    // Root has no ancestor, so no cycle is possible.
    const reparent = await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${grandchildRowId}`,
      { data: { data: { [parentColCode]: { id: rootRowId, label: "Root" } } } },
    );
    expect(reparent.status()).toBe(200);

    // Restore original structure: Grandchild → Child
    const restore = await request.patch(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/${grandchildRowId}`,
      {
        data: { data: { [parentColCode]: { id: childRowId, label: "Child" } } },
      },
    );
    expect(restore.status()).toBe(200);
  });

  // ── Frontend: tree rendering ───────────────────────────────────────────────

  test("tree view: rows appear in DFS order (Root before Child before Grandchild)", async ({
    page,
  }) => {
    await page.goto(viewUrl);
    await expect(
      page.getByRole("heading", { name: "Tree Test" }),
    ).toBeVisible();

    const rows = page.locator("tbody tr.group");
    await expect(rows).toHaveCount(3);

    const texts = await rows.evaluateAll((els: Element[]) =>
      els.map((el) => el.textContent ?? ""),
    );
    const rootIdx = texts.findIndex((t) => t.includes("Root"));
    const childIdx = texts.findIndex(
      (t) => t.includes("Child") && !t.includes("Grandchild"),
    );
    const grandIdx = texts.findIndex((t) => t.includes("Grandchild"));
    expect(rootIdx).toBeLessThan(childIdx);
    expect(childIdx).toBeLessThan(grandIdx);
  });

  test("tree view: Root and Child rows each have an expand/collapse toggle", async ({
    page,
  }) => {
    await page.goto(viewUrl);
    await expect(
      page.getByRole("heading", { name: "Tree Test" }),
    ).toBeVisible();

    // Root has child = Child; Child has child = Grandchild → 2 toggles total
    await expect(page.getByTestId("tree-toggle")).toHaveCount(2);
  });

  test("tree view: collapsing Root hides Child and Grandchild", async ({
    page,
  }) => {
    await page.goto(viewUrl);
    await expect(page.locator("tbody tr.group")).toHaveCount(3);

    await page.getByTestId("tree-toggle").first().click();

    await expect(page.locator("tbody tr.group")).toHaveCount(1);
    await expect(page.locator("tbody").getByText("Child")).not.toBeVisible();
    await expect(
      page.locator("tbody").getByText("Grandchild"),
    ).not.toBeVisible();
  });

  test("tree view: expanding Root after collapse shows all rows again", async ({
    page,
  }) => {
    await page.goto(viewUrl);

    // Collapse
    await page.getByTestId("tree-toggle").first().click();
    await expect(page.locator("tbody tr.group")).toHaveCount(1);

    // Expand
    await page.getByTestId("tree-toggle").first().click();
    await expect(page.locator("tbody tr.group")).toHaveCount(3);
  });

  test("tree view: collapsing Child hides Grandchild but keeps Root and Child visible", async ({
    page,
  }) => {
    await page.goto(viewUrl);
    await expect(page.locator("tbody tr.group")).toHaveCount(3);

    // Second toggle belongs to Child (Root's toggle is first, Child's is second)
    await page.getByTestId("tree-toggle").nth(1).click();

    await expect(page.locator("tbody tr.group")).toHaveCount(2);
    // Name cell of the second data row should contain "Child" (not Grandchild)
    await expect(
      page.locator("tbody tr.group").nth(1).locator("td").nth(1),
    ).toContainText("Child");
    await expect(
      page.locator("tbody tr.group").nth(1).locator("td").nth(1),
    ).not.toContainText("Grandchild");
  });
});
