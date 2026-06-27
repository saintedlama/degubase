import { defineConfig, devices } from "@playwright/test";
import path from "path";

export default defineConfig({
  testDir: ".",
  testMatch: "**/*.spec.ts",
  testIgnore: ["**/screenshots.spec.ts"], // screenshots run via screenshots.config.ts
  timeout: 30_000,
  retries: 0,
  workers: 1,
  reporter: [
    ["list"],
    ["html", { outputFolder: "playwright-report", open: "never" }],
  ],
  use: {
    baseURL: "http://localhost:5174",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  globalSetup: "./global-setup.ts",
  globalTeardown: "./global-teardown.ts",
  webServer: [
    {
      // Go backend on a dedicated test port so it never conflicts with the dev server.
      command: `go run ../cmd/server`,
      env: {
        DEGUBASE_DATA_DIR: path.join(__dirname, "e2e-data"),
        DEGUBASE_PORT: "8089",
        DEGUBASE_DISABLE_AUTH: "true",
      },
      url: "http://localhost:8089/api/health",
      timeout: 60_000,
      reuseExistingServer: !process.env.CI,
      stdout: "ignore",
      stderr: "pipe",
    },
    {
      // Vite dev server proxying to the test backend.
      command: `pnpm --dir ../ui run dev -- --port 5174`,
      env: {
        VITE_PORT: "5174",
        API_PROXY_TARGET: "http://localhost:8089",
        ...(process.env.COVERAGE ? { COVERAGE: "1" } : {}),
      },
      url: "http://localhost:5174",
      timeout: 30_000,
      reuseExistingServer: !process.env.CI,
      stdout: "ignore",
      stderr: "pipe",
    },
  ],
});
