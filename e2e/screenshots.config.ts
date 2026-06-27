import { defineConfig, devices } from '@playwright/test'
import path from 'path'

export default defineConfig({
  testDir: '.',
  testMatch: '**/screenshots.spec.ts',
  timeout: 30_000,
  retries: 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:5174',
    screenshot: 'off',
    video: 'off',
    trace: 'off',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  globalSetup: './global-setup.ts',
  webServer: [
    {
      command: `go run ../cmd/server`,
      env: {
        DEGUBASE_DATA_DIR: path.join(__dirname, 'e2e-data'),
        DEGUBASE_PORT: '8089',
        DEGUBASE_DISABLE_AUTH: 'true',
      },
      url: 'http://localhost:8089/api/health',
      timeout: 60_000,
      reuseExistingServer: false,
      stdout: 'ignore',
      stderr: 'pipe',
    },
    {
      command: `pnpm --dir ../ui run dev -- --port 5174`,
      env: {
        VITE_PORT: '5174',
        API_PROXY_TARGET: 'http://localhost:8089',
      },
      url: 'http://localhost:5174',
      timeout: 30_000,
      reuseExistingServer: false,
      stdout: 'ignore',
      stderr: 'pipe',
    },
  ],
})
