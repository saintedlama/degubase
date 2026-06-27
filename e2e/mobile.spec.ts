import { test } from "./test";
import { expect, devices } from "@playwright/test";
import { cleanup } from "./fixtures";

// Strip defaultBrowserType so test.use() works inside a describe group.
// We still get the iPhone 14 viewport (390×664), touch support, and device scale.
const { defaultBrowserType: _bt, ...IPHONE_14 } = devices["iPhone 14"];

/**
 * Mobile UX regression suite — issues #50, #51, #52, #79.
 *
 * Each test describes the EXPECTED behaviour after the fix.
 *
 * #51 — Sort / Filter / Fields / Add record buttons overflow the 390 px
 *        viewport. toBeInViewport() fails until the toolbar wraps.
 * #50 — "Set as default" and "Delete view" in the view-switcher dropdown have
 *        opacity-0; they are never revealed on touch (no hover event).
 *        evaluate(opacity) must be > 0.
 * #52 — Row-action buttons (open, delete) already use [@media(hover:none)]:flex
 *        and should pass. The column-header menu button uses v-show with
 *        @mouseenter, so it stays display:none on mobile — toBeVisible() fails.
 */
test.describe("Mobile UX", () => {
  test.describe.configure({ mode: "serial" });
  test.use({ ...IPHONE_14 });

  let workspaceCode = "";
  let tabularViewUrl = "";
  let cardsViewUrl = "";

  test.beforeAll(async ({ request }) => {
    const ws = await (
      await request.post("/api/workspaces", {
        data: { name: "Mobile Test", context: "" },
      })
    ).json();
    workspaceCode = ws.code;

    const tbl = await (
      await request.post(`/api/workspaces/${workspaceCode}/tables`, {
        data: { name: "Tasks", template: "blank" },
      })
    ).json();

    const tblDetail = await (
      await request.get(`/api/workspaces/${workspaceCode}/tables/${tbl.code}`)
    ).json();
    const viewCode = tblDetail.views[0].code;
    tabularViewUrl = `/workspaces/${workspaceCode}/tables/${tbl.code}/views/${viewCode}`;

    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tbl.code}/columns`,
      {
        data: { name: "Title", type: "text" },
      },
    );

    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tbl.code}/rows`,
      {
        data: { data: {} },
      },
    );

    const view = await (
      await request.post(
        `/api/workspaces/${workspaceCode}/tables/${tbl.code}/views`,
        { data: { name: "Cards", type: "card" } },
      )
    ).json();
    cardsViewUrl = `/workspaces/${workspaceCode}/tables/${tbl.code}/views/${view.code}`;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  // ── Issue #51: sub-toolbar buttons must be reachable on a 390 px screen ──

  test("#51 Sort button is in viewport on mobile", async ({ page }) => {
    await page.goto(tabularViewUrl);
    await expect(page.getByRole("button", { name: /Sort/ })).toBeInViewport();
  });

  test("#51 Filter button is in viewport on mobile", async ({ page }) => {
    await page.goto(tabularViewUrl);
    await expect(page.getByRole("button", { name: /Filter/ })).toBeInViewport();
  });

  test("#51 Fields button is in viewport on mobile", async ({ page }) => {
    await page.goto(tabularViewUrl);
    await expect(page.getByRole("button", { name: /Fields/ })).toBeInViewport();
  });

  test("#51 Add record button is in viewport on mobile", async ({ page }) => {
    await page.goto(tabularViewUrl);
    await expect(
      page.getByRole("button", { name: "Add record" }).first(),
    ).toBeInViewport();
  });

  // ── Issue #50: view-switcher action buttons must not require hover ─────────

  test("#50 Set as default button has opacity > 0 without hover", async ({
    page,
  }) => {
    await page.goto(cardsViewUrl);
    await page.getByTestId("view-switcher-btn").click();
    const btn = page.getByTitle("Set as default").first();
    await expect(btn).toBeAttached();
    const opacity = await btn.evaluate((el) =>
      parseFloat(window.getComputedStyle(el).opacity),
    );
    expect(opacity).toBeGreaterThan(0);
  });

  test("#50 Delete view button has opacity > 0 without hover", async ({
    page,
  }) => {
    await page.goto(cardsViewUrl);
    await page.getByTestId("view-switcher-btn").click();
    const btn = page.getByTitle("Delete view").first();
    await expect(btn).toBeAttached();
    const opacity = await btn.evaluate((el) =>
      parseFloat(window.getComputedStyle(el).opacity),
    );
    expect(opacity).toBeGreaterThan(0);
  });

  // ── Issue #52: hover-gated buttons must be visible on touch devices ────────

  test("#52 row open-row-btn visible without hover", async ({ page }) => {
    await page.goto(tabularViewUrl);
    const firstRow = page.locator("tbody tr.group:first-child");
    await expect(firstRow).toBeVisible();
    // Open the row actions flapout (… button) — dropdown is teleported to body
    await firstRow.getByTitle("Row actions").click();
    await expect(page.getByTestId("open-row-btn")).toBeVisible();
  });

  test("#52 row delete-row-btn visible without hover", async ({ page }) => {
    await page.goto(tabularViewUrl);
    const firstRow = page.locator("tbody tr.group:first-child");
    await expect(firstRow).toBeVisible();
    await firstRow.getByTitle("Row actions").click();
    await expect(page.getByTestId("delete-row-btn")).toBeVisible();
  });

  test("#52 column header menu button visible without mouseenter", async ({
    page,
  }) => {
    await page.goto(tabularViewUrl);
    const colHeader = page.locator("thead th").nth(1);
    const btn = colHeader.getByTitle("Column options");
    await expect(btn).toBeVisible();
  });

  // ── #79 Skills/scripts master-detail on mobile ────────────────────────────

  test("#79 skills list is full-width on mobile when nothing selected", async ({
    page,
  }) => {
    await page.goto(`/workspaces/${workspaceCode}/automations/skills`);
    const list = page
      .locator("div")
      .filter({ hasText: "No skills yet." })
      .first();
    await expect(list).toBeVisible();

    const placeholder = page.getByText("Select a skill or create a new one.");
    await expect(placeholder).not.toBeVisible();
  });

  test("#79 skills detail is full-width after selecting a skill", async ({
    request,
    page,
  }) => {
    const skill = await (
      await request.post(`/api/workspaces/${workspaceCode}/skills`, {
        data: {
          name: "TestSkill",
          description: "test",
          table_ids: [],
          operations: ["read"],
        },
      })
    ).json();

    await page.goto(
      `/workspaces/${workspaceCode}/automations/skills/${skill.skill?.id ?? skill.id}/install`,
    );

    const backBtn = page.getByRole("button", { name: "Skills" });
    await expect(backBtn).toBeVisible({ timeout: 10_000 });

    await backBtn.click();
    await expect(
      page.getByText("Select a skill or create a new one."),
    ).not.toBeVisible();
    await expect(
      page.locator("div").filter({ hasText: "TestSkill" }).first(),
    ).toBeVisible();

    await request
      .delete(
        `/api/workspaces/${workspaceCode}/skills/${skill.skill?.id ?? skill.id}`,
      )
      .catch(() => {});
  });

  test("#79 scripts list is full-width on mobile when nothing selected", async ({
    page,
  }) => {
    await page.goto(`/workspaces/${workspaceCode}/automations/scripts`);
    const list = page.getByText("No scripts yet.");
    await expect(list).toBeVisible();
    const placeholder = page.getByText("Select a script or create a new one.");
    await expect(placeholder).not.toBeVisible();
  });
});
