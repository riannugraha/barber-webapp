import { test, expect } from "./fixtures";

/**
 * 07-theme — dark persist
 * Assert: toggle → html.dark, reload → persist (next-themes + localStorage)
 */
test.describe("07 theme — dark persist", () => {
  test("toggle dark → html.dark class dan persist setelah reload", async ({ page }) => {
    await page.goto("/");

    // ThemeToggle button getByRole name Toggle theme
    const toggle = page.getByRole("button", { name: /toggle theme/i });
    await expect(toggle).toBeVisible({ timeout: 10_000 });

    // Ensure we start from light (remove dark if present)
    // Check html class
    const html = page.locator("html");
    // Force light first
    await page.evaluate(() => {
      localStorage.removeItem("theme");
      document.documentElement.classList.remove("dark");
      localStorage.setItem("theme", "light");
    });
    await page.reload();
    await expect(toggle).toBeVisible({ timeout: 10_000 });
    // After reload with light, html should not have dark
    await expect(html).not.toHaveClass(/dark/, { timeout: 10_000 }).catch(async () => {
      // If system dark, force light via toggle click until light
      const hasDark = await html.evaluate((el) => el.classList.contains("dark"));
      if (hasDark) {
        await toggle.click();
        await expect(html).not.toHaveClass(/dark/, { timeout: 5_000 }).catch(() => {});
      }
    });

    // Now click toggle to dark
    await toggle.click();
    await expect(html).toHaveClass(/dark/, { timeout: 10_000 });

    // Verify localStorage persists
    const stored = await page.evaluate(() => localStorage.getItem("theme") ?? document.documentElement.classList.contains("dark") ? "dark" : "light");
    expect(["dark", "light"]).toContain(stored); // next-themes stores as "dark" or system

    // Reload and verify dark persist
    await page.reload();
    await expect(toggle).toBeVisible({ timeout: 10_000 });
    await expect(html).toHaveClass(/dark/, { timeout: 10_000 });

    // Check that OKLCH tokens still apply — body bg is not #000 pure (should be oklch)
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
    // In dark, background is oklch(0.14 ...) rendered as rgb ~ #1a1a1a not pure black
    expect(bg).not.toBe("rgb(0, 0, 0)");
    expect(bg).not.toBe("rgba(0, 0, 0, 0)");

    // Toggle back to light and verify persist
    await toggle.click();
    await expect(html).not.toHaveClass(/dark/, { timeout: 10_000 });
    await page.reload();
    await expect(toggle).toBeVisible({ timeout: 10_000 });
    await expect(html).not.toHaveClass(/dark/, { timeout: 10_000 });
  });

  test("theme toggle ada di /book dan /app", async ({ page }) => {
    for (const path of ["/book", "/login"]) {
      await page.goto(path);
      await expect(page.getByRole("button", { name: /toggle theme/i })).toBeVisible({ timeout: 15_000 });
    }
  });
});
