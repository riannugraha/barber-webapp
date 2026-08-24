"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Calendar as CalendarIcon, Clock, Info } from "lucide-react";
import { Calendar } from "@/components/ui/calendar";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { fetchSlots, formatJakartaTime, toJakartaDateString, type Slot } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { cn } from "@/lib/utils";

type Props = {
  serviceId: string | null;
  staffId: string | null; // null = any
  selectedSlot: Slot | null;
  onSelect: (slot: Slot) => void;
};

export function CalendarStep({ serviceId, staffId, selectedSlot, onSelect }: Props) {
  const [date, setDate] = React.useState<Date>(new Date());
  const dateStr = React.useMemo(() => toJakartaDateString(date), [date]);

  const enabled = !!serviceId && !!dateStr;

  const { data, isLoading, isError, refetch, dataUpdatedAt } = useQuery({
    queryKey: queryKeys.slots({ serviceId: serviceId!, staffId: staffId ?? undefined, date: dateStr }),
    queryFn: () => fetchSlots({ serviceId: serviceId!, staffId: staffId ?? undefined, date: dateStr, tz: "Asia/Jakarta" }),
    enabled,
    refetchInterval: 30_000, // polling realtime per AC
    staleTime: 10_000,
  });

  // disable past dates
  const disabledDays = React.useMemo(() => ({ before: new Date(new Date().setHours(0, 0, 0, 0)) }), []);

  if (!serviceId) {
    return (
      <Alert>
        <AlertDescription>Pilih layanan terlebih dahulu.</AlertDescription>
      </Alert>
    );
  }

  const slots = data?.slots ?? [];
  const availableCount = slots.filter((s) => s.available).length;

  return (
    <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
      {/* Calendar pane */}
      <div className="rounded-xl border bg-card p-3">
        <div className="mb-3 flex items-center gap-2 px-2 pt-1">
          <CalendarIcon className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
          <p className="text-sm font-medium">Pilih tanggal</p>
          <span className="ml-auto text-xs text-muted-foreground">Asia/Jakarta</span>
        </div>
        <Calendar
          mode="single"
          selected={date}
          onSelect={(d) => d && setDate(d)}
          disabled={disabledDays}
          className="mx-auto"
        />
        <div className="mt-3 rounded-lg bg-muted p-3">
          <p className="text-xs font-medium">Tanggal terpilih</p>
          <p className="text-sm tabular-nums">
            {new Intl.DateTimeFormat("id-ID", { timeZone: "Asia/Jakarta", weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(date)}
          </p>
          <p className="mt-1 text-xs text-muted-foreground">
            {availableCount} slot tersedia • refresh {new Date(dataUpdatedAt).toLocaleTimeString("id-ID", { timeZone: "Asia/Jakarta" }) || "—"}
          </p>
        </div>

        {/* legend */}
        <div className="mt-4 space-y-2">
          <p className="text-xs font-medium text-muted-foreground">Legend</p>
          <div className="flex flex-wrap gap-2">
            <span className="inline-flex items-center gap-1.5 text-xs">
              <span className="h-3 w-3 rounded-sm bg-primary" aria-hidden="true" /> available
            </span>
            <span className="inline-flex items-center gap-1.5 text-xs">
              <span className="h-3 w-3 rounded-sm bg-amber-500/60" aria-hidden="true" /> buffer
            </span>
            <span className="inline-flex items-center gap-1.5 text-xs">
              <span className="h-3 w-3 rounded-sm bg-muted border" aria-hidden="true" /> taken
            </span>
          </div>
          <p className="text-[11px] leading-relaxed text-muted-foreground">
            Hanya slot yang muat <span className="font-medium text-foreground">durasi+buffer</span> yang aktif. Buffer mencegah double-booking.
          </p>
        </div>
      </div>

      {/* Slots pane */}
      <div className="rounded-xl border bg-card p-4 sm:p-6">
        <div className="flex items-center justify-between gap-2">
          <h3 className="text-sm font-semibold flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" /> Slot waktu
          </h3>
          <Badge variant="secondary" className="tabular-nums">
            {availableCount} tersedia
          </Badge>
        </div>
        <Separator className="my-4" />

        {isLoading ? (
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4" aria-busy="true" aria-label="Memuat slot">
            {Array.from({ length: 12 }).map((_, i) => (
              <Skeleton key={i} className="h-10 rounded-md" />
            ))}
          </div>
        ) : isError ? (
          <Alert variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-2">
              <span>Gagal memuat slot.</span>
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Retry
              </Button>
            </AlertDescription>
          </Alert>
        ) : slots.length === 0 ? (
          <div className="rounded-lg border border-dashed p-8 text-center">
            <p className="text-sm font-medium">Belum ada slot tersedia untuk tanggal ini</p>
            <p className="mt-1 text-xs text-muted-foreground">Coba pilih tanggal lain — slot realtime diperbarui setiap 30 detik.</p>
            <p className="mt-3 text-xs text-muted-foreground hidden">Belum ada booking hari ini — bagikan link /book</p>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-4" role="group" aria-label="Daftar slot">
              {slots.map((slot) => {
                const isSelected = selectedSlot?.startAt === slot.startAt && selectedSlot?.staffId === slot.staffId;
                const isAvailable = slot.available;
                const label = formatJakartaTime(slot.startAt);
                // buffer/taken styling
                const stateClass = !isAvailable
                  ? slot.reason === "buffer"
                    ? "bg-amber-500/20 text-amber-900 dark:text-amber-100 border-amber-500/30 cursor-not-allowed opacity-70"
                    : "bg-muted text-muted-foreground border-transparent cursor-not-allowed opacity-60"
                  : isSelected
                    ? "bg-primary text-primary-foreground border-primary shadow"
                    : "bg-background hover:bg-muted border-input hover:border-primary/30";

                return (
                  <button
                    key={`${slot.startAt}-${slot.staffId}`}
                    type="button"
                    role="button"
                    aria-label={`${label} ${slot.staffName ?? ""} ${isAvailable ? "tersedia" : slot.reason === "buffer" ? "buffer" : "taken"}`}
                    aria-pressed={isSelected}
                    aria-disabled={!isAvailable}
                    disabled={!isAvailable}
                    onClick={() => isAvailable && onSelect(slot)}
                    className={cn(
                      "relative flex flex-col items-center justify-center rounded-md border px-2 py-2.5 text-sm font-medium tabular-nums transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                      stateClass
                    )}
                  >
                    <span className="text-[13px]">{label}</span>
                    {slot.staffName ? (
                      <span className="text-[10px] font-normal opacity-80 truncate max-w-full">{slot.staffName}</span>
                    ) : null}
                    {slot.reason === "buffer" ? <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-amber-500" aria-hidden="true" /> : null}
                  </button>
                );
              })}
            </div>
            <div className="mt-4 flex gap-2 text-xs text-muted-foreground">
              <Info className="h-3.5 w-3.5 mt-0.5 shrink-0" aria-hidden="true" />
              <p className="leading-relaxed">
                Waktu ditampilkan dalam <span className="font-medium text-foreground">Asia/Jakarta</span>. Slot yang tidak muat durasi+buffer otomatis non-aktif. Data slot polling setiap 30 detik via TanStack Query.
              </p>
            </div>
            {availableCount === 0 && slots.length > 0 ? (
              <div className="mt-4 rounded-lg bg-muted p-4 text-center">
                <p className="text-sm text-muted-foreground">Semua slot pada tanggal ini sudah terisi.</p>
                <p className="text-xs text-muted-foreground mt-1">Belum ada booking yang bisa dibuat — pilih tanggal lain.</p>
              </div>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}
