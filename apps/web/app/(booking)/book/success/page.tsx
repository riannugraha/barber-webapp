"use client";

import * as React from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, CalendarDays, Download, ArrowRight, Clock, UserRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { QueryProvider } from "@/components/providers";
import { getBooking, formatJakartaDate, formatJakartaTime, formatPrice } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { downloadICS } from "@/lib/ics";

function SuccessInner() {
  const params = useSearchParams();
  const id = params.get("id");

  const { data: booking, isLoading, isError } = useQuery({
    queryKey: id ? queryKeys.booking(id) : ["booking", "none"],
    queryFn: () => getBooking(id!),
    enabled: !!id,
  });

  const handleDownload = () => {
    if (!booking) return;
    // fetch service detail if needed — for demo just use booking
    downloadICS(booking);
  };

  if (!id) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10">
        <Alert>
          <AlertDescription>ID booking tidak ditemukan. Kembali ke <Link href="/book" className="underline">booking</Link>.</AlertDescription>
        </Alert>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10 space-y-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (isError || !booking) {
    return (
      <div className="mx-auto max-w-2xl px-4 py-10">
        <Alert variant="destructive">
          <AlertDescription>Gagal memuat booking. Cek link atau <Link href={`/track/${id}`} className="underline">lacak booking</Link>.</AlertDescription>
        </Alert>
      </div>
    );
  }

  const isConfirmed = booking.status === "CONFIRMED";

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b bg-card">
        <div className="mx-auto max-w-2xl px-4 py-4 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-primary text-primary-foreground grid place-items-center text-sm font-semibold">FB</div>
            <span className="text-sm font-semibold">FlowBarber Studio</span>
          </Link>
          <Badge variant={isConfirmed ? "default" : "secondary"}>{booking.status}</Badge>
        </div>
      </header>

      <main className="mx-auto max-w-2xl px-4 py-6 sm:py-10 space-y-6">
        <Card className="overflow-hidden border-primary/20">
          <div className="bg-primary p-[1px]" aria-hidden="true" />
          <CardHeader className="text-center pb-4">
            <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
              <CheckCircle2 className="h-7 w-7" aria-hidden="true" />
            </div>
            <CardTitle className="text-xl mt-3">{isConfirmed ? "Booking dikonfirmasi!" : "Booking dibuat"}</CardTitle>
            <CardDescription className="text-sm">
              {isConfirmed
                ? "Kami telah kirim email konfirmasi + .ics. Sampai jumpa di studio!"
                : "Menunggu pembayaran — cek email untuk link Stripe."}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="rounded-lg bg-muted p-4 space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium">
                <CalendarDays className="h-4 w-4 text-muted-foreground" /> Detail booking
              </div>
              <Separator />
              <dl className="grid gap-2 text-sm">
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground">ID</dt>
                  <dd className="font-mono text-xs tabular-nums">{booking.id.slice(0, 8)}…</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground flex items-center gap-1"><UserRound className="h-3 w-3" /> Customer</dt>
                  <dd className="font-medium">{booking.customerName} • {booking.customerEmail}</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground flex items-center gap-1"><Clock className="h-3 w-3" /> Waktu</dt>
                  <dd className="tabular-nums font-medium">
                    {formatJakartaDate(booking.startAt)} • {formatJakartaTime(booking.startAt)} – {formatJakartaTime(booking.endAt)} WIB
                  </dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="text-muted-foreground">Status bayar</dt>
                  <dd>
                    <Badge variant={booking.paymentStatus === "PAID" ? "default" : "secondary"} className="tabular-nums">{booking.paymentStatus ?? "UNPAID"}</Badge>
                  </dd>
                </div>
              </dl>
            </div>

            <div className="grid gap-3 sm:grid-cols-2">
              <Button onClick={handleDownload} variant="outline" className="gap-2" aria-label="Download file kalender .ics">
                <Download className="h-4 w-4" /> Download .ics
              </Button>
              <Button asChild className="gap-2">
                <Link href={`/track/${booking.id}`} aria-label="Lihat detail dan kelola booking">
                  Kelola booking <ArrowRight className="h-4 w-4" />
                </Link>
              </Button>
            </div>

            <div className="rounded-lg border border-dashed p-4 text-center">
              <p className="text-sm text-muted-foreground">Email konfirmasi telah dikirim ke <span className="font-medium text-foreground">{booking.customerEmail}</span> via Resend (log).</p>
              <p className="mt-1 text-xs text-muted-foreground">Tambahkan ke kalender Google/Apple via .ics di atas.</p>
            </div>

            <div className="flex gap-3">
              <Button asChild variant="ghost" className="flex-1">
                <Link href="/book">Booking lagi</Link>
              </Button>
              <Button asChild variant="secondary" className="flex-1">
                <Link href="/">Ke beranda</Link>
              </Button>
            </div>
          </CardContent>
        </Card>

        <p className="text-center text-xs text-muted-foreground">Butuh ubah jadwal? Buka <Link href={`/track/${booking.id}`} className="underline">/track/{booking.id.slice(0, 8)}…</Link> untuk reschedule / cancel.</p>
      </main>
    </div>
  );
}

export default function SuccessPage() {
  return (
    <QueryProvider>
      <React.Suspense fallback={<div className="p-10"><Skeleton className="h-40 w-full" /></div>}>
        <SuccessInner />
      </React.Suspense>
    </QueryProvider>
  );
}
