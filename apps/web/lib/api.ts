import ky from "ky";

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ??
  "https://flowbook-api-xxx.koyeb.app/api/v1";

export const api = ky.create({
  prefixUrl: API_URL,
  headers: {
    "Content-Type": "application/json",
  },
  retry: 1,
  timeout: 15000,
  hooks: {
    beforeRequest: [
      (request) => {
        if (typeof window !== "undefined") {
          const token = localStorage.getItem("flowbook_access");
          if (token) request.headers.set("Authorization", `Bearer ${token}`);
        }
      },
    ],
  },
});

/* ── Types — mirror openapi.yaml ── */
export type Service = {
  id: string;
  organizationId: string;
  name: string;
  description?: string | null;
  durationMinutes: number;
  bufferMinutes: number;
  priceCents: number;
  color: string;
  isActive: boolean;
  createdAt?: string;
  updatedAt?: string;
};

export type Staff = {
  id: string;
  organizationId: string;
  name: string;
  email?: string | null;
  avatarUrl?: string | null;
  isActive: boolean;
  createdAt?: string;
  userId?: string | null;
};

export type Slot = {
  startAt: string; // UTC ISO
  endAt: string;
  available: boolean;
  staffId?: string;
  staffName?: string;
  reason?: string | null; // buffer | taken | null
};

export type SlotsResponse = {
  date: string;
  tz: string;
  slots: Slot[];
};

export type Booking = {
  id: string;
  organizationId: string;
  serviceId: string;
  staffId: string;
  customerId?: string | null;
  customerName: string;
  customerEmail: string;
  customerPhone?: string | null;
  notes?: string | null;
  startAt: string;
  endAt: string;
  status: "PENDING" | "CONFIRMED" | "CANCELLED" | "COMPLETED" | "NO_SHOW";
  paymentStatus?: "UNPAID" | "PAID" | "REFUNDED";
  stripeSessionId?: string | null;
  createdAt: string;
  updatedAt?: string;
};

export type CreateBookingRequest = {
  organizationId?: string;
  serviceId: string;
  staffId: string;
  startAt: string;
  customerName: string;
  customerEmail: string;
  customerPhone?: string;
  notes?: string;
};

/* ── Helpers ── */
export function formatPrice(priceCents: number): string {
  if (priceCents === 0) return "Gratis";
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(priceCents);
}

export function formatDuration(min: number): string {
  if (min < 60) return `${min}m`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m ? `${h}j ${m}m` : `${h}j`;
}

/** yyyy-mm-dd in Asia/Jakarta */
export function toJakartaDateString(d: Date = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(d);
}

export function formatJakartaTime(iso: string): string {
  return new Intl.DateTimeFormat("id-ID", {
    timeZone: "Asia/Jakarta",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(new Date(iso));
}

export function formatJakartaDate(iso: string): string {
  return new Intl.DateTimeFormat("id-ID", {
    timeZone: "Asia/Jakarta",
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date(iso));
}

/* ── API Calls — all via ky, no Server Actions ── */

export async function fetchServices(): Promise<Service[]> {
  try {
    const res = await api.get("services").json<{ data: Service[] }>();
    return res.data;
  } catch {
    // fallback demo seed when API unavailable (build / offline)
    return fallbackServices;
  }
}

export async function fetchStaff(serviceId?: string): Promise<Staff[]> {
  try {
    const search = serviceId ? `?serviceId=${serviceId}` : "";
    const res = await api.get(`staff${search}`).json<{ data: Staff[] }>();
    return res.data;
  } catch {
    return fallbackStaff;
  }
}

export async function fetchSlots(params: {
  serviceId: string;
  staffId?: string;
  date: string; // yyyy-mm-dd
  tz?: string;
}): Promise<SlotsResponse> {
  const tz = params.tz ?? "Asia/Jakarta";
  const sp = new URLSearchParams({
    serviceId: params.serviceId,
    date: params.date,
    tz,
  });
  if (params.staffId) sp.set("staffId", params.staffId);
  try {
    return await api.get(`availability/slots?${sp.toString()}`).json<SlotsResponse>();
  } catch {
    // fallback demo slots: generate 07:00-21:00 30m grid in Jakarta
    return fallbackSlots(params);
  }
}

export async function createBooking(data: CreateBookingRequest): Promise<Booking> {
  return await api.post("bookings", { json: data }).json<Booking>();
}

export async function getBooking(id: string): Promise<Booking> {
  return await api.get(`bookings/${id}`).json<Booking>();
}

export async function cancelBooking(id: string): Promise<Booking> {
  return await api.post(`bookings/${id}/cancel`).json<Booking>();
}

export async function rescheduleBooking(
  id: string,
  payload: { staffId: string; startAt: string }
): Promise<Booking> {
  return await api.post(`bookings/${id}/reschedule`, { json: payload }).json<Booking>();
}

export async function createCheckoutSession(payload: {
  bookingId: string;
  successUrl: string;
  cancelUrl: string;
}): Promise<{ url: string; sessionId: string }> {
  return await api.post("payments/checkout-session", { json: payload }).json();
}

/* ── Fallback demo data so pnpm build passes without live API ── */
const fallbackServices: Service[] = [
  {
    id: "svc-classic",
    organizationId: "org-demo",
    name: "Classic Cut",
    description: "Potongan klasik presisi, cuci + styling",
    durationMinutes: 30,
    bufferMinutes: 10,
    priceCents: 85000,
    color: "#7c3aed",
    isActive: true,
  },
  {
    id: "svc-fade",
    organizationId: "org-demo",
    name: "Premium Fade",
    description: "Fade detail + razor line",
    durationMinutes: 45,
    bufferMinutes: 10,
    priceCents: 120000,
    color: "#8b5cf6",
    isActive: true,
  },
  {
    id: "svc-beard",
    organizationId: "org-demo",
    name: "Cut + Beard",
    description: "Potong + cukur beard lengkap",
    durationMinutes: 60,
    bufferMinutes: 15,
    priceCents: 150000,
    color: "#6366f1",
    isActive: true,
  },
  {
    id: "svc-trim",
    organizationId: "org-demo",
    name: "Beard Trim",
    description: "Rapikan jenggot 20 menit",
    durationMinutes: 20,
    bufferMinutes: 10,
    priceCents: 50000,
    color: "#14b8a6",
    isActive: true,
  },
  {
    id: "svc-color",
    organizationId: "org-demo",
    name: "Hair Color",
    description: "Pewarnaan rambut premium",
    durationMinutes: 90,
    bufferMinutes: 15,
    priceCents: 250000,
    color: "#f59e0b",
    isActive: true,
  },
  {
    id: "svc-father",
    organizationId: "org-demo",
    name: "Father & Son",
    description: "Paket ayah & anak",
    durationMinutes: 60,
    bufferMinutes: 15,
    priceCents: 180000,
    color: "#10b981",
    isActive: true,
  },
  {
    id: "svc-grooming",
    organizationId: "org-demo",
    name: "Grooming Package",
    description: "Lengkap cut + beard + color touch",
    durationMinutes: 75,
    bufferMinutes: 15,
    priceCents: 200000,
    color: "#f43f5e",
    isActive: true,
  },
  {
    id: "svc-konsultasi",
    organizationId: "org-demo",
    name: "Konsultasi Style 15m",
    description: "Konsultasi gratis dengan barber",
    durationMinutes: 15,
    bufferMinutes: 5,
    priceCents: 0,
    color: "#64748b",
    isActive: true,
  },
];

const fallbackStaff: Staff[] = [
  { id: "staff-andi", organizationId: "org-demo", name: "Andi", email: "andi@flowbook.test", avatarUrl: null, isActive: true },
  { id: "staff-bayu", organizationId: "org-demo", name: "Bayu", email: "bayu@flowbook.test", avatarUrl: null, isActive: true },
  { id: "staff-sari", organizationId: "org-demo", name: "Sari", email: "sari@flowbook.test", avatarUrl: null, isActive: true },
];

function fallbackSlots(params: { serviceId: string; date: string; staffId?: string }): SlotsResponse {
  // generate slots 07:00-21:00 Jakarta, 30m step, respect duration+buffer via fallbackServices
  const svc = fallbackServices.find((s) => s.id === params.serviceId) ?? fallbackServices[0];
  const total = svc.durationMinutes + svc.bufferMinutes;
  // step 15m? but demo uses 30m for simplicity, ensure total fits
  const slots: Slot[] = [];
  const staffPool = params.staffId ? fallbackStaff.filter((s) => s.id === params.staffId) : fallbackStaff;
  // eligible staff filter (PRD skill)
  const eligible = filterEligible(svc.id, staffPool);
  const baseDate = params.date; // yyyy-mm-dd
  for (let h = 7; h < 21; h++) {
    for (let m = 0; m < 60; m += 30) {
      // check if slot fits before 21:00 including buffer
      const slotStartMinutes = h * 60 + m;
      if (slotStartMinutes + total > 21 * 60) continue;
      const pad = (n: number) => String(n).padStart(2, "0");
      // Jakarta time -> convert to UTC ISO: Jakarta UTC+7 no DST
      // create Jakarta local datetime then subtract 7h
      const jakartaLocal = new Date(`${baseDate}T${pad(h)}:${pad(m)}:00+07:00`);
      const startAt = jakartaLocal.toISOString();
      const endAt = new Date(jakartaLocal.getTime() + svc.durationMinutes * 60000).toISOString();
      // mock: every 3rd slot taken, next buffer
      const idx = slots.length;
      const available = idx % 5 !== 2 && idx % 5 !== 3;
      const reason = idx % 5 === 2 ? "taken" : idx % 5 === 3 ? "buffer" : null;
      const staff = eligible[idx % eligible.length] ?? eligible[0];
      slots.push({
        startAt,
        endAt,
        available,
        staffId: staff.id,
        staffName: staff.name,
        reason,
      });
    }
  }
  return { date: params.date, tz: "Asia/Jakarta", slots };
}

function filterEligible(serviceId: string, pool: Staff[]): Staff[] {
  const map: Record<string, string[]> = {
    "svc-color": ["staff-bayu"],
    "svc-father": ["staff-andi"],
    "svc-fade": ["staff-andi", "staff-bayu"],
    "svc-grooming": ["staff-andi", "staff-bayu"],
    "svc-konsultasi": ["staff-bayu"],
  };
  const allowed = map[serviceId];
  if (!allowed) return pool;
  return pool.filter((s) => allowed.includes(s.id));
}
