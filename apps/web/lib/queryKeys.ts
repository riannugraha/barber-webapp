export const queryKeys = {
  services: () => ["services"] as const,
  staff: (serviceId?: string) => ["staff", serviceId ?? "all"] as const,
  slots: (params: { serviceId: string; staffId?: string; date: string; tz?: string }) =>
    ["slots", params.serviceId, params.staffId ?? "any", params.date, params.tz ?? "Asia/Jakarta"] as const,
  booking: (id: string) => ["booking", id] as const,
  bookings: (filters?: Record<string, unknown>) => ["bookings", filters] as const,
} as const;
