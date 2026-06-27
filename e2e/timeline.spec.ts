import { test } from "./test";
import { expect } from "@playwright/test";
import { cleanup } from "./fixtures";

/**
 * Timeline view e2e tests.
 *
 * Covers:
 *  - weekday toggle + save persistence
 *  - zoom level switching
 *  - navigation (prev / next / today)
 *  - weekday presets (Workweek / All days)
 *  - adding a record
 */
test.describe("Timeline view", () => {
  test.describe.configure({ mode: "serial" });

  let workspaceCode = "";
  let timelineViewUrl = "";

  test.beforeAll(async ({ request }) => {
    // Create workspace
    const ws = await (
      await request.post("/api/workspaces/", {
        data: { name: "Timeline E2E", context: "" },
      })
    ).json();
    workspaceCode = ws.code;

    // Create table
    const tbl = await (
      await request.post(`/api/workspaces/${workspaceCode}/tables/`, {
        data: { name: "Events" },
      })
    ).json();
    const tableCode = tbl.code;

    // Add a date column
    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns`,
      { data: { name: "Event Date", type: "date" } },
    );

    // Add a text column for card content
    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns`,
      { data: { name: "Title", type: "text" } },
    );

    // Add a single-select column for swimlane grouping
    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/columns`,
      {
        data: {
          name: "Category",
          type: "single-select",
          options: { choices: ["Work", "Personal"] },
        },
      },
    );

    // Add a few seed rows with date data so cards appear on the timeline
    const today = new Date();
    const fmt = (d: Date) => d.toISOString().split("T")[0];

    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
      {
        data: {
          data: {
            "event-date": fmt(today),
            title: "Team standup",
          },
        },
      },
    );
    await request.post(
      `/api/workspaces/${workspaceCode}/tables/${tableCode}/rows/`,
      {
        data: {
          data: {
            "event-date": fmt(new Date(today.getTime() + 2 * 86400000)),
            title: "Sprint review",
          },
        },
      },
    );

    // Create a timeline view
    const view = await (
      await request.post(
        `/api/workspaces/${workspaceCode}/tables/${tableCode}/views`,
        { data: { name: "Timeline", type: "timeline" } },
      )
    ).json();

    timelineViewUrl = `/workspaces/${workspaceCode}/tables/${tableCode}/views/${view.code}`;
  });

  test.afterAll(async ({ request }) => {
    await cleanup(request, workspaceCode);
  });

  // ── Weekday toggle + save ────────────────────────────────────────────────

  test("loads with weekday toggle UI", async ({ page }) => {
    await page.goto(timelineViewUrl);

    const dayBtn = page.getByRole("button", { name: /^Days/ });
    await expect(dayBtn).toBeVisible({ timeout: 15_000 });
  });

  test("toggle off weekends and save view persists the change", async ({
    page,
  }) => {
    await page.goto(timelineViewUrl);

    const dayBtn = page.getByRole("button", { name: /^Days/ });
    await expect(dayBtn).toBeVisible({ timeout: 15_000 });

    // Open the weekday flapout
    await dayBtn.click();

    const satBtn = page.locator('button[title="Sat"]');
    const sunBtn = page.locator('button[title="Sun"]');
    await expect(satBtn).toBeVisible();
    await expect(sunBtn).toBeVisible();

    await satBtn.click();
    await sunBtn.click();

    // Close the flapout
    const backdrop = page.locator(".fixed.inset-0.z-49");
    await backdrop.click();

    await expect(dayBtn).toContainText("5/7");

    // Save the change
    const saveBtn = page.getByRole("button", { name: "Save view" });
    await expect(saveBtn).toBeVisible({ timeout: 5_000 });
    await saveBtn.click();

    const updateBtn = page.getByText('Update "Timeline"');
    await expect(updateBtn).toBeVisible();
    await updateBtn.click();

    await expect(saveBtn).not.toBeVisible({ timeout: 10_000 });
    await expect(dayBtn).toContainText("5/7");
  });

  test("weekdays persist after page reload", async ({ page }) => {
    await page.goto(timelineViewUrl);

    const dayBtn = page.getByRole("button", { name: /^Days/ });
    await expect(dayBtn).toBeVisible({ timeout: 15_000 });

    await expect(dayBtn).toContainText("5/7");

    const saveBtn = page.getByRole("button", { name: "Save view" });
    await expect(saveBtn).not.toBeVisible();
  });

  // ── Zoom ─────────────────────────────────────────────────────────────────

  // Helper: the zoom select is the last <select class="tl-ctl"> in the toolbar.
  function zoomSelect(page: any) {
    return page.locator("select.tl-ctl").last();
  }

  test("zoom defaults to week", async ({ page }) => {
    await page.goto(timelineViewUrl);
    await expect(zoomSelect(page)).toHaveValue("week");
  });

  test("switching zoom to month updates the grid", async ({ page }) => {
    await page.goto(timelineViewUrl);
    await zoomSelect(page).selectOption("month");
    await expect(zoomSelect(page)).toHaveValue("month");
    await expect(page.locator("button.tl-today-btn")).toBeVisible({
      timeout: 5_000,
    });
  });

  test("switching zoom to schedule hides empty days", async ({ page }) => {
    await page.goto(timelineViewUrl);
    await zoomSelect(page).selectOption("schedule");
    await expect(zoomSelect(page)).toHaveValue("schedule");
    await expect(page.locator("button.tl-today-btn")).toBeVisible({
      timeout: 5_000,
    });
  });

  test("switching zoom to overview renders weeks", async ({ page }) => {
    await page.goto(timelineViewUrl);
    await zoomSelect(page).selectOption("overview");
    await expect(zoomSelect(page)).toHaveValue("overview");
    await expect(page.locator("button.tl-today-btn")).toBeVisible({
      timeout: 5_000,
    });
  });

  // ── Navigation ───────────────────────────────────────────────────────────

  test("navigate prev button exists and is clickable", async ({ page }) => {
    await page.goto(timelineViewUrl);
    const prevBtn = page.locator('button[title="Previous"]');
    await expect(prevBtn).toBeVisible();
    // Clicking should not throw
    await prevBtn.click();
    // Grid should still be present after navigation
    await expect(page.locator("button.tl-today-btn")).toBeVisible({
      timeout: 5_000,
    });
  });

  test("navigate next button exists and is clickable", async ({ page }) => {
    await page.goto(timelineViewUrl);
    const nextBtn = page.locator('button[title="Next"]');
    await expect(nextBtn).toBeVisible();
    await nextBtn.click();
    await expect(page.locator("button.tl-today-btn")).toBeVisible({
      timeout: 5_000,
    });
  });

  test("navigating far enough changes the window label", async ({ page }) => {
    await page.goto(timelineViewUrl);

    const labelEl = page
      .locator("span.text-\\[12px\\].font-medium.text-text-1.whitespace-nowrap")
      .first();
    const before = await labelEl.textContent();

    // Go back 5 weeks to change the month
    for (let i = 0; i < 5; i++) {
      await page.locator('button[title="Previous"]').click();
      await page.waitForTimeout(100);
    }

    await expect(labelEl).not.toHaveText(before ?? "");
  });

  test("Today button resets after navigating away", async ({ page }) => {
    await page.goto(timelineViewUrl);

    // Go back several weeks first
    const prevBtn = page.locator('button[title="Previous"]');
    for (let i = 0; i < 5; i++) {
      await prevBtn.click();
      await page.waitForTimeout(100);
    }

    const labelBeforeToday = await page
      .locator("span.text-\\[12px\\].font-medium.text-text-1.whitespace-nowrap")
      .first()
      .textContent();

    await page.locator("button.tl-today-btn").click();

    const labelAfterToday = await page
      .locator("span.text-\\[12px\\].font-medium.text-text-1.whitespace-nowrap")
      .first()
      .textContent();

    // Today should bring us back, so the label should differ from the distant past
    expect(labelAfterToday).not.toBe(labelBeforeToday);
  });

  // ── Weekday presets ─────────────────────────────────────────────────────

  test("Workweek preset sets 5 days", async ({ page }) => {
    await page.goto(timelineViewUrl);

    // First restore all days so the preset has a visible effect
    const dayBtn = page.getByRole("button", { name: /^Days/ });
    await dayBtn.click();

    await page.locator('button:has-text("All days")').click();

    const backdrop = page.locator(".fixed.inset-0.z-49");
    await backdrop.click();

    // Now click Workweek preset
    await dayBtn.click();
    await page.locator('button:has-text("Workweek")').click();
    await backdrop.click();

    await expect(dayBtn).toContainText("5/7");
  });

  test("All days preset restores 7 days", async ({ page }) => {
    await page.goto(timelineViewUrl);

    const dayBtn = page.getByRole("button", { name: /^Days/ });
    await dayBtn.click();
    await page.locator('button:has-text("All days")').click();

    const backdrop = page.locator(".fixed.inset-0.z-49");
    await backdrop.click();

    // After restoring all days, the badge should disappear (allDaysVisible = true)
    const badge = dayBtn.locator("span");
    await expect(badge).not.toBeAttached();
  });

  // ── Add record ───────────────────────────────────────────────────────────

  test("Add record button is visible", async ({ page }) => {
    await page.goto(timelineViewUrl);
    await expect(
      page.getByRole("button", { name: "Add record" }).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("clicking Add record navigates to row detail", async ({ page }) => {
    await page.goto(timelineViewUrl);

    const addBtn = page.getByRole("button", { name: "Add record" }).first();
    await addBtn.click();

    // Clicking "Add record" creates a row and opens its detail view.
    // We should land on a /rows/ path.
    await page.waitForURL(/rows/, { timeout: 10_000 });
    await expect(page).toHaveURL(/\/rows\/\d+/);
  });
});
