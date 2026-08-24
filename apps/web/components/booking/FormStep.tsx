"use client";

import * as React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation } from "@tanstack/react-query";
import { Loader2, Check, CreditCard } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage, FormDescription } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { createBooking, createCheckoutSession, formatJakartaDate, formatJakartaTime, formatPrice, type Service, type Slot, type Staff } from "@/lib/api";
import { toast } from "sonner";

const schema = z.object({
  customerName: z.string().min(2, "Nama minimal 2 karakter").max(60),
  customerEmail: z.string().email("Email tidak valid").max(120),
  customerPhone: z.string().max(20).optional().or(z.literal("")),
  notes: z.string().max(500, "Maks 500 karakter").optional().or(z.literal("")),
});

type FormValues = z.infer<typeof schema>;

type Props = {
  service: Service | null;
  staff: Staff | null; // null = any
  slot: Slot | null;
  onSuccess: (bookingId: string) => void;
  onBack: () => void;
};

export function FormStep({ service, staff, slot, onSuccess, onBack }: Props) {
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { customerName: "", customerEmail: "", customerPhone: "", notes: "" },
  });

  const isFree = (service?.priceCents ?? 0) === 0;

  const bookingMutation = useMutation({
    mutationFn: async (values: FormValues) => {
      if (!service || !slot) throw new Error("Layanan/slot belum dipilih");
      // if Any available, use slot's staffId
      const staffId = staff?.id ?? slot.staffId;
      if (!staffId) throw new Error("Staff belum dipilih");
      return await createBooking({
        serviceId: service.id,
        staffId,
        startAt: slot.startAt,
        customerName: values.customerName,
        customerEmail: values.customerEmail,
        customerPhone: values.customerPhone || undefined,
        notes: values.notes || undefined,
      });
    },
    onSuccess: async (booking) => {
      toast.success(booking.status === "CONFIRMED" ? "Booking berhasil dikonfirmasi!" : "Booking dibuat, lanjut pembayaran");
      if (isFree || booking.paymentStatus === "PAID" || booking.status === "CONFIRMED") {
        onSuccess(booking.id);
        return;
      }
      // paid flow: create checkout session
      try {
        const origin = typeof window !== "undefined" ? window.location.origin : "";
        const { url } = await createCheckoutSession({
          bookingId: booking.id,
          successUrl: `${origin}/book/success?id=${booking.id}`,
          cancelUrl: `${origin}/book?cancelled=${booking.id}`,
        });
        window.location.href = url;
      } catch (e) {
        // fallback: go to success even if stripe fails (demo)
        console.error(e);
        toast.error("Gagal membuat sesi pembayaran — coba lagi atau hubungi admin");
        onSuccess(booking.id);
      }
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : "Gagal membuat booking";
      // handle 409 conflict
      if (String(msg).includes("409") || String(msg).toLowerCase().includes("conflict") || String(msg).toLowerCase().includes("overlap")) {
        toast.error("Slot sudah diambil — pilih slot lain");
      } else {
        toast.error(msg);
      }
    },
  });

  const onSubmit = (values: FormValues) => bookingMutation.mutate(values);

  if (!service || !slot) {
    return (
      <Alert>
        <AlertDescription>Lengkapi langkah sebelumnya — pilih layanan, staff, dan slot.</AlertDescription>
      </Alert>
    );
  }

  const staffName = staff?.name ?? slot.staffName ?? "Any available";

  return (
    <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      {/* Form */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Detail kontak</CardTitle>
          <p className="text-sm text-muted-foreground">Kami akan kirim konfirmasi ke email ini.</p>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5" noValidate>
              <FormField
                control={form.control}
                name="customerName"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel htmlFor="customerName">Nama lengkap</FormLabel>
                    <FormControl>
                      <Input id="customerName" placeholder="Budi Santoso" autoComplete="name" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="customerEmail"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel htmlFor="customerEmail">Email</FormLabel>
                    <FormControl>
                      <Input id="customerEmail" type="email" placeholder="budi@email.com" autoComplete="email" {...field} />
                    </FormControl>
                    <FormDescription>Kami kirim ics + konfirmasi ke email ini.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="customerPhone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel htmlFor="customerPhone">No. HP (opsional)</FormLabel>
                    <FormControl>
                      <Input id="customerPhone" placeholder="08xx xxxx xxxx" autoComplete="tel" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="notes"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel htmlFor="notes">Catatan (opsional)</FormLabel>
                    <FormControl>
                      <Textarea id="notes" placeholder="Model potongan, alergi, request khusus..." rows={3} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {bookingMutation.isError ? (
                <Alert variant="destructive">
                  <AlertDescription>
                    {(bookingMutation.error as Error)?.message ?? "Gagal membuat booking. Coba slot lain."}
                  </AlertDescription>
                </Alert>
              ) : null}

              <div className="flex gap-3 pt-2">
                <Button type="button" variant="outline" onClick={onBack} disabled={bookingMutation.isPending}>
                  Kembali
                </Button>
                <Button type="submit" disabled={bookingMutation.isPending} className="flex-1 gap-2">
                  {bookingMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : isFree ? <Check className="h-4 w-4" /> : <CreditCard className="h-4 w-4" />}
                  {bookingMutation.isPending ? "Memproses..." : isFree ? "Konfirmasi booking gratis" : "Lanjut ke pembayaran"}
                </Button>
              </div>
              {!isFree ? (
                <p className="text-xs text-muted-foreground text-center">Test Stripe: 4242 4242 4242 4242 • 12/34 • 123</p>
              ) : null}
            </form>
          </Form>
        </CardContent>
      </Card>

      {/* Review */}
      <Card className="h-fit lg:sticky lg:top-6">
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Review booking</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-3 rounded-lg border bg-muted/30 p-4">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-semibold">{service.name}</p>
                <p className="text-xs text-muted-foreground">{service.description}</p>
              </div>
              <span className="h-3 w-3 rounded-full shrink-0 mt-1" style={{ background: service.color }} aria-hidden="true" />
            </div>
            <Separator />
            <dl className="space-y-2 text-sm">
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Staff</dt>
                <dd className="font-medium">{staffName}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Tanggal</dt>
                <dd className="font-medium tabular-nums">{formatJakartaDate(slot.startAt)}</dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Jam</dt>
                <dd className="font-medium tabular-nums">
                  {formatJakartaTime(slot.startAt)} – {formatJakartaTime(slot.endAt)} WIB
                </dd>
              </div>
              <div className="flex justify-between gap-4">
                <dt className="text-muted-foreground">Durasi</dt>
                <dd className="tabular-nums">
                  {service.durationMinutes}m + {service.bufferMinutes}m buffer
                </dd>
              </div>
            </dl>
            <Separator />
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Total</span>
              <span className="text-base font-semibold tabular-nums">{formatPrice(service.priceCents)}</span>
            </div>
          </div>

          <div className="rounded-lg bg-primary/5 border border-primary/20 p-3">
            <p className="text-xs leading-relaxed text-muted-foreground">
              Dengan konfirmasi, kamu menyetujui slot <span className="font-medium text-foreground">tidak bisa double-book</span> (EXCLUDE constraint). Pembatalan bisa via link /track.
            </p>
          </div>

          <ul className="text-xs text-muted-foreground list-disc pl-4 space-y-1">
            <li>Email konfirmasi + .ics otomatis</li>
            <li>Timezone Asia/Jakarta</li>
            <li>Reschedule/cancel via /track/[id]</li>
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
