import { defineConfig, devices } from "@playwright/test";

// 3 distinct form factors: phone, tablet, desktop
const allProjects = [
  {
    name: "Desktop Chrome",
    use: { ...devices["Desktop Chrome"] },
  },
  {
    name: "iPhone 14",
    use: { ...devices["iPhone 14"] },
  },
  {
    name: "iPad Pro 11",
    use: { ...devices["iPad Pro 11"] },
  },
];

const chromeOnly = [
  {
    name: "Desktop Chrome",
    use: { ...devices["Desktop Chrome"] },
  },
];

// PRs: Chrome desktop only (fast). Merge to main: all 3 form factors.
// Local dev: all 3 form factors.
const projects =
  process.env.CI && !process.env.CI_FULL_MATRIX ? chromeOnly : allProjects;

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 4 : undefined,
  reporter: [["html"], ["list"]],
  use: {
    baseURL: process.env.BASE_URL || "http://localhost:3000",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects,
  webServer: process.env.CI
    ? undefined
    : {
        command: "bun run dev",
        url: "http://localhost:3000",
        reuseExistingServer: true,
      },
});
