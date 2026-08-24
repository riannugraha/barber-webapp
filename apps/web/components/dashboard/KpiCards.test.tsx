import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { KpiCards } from "./KpiCards";

const mockKpi = {
  totalBookings: 1542,
  confirmedBookings: 1420,
  totalRevenueCents: 142000000,
  avgTicketCents: 128000,
  occupancyPct: 68,
  deltaRevenuePct: 9,
  deltaBookingsPct: 4,
};

describe("KpiCards — Row1 4 KPI AC PLAN T11", () => {
  it("renders 4 KPI with AC exact values tabular-nums click /app/bookings", () => {
    render(<KpiCards kpi={mockKpi} from="2025-11-01" to="2026-08-24" />);
    // Revenue Rp 142jt
    expect(screen.getByTestId("kpi-revenue")).toHaveTextContent(/142/);
    expect(screen.getByTestId("kpi-revenue").className).toMatch(/tabular-nums/);
    // Bookings 1.542 (id-ID format)
    expect(screen.getByTestId("kpi-bookings")).toHaveTextContent("1.542");
    // Occupancy 68%
    expect(screen.getByTestId("kpi-occupancy")).toHaveTextContent("68%");
    // Avg Ticket Rp 128k
    expect(screen.getByTestId("kpi-avg-ticket")).toHaveTextContent(/128k/);

    // delta +9% tabular-nums
    expect(screen.getByText("+9%").className).toMatch(/tabular-nums/);
    expect(screen.getByText("+4%")).toBeInTheDocument();

    // links to /app/bookings?from&to
    const links = screen.getAllByRole("link");
    expect(links[0].getAttribute("href")).toContain("/app/bookings?from=2025-11-01&to=2026-08-24");

    // 4→2x2 mobile grid
    const row = screen.getByTestId("kpi-row");
    expect(row.className).toMatch(/grid-cols-2/);
    expect(row.className).toMatch(/lg:grid-cols-4/);
  });

  it("shows skeleton when loading", () => {
    const { container } = render(<KpiCards isLoading kpi={undefined} />);
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(0);
  });
});
