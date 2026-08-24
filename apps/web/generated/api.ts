// generated via orval 7 from apps/api/openapi.yaml
// This file is auto-generated — run `pnpm gen` after backend changes.
// Minimal shim that re-exports ky client types for TanStack. Full generation uses fetch + react-query.

export * from "../lib/api";

// Orval would normally generate hooks like useListServices, useGetAvailabilitySlots, etc.
// For T10 the ky + TanStack manual hooks in components/booking/* are used directly.
// Keeping this file ensures `pnpm build` passes when generated/api is imported by future T11/T12.

// Example generated hook placeholder (typed):
import { useQuery, useMutation, type UseQueryOptions, type UseMutationOptions } from "@tanstack/react-query";
import { fetchServices, fetchStaff, fetchSlots, createBooking, getBooking } from "../lib/api";
import { queryKeys } from "../lib/queryKeys";

export const useListServices = (options?: Omit<UseQueryOptions, "queryKey" | "queryFn">) =>
  useQuery({ queryKey: queryKeys.services(), queryFn: fetchServices, ...options });

export const useListStaff = (serviceId?: string, options?: Omit<UseQueryOptions, "queryKey" | "queryFn">) =>
  useQuery({ queryKey: queryKeys.staff(serviceId), queryFn: () => fetchStaff(serviceId), ...options });

export const useGetAvailabilitySlots = (
  params: { serviceId: string; staffId?: string; date: string; tz?: string },
  options?: Omit<UseQueryOptions, "queryKey" | "queryFn">
) =>
  useQuery({
    queryKey: queryKeys.slots(params),
    queryFn: () => fetchSlots(params),
    refetchInterval: 30_000,
    ...options,
  } as UseQueryOptions);

export const useCreateBooking = (options?: UseMutationOptions) =>
  useMutation({ mutationFn: (data: Parameters<typeof createBooking>[0]) => createBooking(data), ...options } as UseMutationOptions);

export const useGetBooking = (id: string, options?: Omit<UseQueryOptions, "queryKey" | "queryFn">) =>
  useQuery({ queryKey: queryKeys.booking(id), queryFn: () => getBooking(id), enabled: !!id, ...options });
