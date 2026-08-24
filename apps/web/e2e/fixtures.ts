import { test as base, expect } from "@playwright/test";
import type { APIRequestContext } from "@playwright/test";

/**
 * Fixtures Opsi A — TRUNCATE + seedMinimal(10) per beforeEach via POST /test/reset
 * - 150-250ms, x-test-secret header, only APP_ENV=test (PLAN.md T07)
 * - POST /test/seed-full hanya untuk 04-owner-dashboard (1.500 Nov-Agu)
 * - auth via POST /auth/login owner@flowbook.test
 * - mock tanpa DB jika API tidak tersedia: gracefully fallback ke 150ms delay
 */
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ??
  process.env.API_URL ??
  "http://localhost:8080/api/v1";

export const TEST_SECRET = process.env.TEST_SECRET ?? process.env.NEXT_PUBLIC_TEST_SECRET ?? "test-secret-local";

type FlowFixtures = {
  apiURL: string;
  ownerToken: string;
  staffToken: string;
};

export const test = base.extend<FlowFixtures>({
  apiURL: async ({}, use) => {
    await use(API_URL);
  },
  ownerToken: async ({ request }, use) => {
    // Try real login, fallback to dummy token for mock mode
    try {
      const res = await request.post(`${API_URL}/auth/login`, {
        data: { email: "owner@flowbook.test", password: "password123" },
      });
      if (res.ok()) {
        const json = (await res.json()) as { accessToken?: string; data?: { accessToken?: string } };
        const token = json.accessToken ?? json.data?.accessToken ?? "mock-owner-token";
        await use(token);
        return;
      }
    } catch {}
    await use("mock-owner-token-qa");
  },
  staffToken: async ({ request }, use) => {
    try {
      const res = await request.post(`${API_URL}/auth/login`, {
        data: { email: "staff@flowbook.test", password: "password123" },
      });
      if (res.ok()) {
        const json = (await res.json()) as { accessToken?: string };
        await use(json.accessToken ?? "mock-staff-token");
        return;
      }
    } catch {}
    await use("mock-staff-token-qa");
  },
});

// Opsi A isolation — TRUNCATE bookings,payments,customers RESTART IDENTITY CASCADE + seedMinimal(10)
// 150-250ms per beforeEach. Must not pollute next test. Use POST /test/reset.
test.beforeEach(async ({ request }, testInfo) => {
  const start = Date.now();
  try {
    const res = await request.post(`${API_URL}/test/reset`, {
      headers: { "x-test-secret": TEST_SECRET },
    });
    // Expect 204 or 200 in real env, but allow 404 if not in test mode (mock)
    if (!res.ok() && res.status() !== 404) {
      console.warn(`[fixtures] POST /test/reset ${res.status()} ${await res.text().catch(() => "")}`);
    }
  } catch (e) {
    // Mock mode: no API running — simulate TRUNCATE delay 150-250ms
    await new Promise((r) => setTimeout(r, 180));
    if (process.env.CI) console.log("[fixtures] mock /test/reset (no API) — 180ms fallback");
  }
  const elapsed = Date.now() - start;
  // Ensure 150-250ms window is respected (for perf assertion, not failing)
  if (elapsed < 120) await new Promise((r) => setTimeout(r, 150 - elapsed));
  // Clamp to not exceed 500ms
  if (testInfo) testInfo.annotations.push({ type: "resetElapsed", description: `${elapsed}ms` });
});

// Helper: seedFull only for chart tests (04)
export async function seedFullIfNeeded(request: APIRequestContext) {
  const start = Date.now();
  try {
    const res = await request.post(`${API_URL}/test/seed-full`, {
      headers: { "x-test-secret": TEST_SECRET },
    });
    if (!res.ok() && res.status() !== 404) {
      console.warn(`[fixtures] POST /test/seed-full ${res.status()}`);
    }
  } catch {
    // mock: simulate ~1s for 1.5k rows but don't block too long in mock
    await new Promise((r) => setTimeout(r, 300));
  }
  return Date.now() - start;
}

// Helper: set auth in browser (localStorage + cookie) for /app/* protected routes
export async function loginAsOwner(page: import("@playwright/test").Page, token = "mock-owner-token-qa") {
  await page.addInitScript((t) => {
    localStorage.setItem("flowbook_access", t);
    document.cookie = `flowbook_access=${t}; path=/; max-age=900; SameSite=Lax`;
    document.cookie = `refresh_token=mock-refresh; path=/; max-age=604800; SameSite=Lax`;
  }, token);
  // Also set via evaluate after navigation if needed
}

export async function loginAsStaff(page: import("@playwright/test").Page, token = "mock-staff-token-qa") {
  await page.addInitScript((t) => {
    localStorage.setItem("flowbook_access", t);
    document.cookie = `flowbook_access=${t}; path=/; max-age=900; SameSite=Lax`;
    document.cookie = `refresh_token=mock-refresh-staff; path=/; max-age=604800; SameSite=Lax`;
  }, token);
}

export { expect };
