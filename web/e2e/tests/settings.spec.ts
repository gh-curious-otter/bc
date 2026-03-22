import { test, expect } from "@playwright/test";

test.describe("Settings page", () => {
  test("nickname can be changed and persists after reload", async ({
    page,
  }) => {
    await page.goto("/settings");

    // Wait for the settings page to load
    const heading = page.locator("h1", { hasText: "Settings" });
    await expect(heading).toBeVisible({ timeout: 10000 });

    // Find the nickname input (inside the User section)
    const nicknameInput = page.locator('input[placeholder="@username"]');
    await expect(nicknameInput).toBeVisible();

    // Generate a unique nickname to avoid collisions
    const testNickname = `@e2e${Date.now().toString(36)}`;

    // Clear and type new nickname
    await nicknameInput.fill(testNickname);

    // Click Save within the User section
    const userSection = page.locator("div", { hasText: "Nickname" }).last();
    const saveButton = userSection.locator("button", { hasText: "Save" });
    await saveButton.click();

    // Wait for the save confirmation
    await expect(page.locator("text=Saved")).toBeVisible({ timeout: 5000 });

    // Reload the page
    await page.reload();

    // Wait for settings to load again
    await expect(heading).toBeVisible({ timeout: 10000 });

    // Verify the nickname persisted
    const reloadedInput = page.locator('input[placeholder="@username"]');
    await expect(reloadedInput).toHaveValue(testNickname, { timeout: 5000 });
  });
});
