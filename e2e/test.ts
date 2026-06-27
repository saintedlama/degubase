/**
 * Shared Playwright test instance with automatic coverage collection.
 *
 * Import `test` and `expect` from here instead of `@playwright/test` in every
 * spec file. When COVERAGE=1 is set, each test's page is automatically scraped
 * for istanbul coverage data after the test completes.
 *
 * Usage:
 *   import { test, expect } from './test'
 */
import { test as base } from "@playwright/test";
import { collectCoverage } from "./coverage";

export const test = base.extend({
  page: async ({ page }, use) => {
    await use(page);
    if (process.env.COVERAGE === "1") {
      await collectCoverage(page);
    }
  },
});
