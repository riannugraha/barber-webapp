import * as React from "react";
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Heatmap } from "./Heatmap";

describe("Heatmap — 7x15", () => {
  it("render heatmap header dan grid", () => {
    const data = Array.from({ length: 105 }).map((_, i) => ({ dow: i % 7, hour: 7 + Math.floor(i / 7), count: 10 }));
    render(<Heatmap data={data} />);
    expect(screen.getByTestId("heatmap")).toBeInTheDocument();
    expect(screen.getByText(/Heatmap — Jam Sibuk 7×15/i)).toBeInTheDocument();
    expect(screen.getByText("Sen")).toBeInTheDocument();
    expect(screen.getByText(/07:00/)).toBeInTheDocument();
  });
  it("loading skeleton", () => {
    const { container } = render(<Heatmap isLoading />);
    expect(container.querySelector(".animate-pulse")).toBeTruthy();
  });
});
