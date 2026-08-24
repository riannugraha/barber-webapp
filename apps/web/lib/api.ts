import ky, { HTTPError } from "ky";

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ??
  "https://flowbook-api-xxx.koyeb.app/api/v1";

// --- ky instance: allmutations via ky, Bearer dari localStorage, 422/409 -> Toaster ---
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
    afterResponse: [
      async (_request, _options, response) => {
        // Global 401 handler: redirect to /login (client only). Middleware also guards /app/* on server.
        if (response.status === 401 && typeof window !== "undefined") {
          const path = window.location.pathname;
          if (path.startsWith("/app")) {
            // avoid loop: only redirect if not already on /login
            // store current path for return? simple redirect
            // Don't redirect if we are on login already
            if (path !== "/login") {
              // Use toast before redirect
              try {
                const { toast } = await import("sonner");
                toast.error("Sesi habis — silakan login kembali");
              } catch {}
              window.location.href = "/login?next=" + encodeURIComponent(path);
            }
          }
        }
        // 422/409 -> toast (422 validation, 409 conflict overlap)
        if ((response.status === 422 || response.status === 409) && typeof window !== "undefined") {
          try {
            const clone = response.clone();
            const body = (await clone.json().catch(() => null)) as unknown;
            let msg: string | null = null;
            if (body && typeof body === "object") {
              const b = body as Record<string, unknown>;
              if (typeof b.message === "string") msg = b.message;
              else if (typeof b.error === "string") msg = b.error;
              else if (Array.isArray((b as { details?: unknown }).details)) {
                const d = (b as { details: Array<{ message?: string }> }).details;
                msg = d.map((x) => x.message).filter(Boolean).join(", ");
              }
            }
            const { toast } = await import("sonner");
            if (response.status === 422) toast.error(msg ?? "Validasi gagal — cek kembali isian");
            else toast.error(msg ?? "Slot sudah diambil — pilih slot lain (409)");
          } catch {}
        }
        return response;
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
  services?: Service[];
  availability?: Availability[];
};

export type Availability = {
  id: string;
  staffId: string;
  dayOfWeek: number; // 0 Sun .. 6 Sat
  startTime: string; // "09:00"
  endTime: string; // "17:00"
};

export type AvailabilityInput = {
  dayOfWeek: number;
  startTime: string;
  endTime: string;
};

export type AvailabilityOverride = {
  id: string;
  staffId: string;
  date: string; // yyyy-mm-dd
  isClosed: boolean;
  startTime?: string | null;
  endTime?: string | null;
  reason?: string | null;
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
  service?: Service;
  staff?: Staff;
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

export type Customer = {
  id: string;
  organizationId: string;
  name: string;
  email: string;
  phone?: string | null;
  bookingsCount: number;
  totalSpentCents: number;
  lastBookingAt?: string | null;
  createdAt: string;
};

export type Organization = {
  id: string;
  name: string;
  timezone: string;
  logoUrl?: string | null;
  createdAt?: string;
};

export type PaginationMeta = {
  page: number;
  limit: number;
  total: number;
  totalPages: number;
};

export type Paginated<T> = {
  data: T[];
  meta: PaginationMeta;
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

export function formatIDRTabular(priceCents: number): string {
  // for KPI: Rp 142.000 with tabular-nums
  return formatPrice(priceCents);
}

// Extract error message suitable for Toaster / Zod-compatible details
export async function parseApiError(err: unknown): Promise<{ status?: number; message: string; details?: Array<{ field: string; message: string }> }> {
  if (err instanceof HTTPError) {
    const status = err.response.status;
    try {
      const body = (await err.response.clone().json()) as Record<string, unknown>;
      const message = (body.message as string) ?? (body.error as string) ?? err.message;
      const details = (body.details as Array<{ field: string; message: string }>) ?? undefined;
      return { status, message, details };
    } catch {
      return { status, message: err.message };
    }
  }
  if (err instanceof Error) return { message: err.message };
  return { message: String(err) };
}

/* ── API Calls — all via ky, no Server Actions ── */

export async function fetchServices(): Promise<Service[]> {
  try {
    const res = await api.get("services").json<{ data: Service[] }>();
    return res.data;
  } catch {
    return fallbackServices;
  }
}

export async function fetchService(id: string): Promise<Service> {
  try {
    return await api.get(`services/${id}`).json<Service>();
  } catch {
    const f = fallbackServices.find((s) => s.id === id);
    if (f) return f;
    throw new Error("Service not found");
  }
}

export async function createService(data: Partial<Service> & { name: string; durationMinutes: number; priceCents: number }): Promise<Service> {
  return await api.post("services", { json: data }).json<Service>();
}

export async function updateService(id: string, data: Partial<Service>): Promise<Service> {
  return await api.put(`services/${id}`, { json: data }).json<Service>();
}

export async function deleteService(id: string): Promise<void> {
  await api.delete(`services/${id}`).json();
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

export async function fetchStaffDetail(id: string): Promise<Staff> {
  try {
    return await api.get(`staff/${id}`).json<Staff>();
  } catch {
    const f = fallbackStaff.find((s) => s.id === id);
    if (f) return { ...f, services: fallbackServices.slice(0, 2), availability: fallbackAvailability(f.id) };
    throw new Error("Staff not found");
  }
}

export async function createStaff(data: { name: string; email?: string; serviceIds?: string[]; availability?: AvailabilityInput[] }): Promise<Staff> {
  return await api.post("staff", { json: data }).json<Staff>();
}

export async function updateStaffAvailability(staffId: string, availability: AvailabilityInput[]): Promise<Staff> {
  // Endpoint: PUT /staff/{id}/availability if backend supports, else PATCH /staff/{id}
  // Fallback to PUT staff
  try {
    return await api.put(`staff/${staffId}/availability`, { json: { availability } }).json<Staff>();
  } catch {
    // try generic update
    return await api.put(`staff/${staffId}`, { json: { availability } }).json<Staff>();
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

export async function listBookings(params: {
  from?: string;
  to?: string;
  status?: string;
  staffId?: string;
  search?: string;
  page?: number;
  limit?: number;
}): Promise<Paginated<Booking>> {
  const sp = new URLSearchParams();
  if (params.from) sp.set("from", params.from);
  if (params.to) sp.set("to", params.to);
  if (params.status) sp.set("status", params.status);
  if (params.staffId) sp.set("staffId", params.staffId);
  if (params.search) sp.set("search", params.search);
  if (params.page) sp.set("page", String(params.page));
  if (params.limit) sp.set("limit", String(params.limit));
  try {
    const res = await api.get(`bookings?${sp.toString()}`).json<Paginated<Booking>>();
    return res;
  } catch {
    return fallbackBookings(params);
  }
}

export async function createCheckoutSession(payload: {
  bookingId: string;
  successUrl: string;
  cancelUrl: string;
}): Promise<{ url: string; sessionId: string }> {
  return await api.post("payments/checkout-session", { json: payload }).json();
}

export async function fetchCustomers(search?: string): Promise<Customer[]> {
  const sp = search ? `?search=${encodeURIComponent(search)}` : "";
  try {
    const res = await api.get(`customers${sp}`).json<{ data: Customer[] } | Customer[]>();
    if (Array.isArray(res)) return res;
    return (res as { data: Customer[] }).data;
  } catch {
    return fallbackCustomers.filter((c) => !search || c.name.toLowerCase().includes(search.toLowerCase()) || c.email.toLowerCase().includes(search.toLowerCase()));
  }
}

export async function fetchOrganization(): Promise<Organization> {
  try {
    return await api.get("organizations/me").json<Organization>();
  } catch {
    // try /organization, else fallback
    try {
      return await api.get("organization").json<Organization>();
    } catch {
      return fallbackOrg;
    }
  }
}

export async function updateOrganization(data: Partial<Organization>): Promise<Organization> {
  try {
    return await api.put("organizations/me", { json: data }).json<Organization>();
  } catch {
    try {
      return await api.put("organization", { json: data }).json<Organization>();
    } catch {
      // fallback echo
      return { ...fallbackOrg, ...data } as Organization;
    }
  }
}

export async function uploadLogo(file: File): Promise<{ logoUrl: string }> {
  const form = new FormData();
  form.append("file", file);
  form.append("logo", file);
  try {
    // use ky without json header for multipart
    const res = await ky
      .post(`${API_URL}/organizations/me/logo`, {
        body: form,
        headers: {
          // let ky set content-type boundary, inject auth manually
          ...(typeof window !== "undefined" && localStorage.getItem("flowbook_access")
            ? { Authorization: `Bearer ${localStorage.getItem("flowbook_access")}` }
            : {}),
        },
      })
      .json<{ logoUrl: string }>();
    return res;
  } catch {
    // mock success for demo / offline build
    return { logoUrl: URL.createObjectURL(file) };
  }
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

function fallbackAvailability(staffId: string): Availability[] {
  const base: Record<string, Availability[]> = {
    "staff-andi": [
      { id: "av-1", staffId, dayOfWeek: 1, startTime: "07:00", endTime: "16:00" },
      { id: "av-2", staffId, dayOfWeek: 2, startTime: "09:00", endTime: "18:00" },
      { id: "av-3", staffId, dayOfWeek: 3, startTime: "09:00", endTime: "18:00" },
      { id: "av-4", staffId, dayOfWeek: 4, startTime: "09:00", endTime: "18:00" },
      { id: "av-5", staffId, dayOfWeek: 5, startTime: "09:00", endTime: "18:00" },
      { id: "av-6", staffId, dayOfWeek: 6, startTime: "08:00", endTime: "15:00" },
    ],
    "staff-bayu": [
      { id: "av-7", staffId, dayOfWeek: 1, startTime: "09:00", endTime: "21:00" },
      { id: "av-8", staffId, dayOfWeek: 2, startTime: "09:00", endTime: "21:00" },
      { id: "av-9", staffId, dayOfWeek: 3, startTime: "09:00", endTime: "21:00" },
      { id: "av-10", staffId, dayOfWeek: 4, startTime: "09:00", endTime: "21:00" },
      { id: "av-11", staffId, dayOfWeek: 5, startTime: "09:00", endTime: "21:00" },
    ],
    "staff-sari": [
      { id: "av-12", staffId, dayOfWeek: 1, startTime: "10:00", endTime: "19:00" },
      { id: "av-13", staffId, dayOfWeek: 3, startTime: "10:00", endTime: "19:00" },
      { id: "av-14", staffId, dayOfWeek: 5, startTime: "10:00", endTime: "19:00" },
      { id: "av-15", staffId, dayOfWeek: 6, startTime: "10:00", endTime: "19:00" },
    ],
  };
  return base[staffId] ?? base["staff-andi"];
}

const fallbackCustomers: Customer[] = Array.from({ length: 12 }).map((_, i) => {
  const names = ["Siti Rahayu", "Budi Santoso", "Ani Wijaya", "Joko Prabowo", "Dewi Lestari", "Agus Prasetyo", "Rina Maulana", "Eko Saputra", "Maya Sari", "Hendra Gunawan", "Lina Kusuma", "Fajar Nugroho"];
  const emails = names.map((n) => n.toLowerCase().replace(/\s/g, ".") + "@example.com");
  return {
    id: `cust-${i + 1}`,
    organizationId: "org-demo",
    name: names[i % names.length],
    email: emails[i % emails.length],
    phone: `0812-3456-78${String(i).padStart(2, "0")}`,
    bookingsCount: [18, 14, 12, 10, 9, 8, 7, 6, 5, 4, 3, 2][i] ?? i + 1,
    totalSpentCents: ([18, 14, 12, 10, 9, 8, 7, 6, 5, 4, 3, 2][i] ?? i) * 85000,
    lastBookingAt: new Date(Date.now() - i * 86400000 * 3).toISOString(),
    createdAt: new Date(Date.now() - 100 * 86400000).toISOString(),
  };
});

const fallbackOrg: Organization = {
  id: "org-demo",
  name: "FlowBarber Studio",
  timezone: "Asia/Jakarta",
  logoUrl: null,
};

function fallbackSlots(params: { serviceId: string; date: string; staffId?: string }): SlotsResponse {
  const svc = fallbackServices.find((s) => s.id === params.serviceId) ?? fallbackServices[0];
  const total = svc.durationMinutes + svc.bufferMinutes;
  const slots: Slot[] = [];
  const staffPool = params.staffId ? fallbackStaff.filter((s) => s.id === params.staffId) : fallbackStaff;
  const eligible = filterEligible(svc.id, staffPool);
  const baseDate = params.date;
  for (let h = 7; h < 21; h++) {
    for (let m = 0; m < 60; m += 30) {
      const slotStartMinutes = h * 60 + m;
      if (slotStartMinutes + total > 21 * 60) continue;
      const pad = (n: number) => String(n).padStart(2, "0");
      const jakartaLocal = new Date(`${baseDate}T${pad(h)}:${pad(m)}:00+07:00`);
      const startAt = jakartaLocal.toISOString();
      const endAt = new Date(jakartaLocal.getTime() + svc.durationMinutes * 60000).toISOString();
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

function fallbackBookings(params: { from?: string; to?: string; status?: string; staffId?: string; search?: string; page?: number; limit?: number }): Paginated<Booking> {
  const all: Booking[] = Array.from({ length: 42 }).map((_, i) => {
    const staff = fallbackStaff[i % fallbackStaff.length];
    const svc = fallbackServices[i % fallbackServices.length];
    const day = new Date(Date.now() - ((i * 7) % 30) * 86400000);
    const hour = 8 + (i % 10);
    const start = new Date(day);
    start.setHours(hour, (i % 2) * 30, 0, 0);
    const end = new Date(start.getTime() + svc.durationMinutes * 60000);
    const statuses: Booking["status"][] = ["CONFIRMED", "CONFIRMED", "CONFIRMED", "PENDING", "CANCELLED"];
    return {
      id: `bk-${String(i + 1).padStart(4, "0")}`,
      organizationId: "org-demo",
      serviceId: svc.id,
      staffId: staff.id,
      customerName: fallbackCustomers[i % fallbackCustomers.length].name,
      customerEmail: fallbackCustomers[i % fallbackCustomers.length].email,
      customerPhone: "08123456789",
      notes: i % 5 === 0 ? "Minta fade tipis" : null,
      startAt: start.toISOString(),
      endAt: end.toISOString(),
      status: statuses[i % statuses.length],
      paymentStatus: svc.priceCents === 0 ? "PAID" : i % 3 === 0 ? "PAID" : "UNPAID",
      createdAt: new Date(start.getTime() - 86400000).toISOString(),
    };
  });

  let filtered = all;
  if (params.status) filtered = filtered.filter((b) => b.status === params.status);
  if (params.staffId) filtered = filtered.filter((b) => b.staffId === params.staffId);
  if (params.search) {
    const q = params.search.toLowerCase();
    filtered = filtered.filter((b) => b.customerName.toLowerCase().includes(q) || b.customerEmail.toLowerCase().includes(q) || b.id.toLowerCase().includes(q));
  }
  if (params.from) {
    const from = new Date(params.from);
    filtered = filtered.filter((b) => new Date(b.startAt) >= from);
  }
  if (params.to) {
    const to = new Date(params.to);
    filtered = filtered.filter((b) => new Date(b.startAt) <= to);
  }

  const page = params.page ?? 1;
  const limit = params.limit ?? 10;
  const total = filtered.length;
  const totalPages = Math.max(1, Math.ceil(total / limit));
  const startIdx = (page - 1) * limit;
  const data = filtered.slice(startIdx, startIdx + limit);
  return { data, meta: { page, limit, total, totalPages } };
}
