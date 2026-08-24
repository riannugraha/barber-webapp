import * as React from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StaffStep } from "./StaffStep";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    fetchStaff: vi.fn().mockResolvedValue([
      { id: "staff-andi", organizationId: "org-demo", name: "Andi", email: "andi@flowbook.test", avatarUrl: null, isActive: true },
      { id: "staff-bayu", organizationId: "org-demo", name: "Bayu", email: "bayu@flowbook.test", avatarUrl: null, isActive: true },
    ]),
  };
});

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("StaffStep — Avatar staff + Any available", () => {
  it("render Any available + staff list via getByRole", async () => {
    render(
      <Wrapper>
        <StaffStep serviceId={null} selectedId={null} onSelect={vi.fn()} />
      </Wrapper>
    );
    expect(await screen.findByRole("radio", { name: /any available/i })).toBeInTheDocument();
    expect(await screen.findByText("Andi")).toBeInTheDocument();
    expect(await screen.findByText("Bayu")).toBeInTheDocument();
    const andiRadio = screen.getByRole("radio", { name: /pilih andi/i });
    expect(andiRadio).toBeInTheDocument();
    expect(andiRadio).toHaveAttribute("aria-label", expect.stringContaining("Andi"));
  });

  it("select staff memanggil onSelect dengan id", async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    render(
      <Wrapper>
        <StaffStep serviceId="svc-classic" selectedId={null} onSelect={onSelect} />
      </Wrapper>
    );
    const andi = await screen.findByRole("radio", { name: /pilih andi/i });
    await user.click(andi);
    expect(onSelect).toHaveBeenCalledWith("staff-andi", expect.objectContaining({ name: "Andi" }));
    // Any available
    const any = screen.getByRole("radio", { name: /any available/i });
    await user.click(any);
    expect(onSelect).toHaveBeenCalledWith(null, null);
  });

  it("selectedId null menandai Any available checked", async () => {
    render(
      <Wrapper>
        <StaffStep serviceId={null} selectedId={null} onSelect={vi.fn()} />
      </Wrapper>
    );
    const any = await screen.findByRole("radio", { name: /any available/i });
    expect(any).toHaveAttribute("aria-checked", "true");
  });
});
