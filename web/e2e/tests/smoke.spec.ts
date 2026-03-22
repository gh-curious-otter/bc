import { test, expect } from "@playwright/test";

const pages = [
  { path: "/", name: "Dashboard" },
  { path: "/agents", name: "Agents" },
  { path: "/channels", name: "Channels" },
  { path: "/costs", name: "Costs" },
  { path: "/roles", name: "Roles" },
  { path: "/tools", name: "Tools" },
  { path: "/mcp", name: "MCP" },
  { path: "/cron", name: "Cron" },
  { path: "/secrets", name: "Secrets" },
  { path: "/settings", name: "Settings" },
  { path: "/doctor", name: "Doctor" },
  { path: "/logs", name: "Logs" },
  { path: "/stats", name: "Stats" },
  { path: "/workspace", name: "Workspace" },
  { path: "/daemons", name: "Daemons" },
];

test.describe("Smoke tests — every sidebar page loads", () => {
  for (const { path, name } of pages) {
    test(`${name} (${path}) has a visible heading`, async ({ page }) => {
      await page.goto(path);
      const heading = page.locator("h1, h2, h3").first();
      await expect(heading).toBeVisible({ timeout: 10000 });
    });
  }
});
