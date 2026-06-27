import { test, expect } from "@playwright/test";
import {
  createWorkspace,
  createTable,
  seedRow,
  cleanup,
  type ColSpec,
} from "../fixtures";

/**
 * Referencing records — verifies that the /referencing API and UI work
 * end-to-end:
 *   1. Record detail shows "Referenced by" with links
 *   2. API returns correct paginated data
 */

test.describe("Referencing records", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let projTableCode = "";
  let taskTableCode = "";
  let taskTitleCode = "";
  let rlCode = "";
  let projectRowId = 0;
  let task1RowId = 0;
  let task2RowId = 0;

  test.beforeAll(async ({ request }) => {
    workspaceCode = await createWorkspace(request, "RefE2E");

    // Create Projects table (target of the links).
    const projTbl = await createTable(request, workspaceCode, "Projects");
    projTableCode = projTbl.code;

    // Create Tasks table (source of the links).
    const taskTbl = await createTable(request, workspaceCode, "Tasks");
    taskTableCode = taskTbl.code;

    // Add a text column to Tasks for label resolution.
    const tcRes = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${taskTableCode}/columns/`,
      { data: { name: "Title", type: "text" } },
    );
    expect(tcRes.status()).toBe(201);
    taskTitleCode = (await tcRes.json()).code;

    // Add a row-link column on Tasks → Projects.
    const rlCols: ColSpec[] = [
      {
        name: "project",
        type: "row-link",
        options: { targetTableCode: projTableCode },
      },
    ];
    const rlRes = await request.post(
      `/api/workspaces/${workspaceCode}/tables/${taskTableCode}/columns/`,
      { data: rlCols[0] },
    );
    expect(rlRes.status()).toBe(201);
    rlCode = (await rlRes.json()).code;

    // Create a project row.
    projectRowId = await seedRow(request, workspaceCode, projTableCode);

    // Create two tasks linked to the project.
    task1RowId = await seedRow(request, workspaceCode, taskTableCode, {
      [taskTitleCode]: "Fix login",
      [rlCode]: { id: projectRowId, label: "Project" },
    });
    task2RowId = await seedRow(request, workspaceCode, taskTableCode, {
      [taskTitleCode]: "Add dashboard",
      [rlCode]: { id: projectRowId, label: "Project" },
    });

    // Create an unlinked task to ensure it does NOT appear.
    await seedRow(request, workspaceCode, taskTableCode, {
      [taskTitleCode]: "Unrelated task",
    });
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  // ── 1. Record detail: "Referenced by" section ──────────────────────────────

  test("record detail shows Referenced by section with linking records", async ({
    page,
  }) => {
    const detailUrl = `/workspaces/${workspaceCode}/tables/${projTableCode}/rows/${projectRowId}`;
    await page.goto(detailUrl);
    await page.waitForURL(new RegExp(`/rows/${projectRowId}`));

    // The "Referenced by" toggle button should be visible.
    const refsBtn = page.getByRole("button", { name: /Referenced by/ });
    await expect(refsBtn).toBeVisible();

    // Expand the section.
    await refsBtn.click();

    // Should list the two referencing tasks.
    await expect(page.getByText("Fix login")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("Add dashboard")).toBeVisible();

    // The unlinked task should not appear.
    await expect(page.getByText("Unrelated task")).not.toBeVisible();
  });

  // ── 2. API: direct call ────────────────────────────────────────────────────

  test("referencing API returns correct paginated data", async ({
    request,
  }) => {
    const res = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${projTableCode}/rows/${projectRowId}/referencing`,
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.total).toBe(2);
    expect(body.data).toHaveLength(2);

    const labels = body.data.map((r: any) => r.sourceRowLabel).sort();
    expect(labels).toEqual(["Add dashboard", "Fix login"]);

    // Pagination: page size 1.
    const p1 = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${projTableCode}/rows/${projectRowId}/referencing?pageSize=1&page=1`,
    );
    expect(p1.status()).toBe(200);
    const p1Body = await p1.json();
    expect(p1Body.data).toHaveLength(1);
    expect(p1Body.total).toBe(2);

    // Unreferenced row returns empty.
    const empty = await request.get(
      `/api/workspaces/${workspaceCode}/tables/${taskTableCode}/rows/${task1RowId}/referencing`,
    );
    expect(empty.status()).toBe(200);
    const emptyBody = await empty.json();
    expect(emptyBody.total).toBe(0);
    expect(emptyBody.data).toHaveLength(0);
  });
});
