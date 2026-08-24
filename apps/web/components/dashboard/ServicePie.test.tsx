import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ServicePie } from "./ServicePie";

describe("ServicePie — Pie Classic Cut 35%", () => {
  it("render Classic Cut 35% title dan legend", () => {
    const data = [
      { id: "svc-classic", name: "Classic Cut", bookingsCount: 540, percentage: 35, color: "oklch(0.62 0.19 260)" },
      { id: "svc-fade", name: "Premium Fade", bookingsCount: 320, percentage: 21, color: "oklch(0.55 0.16 260)" },
    ];
    render(<ServicePie data={data} />);
    expect(screen.getByText(/Classic Cut.*35%/i)).toBeInTheDocument();
    expect(screen.getByTestId("service-pie")).toBeInTheDocument();
  });

  it("skeleton saat loading", () => {
    const { container } = render(<ServicePie isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
