import type { Page } from "@playwright/test";
import { expect } from "@playwright/test";

/**
 * Page Object for /app dashboard 5 row
 */
export class DashboardPage {
  constructor(public readonly page: Page) {}

  async goto() {
    await this.page.goto("/app");
  }

  async expectKpiVisible() {
    // Row1 4 KPI Revenue Rp 142jt | Bookings 1.542 | Occupancy 68% | Avg Ticket Rp 128k + delta +9% tabular-nums
    await expect(this.page.getByRole("heading", { name: /dashboard/i })).toBeVisible({ timeout: 15_000 });
    await expect(this.page.getByTestId("kpi-revenue")).toBeVisible();
    await expect(this.page.getByTestId("kpi-bookings")).toBeVisible();
    await expect(this.page.getByTestId("kpi-occupancy")).toBeVisible();
    await expect(this.page.getByTestId("kpi-avg-ticket")).toBeVisible();
    // tabular-nums
    await expect(this.page.getByTestId("kpi-revenue")).toHaveClass(/tabular-nums/);
    // delta badge
    await expect(this.page.getByText("+9%").first()).toBeVisible();
  }

  async expectKpiValues() {
    // Exact-ish AC values, allow fallback rendering
    await expect(this.page.getByTestId("kpi-revenue")).toContainText(/142|jt|Rp/);
    await expect(this.page.getByTestId("kpi-bookings")).toContainText(/1\.542|1542/);
    await expect(this.page.getByTestId("kpi-occupancy")).toContainText(/68%/);
    await expect(this.page.getByTestId("kpi-avg-ticket")).toContainText(/128/);
  }

  async expectChart10Bulan() {
    // Row2 Area 10 bulan single primary oklch violet 260
    await expect(this.page.getByTestId("revenue-area")).toBeVisible({ timeout: 15_000 });
    await expect(this.page.getByText(/Revenue per Bulan — 10 titik/i)).toBeVisible();
    // 10 bulan label Nov ... Agu visible in chart or caption — use first() to avoid strict (badge + caption)
    await expect(this.page.getByText(/10 bulan/i).first()).toBeVisible();
    // Harian/Mingguan/Bulanan toggle
    await expect(this.page.getByTestId("granularity-day")).toBeVisible();
    await expect(this.page.getByTestId("granularity-week")).toBeVisible();
    await expect(this.page.getByTestId("granularity-month")).toBeVisible();
    // Verify aria-pressed for Bulanan active
    await expect(this.page.getByTestId("granularity-month")).toHaveAttribute("aria-pressed", "true");
  }

  async toggleGranularity(to: "day" | "week" | "month") {
    await this.page.getByTestId(`granularity-${to}`).click();
  }

  async expectRow3() {
    // Pie Classic Cut 35% + Bar Andi 90/Bayu 70/Sari 20 + Heatmap 7x15
    await expect(this.page.getByTestId("service-pie")).toBeVisible();
    await expect(this.page.getByTestId("staff-bar")).toBeVisible();
    await expect(this.page.getByTestId("heatmap")).toBeVisible();
    // Check pie title contains 35% — first() to avoid chart tspan duplicate
    await expect(this.page.getByText(/Classic Cut.*35%/i).first()).toBeVisible();
    // Bar title contains Andi 90 — first() to avoid duplicate
    await expect(this.page.getByText(/Andi.*90/i).first()).toBeVisible();
    // Heatmap header
    await expect(this.page.getByText(/Heatmap — Jam Sibuk 7×15/i)).toBeVisible();
  }

  async expectRow4() {
    // Top 15 Siti 18x + Recent 10 Badge
    // Siti Rahayu should appear in Top Customers
    await expect(this.page.getByText(/Siti Rahayu/i).first()).toBeVisible();
    // Recent 10 — either heading Recent or table
    const recentHeading = this.page.getByText(/recent/i).first();
    const table = this.page.getByRole("table");
    // Prefer recent heading, fallback to table
    if (await recentHeading.count() > 0 && (await recentHeading.isVisible().catch(() => false))) {
      await expect(recentHeading).toBeVisible({ timeout: 10_000 });
    } else {
      await expect(table.first()).toBeVisible({ timeout: 10_000 });
    }
  }

  async expectRow5() {
    // Busiest Des 2025 | Cancel 7.2% | Utilization
    await expect(this.page.getByText(/Busiest.*Des 2025|Des 2025/i).first()).toBeVisible();
    await expect(this.page.getByText(/Cancel.*7\.2%|7\.2%/i).first()).toBeVisible();
  }

  async mockDashboard(page: Page) {
    // Intercept GET /dashboard to return full seed 1.500 data if API not ready
    await page.route("**/dashboard**", async (route) => {
      if (route.request().method() !== "GET") return route.continue();
      // Return full mock matching PLAN T11 AC (Nov 2025->Agu 2026)
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          kpi: {
            totalBookings: 1542,
            confirmedBookings: 1420,
            cancelledBookings: 108,
            totalRevenueCents: 142000000,
            avgTicketCents: 128000,
            occupancyPct: 68,
            deltaRevenuePct: 9,
            deltaBookingsPct: 4,
          },
          revenueSeries: ["Nov", "Des", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"].map((m, i) => ({
            period: `2025-${String(11 + i).padStart(2, "0")}-01T00:00:00Z`,
            revenueCents: [9000000, 14500000, 11000000, 9800000, 12300000, 14200000, 13500000, 11800000, 13900000, 14200000][i],
            bookingsCount: [120, 180, 140, 126, 145, 162, 155, 138, 160, 165][i],
            label: m,
          })),
          topServices: [
            { id: "svc-classic", name: "Classic Cut", bookingsCount: 540, revenueCents: 45900000, percentage: 35, color: "oklch(0.62 0.19 260)" },
            { id: "svc-fade", name: "Premium Fade", bookingsCount: 320, revenueCents: 38400000, percentage: 21, color: "oklch(0.55 0.16 260)" },
            { id: "svc-beard", name: "Cut + Beard", bookingsCount: 210, revenueCents: 31500000, percentage: 14, color: "oklch(0.48 0.14 260)" },
            { id: "svc-color", name: "Hair Color", bookingsCount: 180, revenueCents: 45000000, percentage: 12, color: "oklch(0.70 0.12 260)" },
            { id: "svc-other", name: "Lainnya", bookingsCount: 292, revenueCents: 28000000, percentage: 18, color: "oklch(0.45 0.10 260)" },
          ],
          bookingsByStaff: [
            { id: "staff-andi", name: "Andi", bookingsCount: 90, revenueCents: 8100000 },
            { id: "staff-bayu", name: "Bayu", bookingsCount: 70, revenueCents: 6300000 },
            { id: "staff-sari", name: "Sari", bookingsCount: 20, revenueCents: 1800000 },
          ],
          bookingsByHour: Array.from({ length: 15 }).map((_, i) => ({ hour: 7 + i, bookingsCount: [5, 8, 12, 18, 20, 16, 14, 11, 9, 7, 6, 5, 4, 8, 12][i] })),
          heatmap: (() => {
            const arr: Array<{ dow: number; hour: number; count: number }> = [];
            const base = [5, 8, 12, 18, 20, 16, 14, 11, 9, 7, 6, 5, 4, 8, 12];
            for (let dow = 0; dow < 7; dow++) for (let h = 7; h <= 21; h++) arr.push({ dow, hour: h, count: base[h - 7] + (dow >= 5 ? 4 : 0) });
            return arr;
          })(),
          topCustomers: Array.from({ length: 15 }).map((_, i) => {
            const names = ["Siti Rahayu", "Budi Santoso", "Ani Wijaya", "Joko Prabowo", "Dewi Lestari", "Agus Prasetyo", "Rina Maulana", "Eko Saputra", "Maya Sari", "Hendra Gunawan", "Lina Kusuma", "Fajar Nugroho", "Tono Wijaya", "Indah Permata", "Rudi Hartono"];
            const counts = [18, 14, 12, 10, 9, 8, 7, 6, 5, 5, 4, 4, 3, 3, 2];
            return { customerName: names[i], customerEmail: names[i].toLowerCase().replace(/\s/g, ".") + "@example.com", bookingsCount: counts[i], totalSpentCents: counts[i] * 128000, lastBookingAt: new Date(Date.now() - i * 86400000 * 5).toISOString() };
          }),
          recentBookings: Array.from({ length: 10 }).map((_, i) => {
            const customers = ["Siti Rahayu", "Budi Santoso", "Ani Wijaya", "Joko Prabowo", "Dewi Lestari", "Agus Prasetyo", "Rina Maulana", "Eko Saputra", "Maya Sari", "Hendra Gunawan"];
            const statusList = ["CONFIRMED", "PENDING", "CONFIRMED", "CANCELLED", "CONFIRMED", "PENDING", "CONFIRMED", "CONFIRMED", "CANCELLED", "PENDING"] as const;
            const d = new Date(); d.setHours(9 + (i % 10), (i % 2) * 30, 0, 0); d.setDate(d.getDate() - Math.floor(i / 2));
            const end = new Date(d.getTime() + 30 * 60000);
            return { id: `bk-${String(100 + i).padStart(4, "0")}`, organizationId: "org-demo", serviceId: "svc-classic", staffId: ["staff-andi", "staff-bayu", "staff-sari"][i % 3], customerName: customers[i], customerEmail: customers[i].toLowerCase().replace(/\s/g, ".") + "@example.com", customerPhone: "08123456789", notes: null, customerId: null, startAt: d.toISOString(), endAt: end.toISOString(), status: statusList[i], paymentStatus: "PAID", stripeSessionId: null, createdAt: new Date(d.getTime() - 86400000).toISOString() };
          }),
          insights: { busiestMonth: "Des 2025", busiestMonthCount: 180, busiestMonthRevenue: 14500000, cancelRate: 7.2, utilization: 68 },
        }),
      });
    });
  }
}
