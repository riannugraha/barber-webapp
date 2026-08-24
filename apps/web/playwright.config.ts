import { defineConfig, devices } from "@playwright/test";

/**
 * Playwright config per PLAN.md T14 AC — Opsi A
 * - workers:1 di CI (pooler 60 conns, Koyeb 0.1 vCPU), fullyParallel lokal
 * - retries 2 di CI, trace on-first-retry, screenshot only-on-failure
 * - webServer pnpm --filter web dev, chromium only, baseURL localhost:3000
 * - bypass x-vercel-protection-bypass jika aktif (preview)
 */
const PORT = Number(process.env.PORT ?? 3000);
const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? `http://localhost:${PORT}`;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: !process.env.CI, // lokal fullyParallel true, CI workers:1 so false
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined, // pooler 60, Koyeb 0.1 vCPU — must be 1 in CI
  reporter: process.env.CI ? [["html", { open: "never" }], ["list"]] : "html",
  timeout: 30_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
    // bypass Vercel protection header if set
    extraHTTPHeaders: process.env.VERCEL_PROTECTION_BYPASS
      ? { "x-vercel-protection-bypass": process.env.VERCEL_PROTECTION_BYPASS }
      : undefined,
  },
  webServer: {
    command: "pnpm --filter web dev",
    url: BASE_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    env: {
      NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1",
      PORT: String(PORT),
    },
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
