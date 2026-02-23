import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

const webAppRoot = __dirname;

export default defineConfig({
  testDir: path.join(webAppRoot, "e2e"),
  outputDir: path.join(webAppRoot, "test-results"),
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "on-first-retry"
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] }
    }
  ],
  webServer: {
    command: "pnpm dev --port 3000",
    cwd: webAppRoot,
    url: "http://127.0.0.1:3000",
    reuseExistingServer: !process.env.CI,
    timeout: 120000
  }
});
