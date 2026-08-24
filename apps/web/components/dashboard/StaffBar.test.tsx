import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StaffBar } from "./StaffBar";

describe("StaffBar — Bar Andi 90/Bayu 70/Sari 20", () => {
  it("render staff bar dengan data Andi 90 Bayu 70 Sari 20", () => {
    const data = [
      { id: "staff-andi", name: "Andi", bookingsCount: 90, revenueCents: 8100000 },
      { id: "staff-bayu", name: "Bayu", bookingsCount: 70, revenueCents: 6300000 },
      { id: "staff-sari", name: "Sari", bookingsCount: 20, revenueCents: 1800000 },
    ];
    render(<StaffBar data={data} />);
    expect(screen.getByTestId("staff-bar")).toBeInTheDocument();
    // Title is in h3, use getByRole heading
    expect(screen.getByRole("heading", { name: /Andi 90.*Bayu 70.*Sari 20/i })).toBeInTheDocument();
  });

  it("loading skeleton", () => {
    const { container } = render(<StaffBar isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
