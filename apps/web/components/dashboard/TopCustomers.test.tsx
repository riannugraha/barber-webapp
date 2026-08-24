import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { TopCustomers } from "./TopCustomers";

describe("TopCustomers — Siti 18x", () => {
  it("render top customer Siti Rahayu 18x", () => {
    const data = [
      { customerName: "Siti Rahayu", customerEmail: "siti@example.com", bookingsCount: 18, totalSpentCents: 2304000 },
      { customerName: "Budi Santoso", customerEmail: "budi@example.com", bookingsCount: 14, totalSpentCents: 1792000 },
    ];
    render(<TopCustomers data={data} />);
    expect(screen.getByTestId("top-customers")).toBeInTheDocument();
    expect(screen.getAllByText(/Siti Rahayu/).length).toBeGreaterThan(0);
    expect(screen.getAllByText("18×").length).toBeGreaterThan(0);
  });
  it("loading skeleton", () => {
    const { container } = render(<TopCustomers isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
