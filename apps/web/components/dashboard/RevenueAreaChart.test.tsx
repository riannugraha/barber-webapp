import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RevenueAreaChart } from "./RevenueAreaChart";

const mockSeries = ["Nov", "Des", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"].map((m, i) => ({
  period: `2025-${String(11 + i).padStart(2, "0")}-01T00:00:00Z`,
  revenueCents: [9000000, 14500000, 11000000, 9800000, 12300000, 14200000, 13500000, 11800000, 13900000, 14200000][i],
  bookingsCount: [120, 180, 140, 126, 145, 162, 155, 138, 160, 165][i],
  label: m,
}));

describe("RevenueAreaChart — Row2 AC", () => {
  it("renders title 10 titik + single primary + toggle Harian/Mingguan/Bulanan", async () => {
    let granularity: "day" | "week" | "month" = "month";
    const onChange = (g: typeof granularity) => (granularity = g);
    render(<RevenueAreaChart data={mockSeries} granularity="month" onGranularityChange={onChange} />);

    expect(screen.getByText(/Revenue per Bulan — 10 titik/)).toBeInTheDocument();
    expect(screen.getByText(/Single primary/)).toBeInTheDocument();
    expect(screen.getByTestId("granularity-day")).toHaveTextContent("Harian");
    expect(screen.getByTestId("granularity-week")).toHaveTextContent("Mingguan");
    expect(screen.getByTestId("granularity-month")).toHaveTextContent("Bulanan");

    // isAnimationActive is not easily testable via DOM, but Recharts Area should be in doc
    expect(screen.getByTestId("revenue-area")).toBeInTheDocument();

    // toggle click
    fireEvent.click(screen.getByTestId("granularity-day"));
    expect(granularity).toBe("day");

    fireEvent.click(screen.getByTestId("granularity-week"));
    expect(granularity).toBe("week");
  });

  it("has overflow-x-auto + min-w-560 for horizontal scroll on mobile", () => {
    const { container } = render(<RevenueAreaChart data={mockSeries} granularity="month" onGranularityChange={() => {}} />);
    const scroll = container.querySelector(".overflow-x-auto");
    expect(scroll).toBeTruthy();
    expect(scroll?.querySelector(".min-w-\\[560px\\]")).toBeTruthy();
  });
});
