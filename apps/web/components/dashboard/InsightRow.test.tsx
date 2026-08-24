import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { InsightRow } from "./InsightRow";

describe("InsightRow — Busiest Des 2025, Cancel 7.2%", () => {
  it("render insight busiest month Des 2025 dan cancel 7.2%", () => {
    render(<InsightRow insights={{ busiestMonth: "Des 2025", busiestMonthCount: 180, busiestMonthRevenue: 14500000, cancelRate: 7.2, utilization: 68 }} kpi={{ totalBookings: 1542, confirmedBookings: 1420, totalRevenueCents: 142000000, avgTicketCents: 128000, occupancyPct: 68 } as any} />);
    expect(screen.getByTestId("insight-row")).toBeInTheDocument();
    expect(screen.getByText(/Des 2025/)).toBeInTheDocument();
    expect(screen.getByText(/7\.2%/)).toBeInTheDocument();
    expect(screen.getByText(/68%/)).toBeInTheDocument();
  });
  it("loading skeleton", () => {
    const { container } = render(<InsightRow isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
