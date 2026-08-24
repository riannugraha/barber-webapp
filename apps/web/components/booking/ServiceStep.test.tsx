import * as React from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ServiceStep } from "./ServiceStep";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    fetchServices: vi.fn().mockResolvedValue([
      { id: "svc-classic", organizationId: "org-demo", name: "Classic Cut", description: "Potongan klasik", durationMinutes: 30, bufferMinutes: 10, priceCents: 85000, color: "#7c3aed", isActive: true },
      { id: "svc-konsultasi", organizationId: "org-demo", name: "Konsultasi Style 15m", description: "Gratis", durationMinutes: 15, bufferMinutes: 5, priceCents: 0, color: "#64748b", isActive: true },
    ]),
  };
});

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("ServiceStep — Card layanan durasi/harga", () => {
  it("render radiogroup dengan layanan dan getByRole", async () => {
    render(
      <Wrapper>
        <ServiceStep selectedId={null} onSelect={vi.fn()} />
      </Wrapper>
    );
    const group = await screen.findByRole("radiogroup", { name: /pilih layanan/i });
    expect(group).toBeInTheDocument();
    expect(await screen.findByText("Classic Cut")).toBeInTheDocument();
    expect(await screen.findByText("Konsultasi Style 15m")).toBeInTheDocument();
    expect(screen.getByText(/30m/i)).toBeInTheDocument();
    expect(screen.getAllByText(/Gratis/i).length).toBeGreaterThan(0);
  });

  it("onSelect dipanggil saat click dan keyboard Enter", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    render(
      <Wrapper>
        <ServiceStep selectedId={null} onSelect={onSelect} />
      </Wrapper>
    );
    const radio = await screen.findByRole("radio", { name: /classic cut/i });
    await user.click(radio);
    expect(onSelect).toHaveBeenCalledWith("svc-classic", expect.objectContaining({ name: "Classic Cut" }));
    // keyboard
    onSelect.mockClear();
    radio.focus();
    await user.keyboard("{Enter}");
    expect(onSelect).toHaveBeenCalled();
  });

  it("selectedId menandai aria-checked true", async () => {
    render(
      <Wrapper>
        <ServiceStep selectedId="svc-classic" onSelect={vi.fn()} />
      </Wrapper>
    );
    const radio = await screen.findByRole("radio", { name: /classic cut/i });
    expect(radio).toHaveAttribute("aria-checked", "true");
  });
});
