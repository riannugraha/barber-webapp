import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { RecentTable } from "./RecentTable";

describe("RecentTable — Recent 10", () => {
  it("render recent 10 bookings badge", () => {
    const data = Array.from({ length: 3 }).map((_, i) => ({
      id: `bk-000${i}`,
      organizationId: "org-demo",
      serviceId: "svc-classic",
      staffId: "staff-andi",
      customerName: i === 0 ? "Siti Rahayu" : i === 1 ? "Budi Santoso" : "Ani Wijaya",
      customerEmail: `user${i}@example.com`,
      customerPhone: "0812",
      notes: null,
      customerId: null,
      startAt: new Date().toISOString(),
      endAt: new Date(Date.now() + 1800000).toISOString(),
      status: "CONFIRMED" as const,
      paymentStatus: "PAID" as const,
      stripeSessionId: null,
      createdAt: new Date().toISOString(),
    }));
    render(<RecentTable data={data as any} />);
    expect(screen.getByTestId("recent-table")).toBeInTheDocument();
    expect(screen.getByText(/Recent 10/)).toBeInTheDocument();
    expect(screen.getByText("Siti Rahayu")).toBeInTheDocument();
    expect(screen.getAllByText("CONFIRMED").length).toBeGreaterThan(0);
  });
  it("loading skeleton", () => {
    const { container } = render(<RecentTable isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
