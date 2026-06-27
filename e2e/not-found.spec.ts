import { test } from "./test";
import { expect } from "@playwright/test";

test.describe("404 Not Found page", () => {
  test("shows logo, title, and funny message for unknown deep path", async ({
    page,
  }) => {
    await page.goto("/foo/bar/baz");

    const logo = page.locator("img.not-found__logo");
    await expect(logo).toBeVisible();
    await expect(logo).toHaveAttribute("src", "/assets/logo.svg");

    await expect(page.getByRole("heading", { level: 1 })).toContainText("404");
    await expect(page.getByRole("heading", { level: 1 })).toContainText("degu");

    await expect(page.getByText("not the degu")).toBeVisible();

    const link = page.getByRole("link", { name: /go back home/i });
    await expect(link).toBeVisible();
    await link.click();
    await page.waitForURL("/");
  });
});
