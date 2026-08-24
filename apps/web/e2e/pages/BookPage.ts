import type { Page, Locator } from "@playwright/test";
import { expect } from "@playwright/test";

/**
 * Page Object for /book 4-step — uses getByRole, auto-wait, no sleep
 */
export class BookPage {
  constructor(public readonly page: Page) {}

  async goto() {
    await this.page.goto("/book");
    await expect(this.page.getByRole("heading", { name: /pilih layanan/i })).toBeVisible();
  }

  // Step1: pilih layanan via radio group
  async selectService(name: string | RegExp) {
    const matcher = typeof name === "string" ? name : name.source;
    // ServiceStep uses role="radio" with aria-label contains name
    const card = this.page.getByRole("radio", { name: typeof name === "string" ? new RegExp(name) : name }).first();
    await expect(card).toBeVisible();
    await card.click();
    // verify selected ring
    await expect(card).toHaveAttribute("aria-checked", "true");
  }

  async selectServiceByIndex(idx = 0) {
    const radios = this.page.getByRole("radio", { name: /Classic Cut|Premium Fade|Hair Color|Cut \+ Beard|Beard Trim|Grooming|Konsultasi/i });
    await expect(radios.first()).toBeVisible({ timeout: 10_000 });
    await radios.nth(idx).click();
  }

  // Step2 staff
  async selectStaff(name: string | null) {
    // Any available = null, else specific name
    if (name === null) {
      const any = this.page.getByRole("radio", { name: /any available/i });
      await expect(any).toBeVisible();
      await any.click();
    } else {
      const card = this.page.getByRole("radio", { name: new RegExp(`Pilih ${name}`) });
      await expect(card).toBeVisible();
      await card.click();
    }
  }

  async next() {
    const btn = this.page.getByRole("button", { name: /lanjut/i });
    await expect(btn).toBeEnabled();
    await btn.click();
  }

  async back() {
    await this.page.getByRole("button", { name: /kembali/i }).first().click();
  }

  // Step3 calendar — pilih tanggal + slot
  async waitForSlots() {
    // Legend must be visible — use first() to avoid strict mode (multiple matches durasi+buffer text)
    await expect(this.page.getByText("available").first()).toBeVisible();
    await expect(this.page.getByText("buffer").first()).toBeVisible();
    await expect(this.page.getByText("taken").first()).toBeVisible();
    // Wait for at least one slot button
    // Slots use button with aria-label contains waktu + tersedia/buffer/taken
    // Fallback: group Daftar slot visible
    const group = this.page.getByRole("group", { name: /daftar slot/i });
    await expect(group.or(this.page.getByText(/slot waktu/i))).toBeVisible({ timeout: 15_000 });
  }

  async selectSlot(timeLabel?: string | RegExp) {
    await this.waitForSlots();
    if (timeLabel) {
      const btn = this.page.getByRole("button", { name: typeof timeLabel === "string" ? new RegExp(timeLabel) : timeLabel }).first();
      await expect(btn).toBeVisible();
      await expect(btn).toBeEnabled();
      await btn.click();
    } else {
      // pick first available
      const btn = this.page.getByRole("button", { name: /tersedia/i }).first();
      await expect(btn).toBeVisible({ timeout: 10_000 });
      await btn.click();
    }
  }

  // Step4 form
  async fillForm(values: { name: string; email: string; phone?: string; notes?: string }) {
    await expect(this.page.getByRole("heading", { name: /detail kontak|isi detail/i })).toBeVisible({ timeout: 10_000 });
    const name = this.page.getByRole("textbox", { name: /nama lengkap/i });
    const email = this.page.getByRole("textbox", { name: /email/i });
    await name.fill(values.name);
    await email.fill(values.email);
    if (values.phone) {
      const phone = this.page.getByRole("textbox", { name: /no\. hp/i });
      await phone.fill(values.phone);
    }
    if (values.notes) {
      const notes = this.page.getByRole("textbox", { name: /catatan/i });
      await notes.fill(values.notes);
    }
  }

  async submitForm(expectPayment = false) {
    const btn = this.page.getByRole("button", { name: expectPayment ? /lanjut ke pembayaran/i : /konfirmasi|lanjut/i }).first();
    await expect(btn).toBeVisible();
    await btn.click();
  }

  // Helpers for assertions
  get stepper() {
    return this.page.getByRole("navigation", { name: /langkah booking/i });
  }

  get successHeading() {
    return this.page.getByRole("heading", { name: /booking dikonfirmasi|booking dibuat/i });
  }

  // Mock helpers
  async mockCreateBooking(page: Page, handler?: (route: import("@playwright/test").Route) => void) {
    // Default mock: intercept POST to /bookings (via ky prefixUrl)
    await page.route("**/bookings", async (route) => {
      if (handler) return handler(route);
      const req = route.request();
      if (req.method() === "POST") {
        const body = req.postDataJSON() as Record<string, unknown>;
        // Return CONFIRMED booking with ICS logic
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({
            id: "00000000-0000-4000-a000-000000000001",
            organizationId: "org-demo",
            serviceId: (body.serviceId as string) ?? "svc-classic",
            staffId: (body.staffId as string) ?? "staff-andi",
            customerName: body.customerName ?? "Budi Santoso",
            customerEmail: body.customerEmail ?? "budi@email.com",
            startAt: (body.startAt as string) ?? new Date(Date.now() + 86400000).toISOString(),
            endAt: new Date(new Date((body.startAt as string) ?? Date.now()).getTime() + 30 * 60000).toISOString(),
            status: "CONFIRMED",
            paymentStatus: "PAID",
            createdAt: new Date().toISOString(),
          }),
        });
      } else {
        await route.continue();
      }
    });
  }

  async mockSlots(page: Page, slots?: Array<{ startAt: string; available: boolean; reason?: string | null; staffName?: string }>) {
    // Intercept GET /availability/slots
    await page.route("**/availability/slots**", async (route) => {
      if (route.request().method() !== "GET") return route.continue();
      const url = new URL(route.request().url());
      const date = url.searchParams.get("date") ?? new Date().toISOString().slice(0, 10);
      const svc = url.searchParams.get("serviceId") ?? "svc-classic";
      // Use fallback logic if no custom slots
      if (slots) {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({ date, tz: "Asia/Jakarta", slots }),
        });
        return;
      }
      await route.continue();
    });
  }
}
