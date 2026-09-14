import path from "node:path";
import { defineConfig, devices } from "@playwright/test";

const webAppRoot = __dirname;
const chromiumExecutablePath = process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH;

export default defineConfig({
  testDir: path.join(webAppRoot, "e2e"),
  outputDir: path.join(webAppRoot, "test-results"),
  fullyParallel: !!process.env.CI,
  workers: process.env.CI ? undefined : 1,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "on-first-retry"
  },
  projects: [
    {
      name: "chromium",
      use: {
        ...devices["Desktop Chrome"],
        ...(chromiumExecutablePath
          ? { launchOptions: { executablePath: chromiumExecutablePath } }
          : {})
      }
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
