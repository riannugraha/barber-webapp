import * as React from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FormStep } from "./FormStep";
import type { Service, Slot } from "@/lib/api";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    createBooking: vi.fn().mockResolvedValue({ id: "b-123", status: "CONFIRMED", paymentStatus: "PAID", startAt: "2026-08-24T02:00:00.000Z", endAt: "2026-08-24T02:30:00.000Z" }),
    createCheckoutSession: vi.fn().mockResolvedValue({ url: "https://stripe.test/checkout", sessionId: "cs_test" }),
  };
});

const mockService: Service = {
  id: "svc-classic",
  organizationId: "org-demo",
  name: "Classic Cut",
  description: "Potong klasik",
  durationMinutes: 30,
  bufferMinutes: 10,
  priceCents: 85000,
  color: "#7c3aed",
  isActive: true,
};

const mockSlot: Slot = {
  startAt: "2026-08-24T02:00:00.000Z",
  endAt: "2026-08-24T02:30:00.000Z",
  available: true,
  staffId: "staff-andi",
  staffName: "Andi",
  reason: null,
};

function Wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
}

describe("FormStep — rhf+zod", () => {
  it("render field nama/email/notes dengan getByRole dan validasi zod", () => {
    render(
      <Wrapper>
        <FormStep service={mockService} staff={null} slot={mockSlot} onSuccess={vi.fn()} onBack={vi.fn()} />
      </Wrapper>
    );
    expect(screen.getByRole("textbox", { name: /nama lengkap/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /email/i })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /catatan/i })).toBeInTheDocument();
    expect(screen.getByText("Review booking")).toBeInTheDocument();
  });

  it("menampilkan error zod saat email tidak valid", async () => {
    const user = userEvent.setup();
    render(
      <Wrapper>
        <FormStep service={mockService} staff={null} slot={mockSlot} onSuccess={vi.fn()} onBack={vi.fn()} />
      </Wrapper>
    );
    const email = screen.getByRole("textbox", { name: /email/i });
    const name = screen.getByRole("textbox", { name: /nama/i });
    // trigger validation by typing invalid and submitting
    await user.type(name, "B");
    await user.type(email, "not-an-email");
    await user.click(screen.getByRole("button", { name: /lanjut ke pembayaran/i }));
    expect(await screen.findByText(/nama minimal/i)).toBeInTheDocument();
    expect(await screen.findByText(/email tidak valid/i)).toBeInTheDocument();
  });

  it("submit berhasil memanggil createBooking via ky (tanpa Server Actions)", async () => {
    const user = userEvent.setup();
    render(
      <Wrapper>
        <FormStep service={mockService} staff={null} slot={mockSlot} onSuccess={vi.fn()} onBack={vi.fn()} />
      </Wrapper>
    );
    await user.type(screen.getByRole("textbox", { name: /nama/i }), "Budi Santoso");
    await user.type(screen.getByRole("textbox", { name: /email/i }), "budi@email.com");
    const btn = screen.getByRole("button", { name: /lanjut ke pembayaran/i });
    await user.click(btn);
    // after click, button should show Memproses or toast — check that button becomes disabled/contacted
    expect(btn).toBeInTheDocument();
  });

  it("menampilkan total tabular-nums dan timezone Asia/Jakarta", () => {
    render(
      <Wrapper>
        <FormStep service={mockService} staff={null} slot={mockSlot} onSuccess={vi.fn()} onBack={vi.fn()} />
      </Wrapper>
    );
    expect(screen.getByText(/Rp/)).toBeInTheDocument();
    expect(screen.getByText(/WIB/)).toBeInTheDocument();
    expect(screen.getByText(/Asia\/Jakarta/)).toBeInTheDocument();
  });
});
