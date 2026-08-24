"use client";

import * as React from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Calendar as CalendarIconLucide, Clock, UserRound, Phone, Mail, StickyNote, AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Calendar } from "@/components/ui/calendar";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { QueryProvider } from "@/components/providers";
import { getBooking, cancelBooking, rescheduleBooking, fetchSlots, formatJakartaDate, formatJakartaTime, toJakartaDateString, type Slot } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

function TrackInner() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const qc = useQueryClient();

  const [cancelOpen, setCancelOpen] = React.useState(false);
  const [rescheduleOpen, setRescheduleOpen] = React.useState(false);
  const [rescheduleDate, setRescheduleDate] = React.useState<Date>(new Date());
  const [selectedSlot, setSelectedSlot] = React.useState<Slot | null>(null);

  const { data: booking, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.booking(id),
    queryFn: () => getBooking(id),
    enabled: !!id,
  });

  const dateStr = toJakartaDateString(rescheduleDate);

  const { data: slotsData, isLoading: slotsLoading } = useQuery({
    queryKey: booking ? queryKeys.slots({ serviceId: booking.serviceId, staffId: booking.staffId, date: dateStr }) : ["slots", "none"],
    queryFn: () => fetchSlots({ serviceId: booking!.serviceId, staffId: booking!.staffId, date: dateStr, tz: "Asia/Jakarta" }),
    enabled: !!booking && rescheduleOpen,
    refetchInterval: 30_000,
  });

  const cancelMut = useMutation({
    mutationFn: () => cancelBooking(id),
    onSuccess: () => {
      toast.success("Booking dibatalkan");
      qc.invalidateQueries({ queryKey: queryKeys.booking(id) });
      setCancelOpen(false);
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal cancel"),
  });

  const rescheduleMut = useMutation({
    mutationFn: () => {
      if (!selectedSlot || !booking) throw new Error("Pilih slot baru");
      return rescheduleBooking(id, { staffId: selectedSlot.staffId!, startAt: selectedSlot.startAt });
    },
    onSuccess: () => {
      toast.success("Jadwal dipindahkan");
      qc.invalidateQueries({ queryKey: queryKeys.booking(id) });
      setRescheduleOpen(false);
      setSelectedSlot(null);
    },
    onError: (e: unknown) => toast.error(e instanceof Error ? e.message : "Gagal reschedule — slot mungkin sudah diambil (409)"),
  });

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (isError || !booking) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10">
        <Alert variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>Belum ada booking ditemukan untuk ID ini.</span>
            <Button variant="outline" size="sm" onClick={() => refetch()}>Retry</Button>
          </AlertDescription>
        </Alert>
        <div className="mt-4 text-center">
          <Link href="/book" className="text-sm underline">Buat booking baru</Link>
        </div>
      </div>
    );
  }

  const statusVariant = booking.status === "CONFIRMED" ? "default" : booking.status === "CANCELLED" ? "destructive" : "secondary";

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-card">
        <div className="mx-auto max-w-2xl px-4 py-4 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-primary text-primary-foreground grid place-items-center text-sm font-semibold">FB</div>
            <span className="text-sm font-semibold">FlowBarber Studio</span>
          </Link>
          <Link href="/book" className="text-xs text-muted-foreground hover:text-foreground underline">Book lagi</Link>
        </div>
      </header>

      <main className="mx-auto max-w-2xl px-4 py-6 sm:py-8 space-y-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">Detail booking</h1>
            <p className="text-sm text-muted-foreground font-mono tabular-nums">ID {booking.id}</p>
          </div>
          <Badge variant={statusVariant} className="shrink-0">{booking.status}</Badge>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <CalendarIconLucide className="h-4 w-4 text-muted-foreground" /> Jadwal
            </CardTitle>
            <CardDescription>Timezone Asia/Jakarta • {booking.paymentStatus ?? "UNPAID"}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <dl className="grid gap-3 text-sm">
              <div className="flex gap-3">
                <Clock className="h-4 w-4 text-muted-foreground mt-0.5" />
                <div>
                  <dt className="text-muted-foreground text-xs">Waktu</dt>
                  <dd className="font-medium tabular-nums">{formatJakartaDate(booking.startAt)} • {formatJakartaTime(booking.startAt)} – {formatJakartaTime(booking.endAt)} WIB</dd>
                </div>
              </div>
              <div className="flex gap-3">
                <UserRound className="h-4 w-4 text-muted-foreground mt-0.5" />
                <div>
                  <dt className="text-muted-foreground text-xs">Staff & Layanan</dt>
                  <dd className="font-medium">{booking.staffId.slice(0, 8)} • {booking.serviceId.slice(0, 8)}</dd>
                  <dd className="text-xs text-muted-foreground">Service {booking.serviceId} — Staff {booking.staffId}</dd>
                </div>
              </div>
              <div className="flex gap-3">
                <Mail className="h-4 w-4 text-muted-foreground mt-0.5" />
                <div>
                  <dt className="text-muted-foreground text-xs">Customer</dt>
                  <dd className="font-medium">{booking.customerName}</dd>
                  <dd className="text-xs text-muted-foreground">{booking.customerEmail} {booking.customerPhone ? `• ${booking.customerPhone}` : ""}</dd>
                </div>
              </div>
              {booking.notes ? (
                <div className="flex gap-3">
                  <StickyNote className="h-4 w-4 text-muted-foreground mt-0.5" />
                  <div>
                    <dt className="text-muted-foreground text-xs">Catatan</dt>
                    <dd className="text-sm">{booking.notes}</dd>
                  </div>
                </div>
              ) : null}
              {booking.customerPhone ? (
                <div className="flex gap-3">
                  <Phone className="h-4 w-4 text-muted-foreground mt-0.5" />
                  <div>
                    <dt className="text-muted-foreground text-xs">Telepon</dt>
                    <dd className="font-medium tabular-nums">{booking.customerPhone}</dd>
                  </div>
                </div>
              ) : null}
            </dl>

            <Separator />

            <div className="grid grid-cols-2 gap-3">
              <Button variant="outline" onClick={() => setRescheduleOpen(true)} disabled={booking.status === "CANCELLED"} aria-label="Reschedule booking">
                Reschedule
              </Button>
              <Button variant="destructive" onClick={() => setCancelOpen(true)} disabled={booking.status === "CANCELLED"} aria-label="Cancel booking">
                Cancel
              </Button>
            </div>
            {booking.status === "CANCELLED" ? (
              <Alert>
                <AlertDescription className="flex gap-2 text-sm"><AlertTriangle className="h-4 w-4" /> Booking ini sudah dibatalkan.</AlertDescription>
              </Alert>
            ) : null}
          </CardContent>
        </Card>

        {/* Reschedule Dialog */}
        <Dialog open={rescheduleOpen} onOpenChange={setRescheduleOpen}>
          <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Reschedule booking</DialogTitle>
              <DialogDescription>Pilih tanggal baru via Calendar, lalu pilih slot yang tersedia (Asia/Jakarta). Hanya slot muat durasi+buffer yang aktif.</DialogDescription>
            </DialogHeader>

            <div className="grid gap-6 sm:grid-cols-[300px_1fr]">
              <div className="rounded-xl border p-3">
                <Calendar mode="single" selected={rescheduleDate} onSelect={(d) => d && setRescheduleDate(d)} disabled={{ before: new Date(new Date().setHours(0, 0, 0, 0)) }} />
                <p className="mt-2 text-xs text-muted-foreground text-center tabular-nums">{dateStr} • Asia/Jakarta</p>
              </div>
              <div className="space-y-3">
                <p className="text-sm font-medium">Slot tersedia</p>
                {slotsLoading ? (
                  <div className="grid grid-cols-3 gap-2">
                    {Array.from({ length: 9 }).map((_, i) => (
                      <Skeleton key={i} className="h-10" />
                    ))}
                  </div>
                ) : !slotsData || slotsData.slots.length === 0 ? (
                  <div className="rounded-lg border border-dashed p-6 text-center">
                    <p className="text-sm text-muted-foreground">Belum ada slot tersedia untuk tanggal ini</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-3 gap-2 max-h-[260px] overflow-y-auto pr-1" role="group" aria-label="Pilih slot baru">
                    {slotsData.slots.map((slot) => {
                      const available = slot.available;
                      const selected = selectedSlot?.startAt === slot.startAt;
                      return (
                        <button
                          key={slot.startAt}
                          type="button"
                          disabled={!available}
                          aria-pressed={selected}
                          aria-label={`${formatJakartaTime(slot.startAt)} ${available ? "tersedia" : slot.reason}`}
                          onClick={() => available && setSelectedSlot(slot)}
                          className={[
                            "rounded-md border px-2 py-2 text-sm tabular-nums transition-colors",
                            !available ? "bg-muted text-muted-foreground opacity-60 cursor-not-allowed" : selected ? "bg-primary text-primary-foreground border-primary" : "bg-background hover:bg-muted",
                          ].join(" ")}
                        >
                          {formatJakartaTime(slot.startAt)}
                        </button>
                      );
                    })}
                  </div>
                )}
                <div className="flex gap-2 text-xs text-muted-foreground">
                  <span className="h-3 w-3 rounded-sm bg-primary shrink-0 mt-0.5" /> available
                  <span className="h-3 w-3 rounded-sm bg-amber-500/60 shrink-0 mt-0.5" /> buffer
                  <span className="h-3 w-3 rounded-sm bg-muted border shrink-0 mt-0.5" /> taken
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setRescheduleOpen(false)}>Batal</Button>
              <Button onClick={() => rescheduleMut.mutate()} disabled={!selectedSlot || rescheduleMut.isPending} className="gap-2">
                {rescheduleMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                Konfirmasi reschedule
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Cancel Dialog */}
        <Dialog open={cancelOpen} onOpenChange={setCancelOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Batalkan booking?</DialogTitle>
              <DialogDescription>Tindakan ini tidak dapat dibatalkan. Slot akan kembali tersedia untuk orang lain.</DialogDescription>
            </DialogHeader>
            <div className="rounded-lg bg-destructive/10 border border-destructive/20 p-3 flex gap-2 text-sm">
              <AlertTriangle className="h-4 w-4 text-destructive mt-0.5" />
              <span>Booking <span className="font-mono tabular-nums">{booking.id.slice(0, 8)}…</span> akan dibatalkan.</span>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCancelOpen(false)}>Kembali</Button>
              <Button variant="destructive" onClick={() => cancelMut.mutate()} disabled={cancelMut.isPending} className="gap-2">
                {cancelMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                Ya, batalkan
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <div className="text-center">
          <Link href="/book" className="text-sm underline underline-offset-4">Buat booking baru</Link>
        </div>
      </main>
    </div>
  );
}

export default function TrackPage() {
  return (
    <QueryProvider>
      <TrackInner />
    </QueryProvider>
  );
}
