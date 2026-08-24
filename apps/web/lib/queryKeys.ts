export const queryKeys = {
  services: () => ["services"] as const,
  service: (id: string) => ["services", id] as const,
  staff: (serviceId?: string) => ["staff", serviceId ?? "all"] as const,
  staffDetail: (id: string) => ["staff", id] as const,
  slots: (params: { serviceId: string; staffId?: string; date: string; tz?: string }) =>
    ["slots", params.serviceId, params.staffId ?? "any", params.date, params.tz ?? "Asia/Jakarta"] as const,
  booking: (id: string) => ["booking", id] as const,
  bookings: (filters?: Record<string, unknown>) => ["bookings", filters] as const,
  customers: (search?: string) => ["customers", search ?? ""] as const,
  organization: () => ["organization"] as const,
  dashboard: (params?: Record<string, unknown>) => ["dashboard", params] as const,
} as const;
