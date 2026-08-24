"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import { ArrowLeft, ArrowRight, Check, CalendarDays, UserRound, Scissors, FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import { QueryProvider } from "@/components/providers";
import { ServiceStep } from "@/components/booking/ServiceStep";
import { StaffStep } from "@/components/booking/StaffStep";
import { CalendarStep } from "@/components/booking/CalendarStep";
import { FormStep } from "@/components/booking/FormStep";
import type { Service, Staff, Slot } from "@/lib/api";

const STEPS = [
  { id: 1, label: "Layanan", icon: Scissors },
  { id: 2, label: "Staff", icon: UserRound },
  { id: 3, label: "Jadwal", icon: CalendarDays },
  { id: 4, label: "Detail", icon: FileText },
] as const;

function BookFlowInner() {
  const router = useRouter();
  const [step, setStep] = React.useState<1 | 2 | 3 | 4>(1);
  const [service, setService] = React.useState<Service | null>(null);
  const [staff, setStaff] = React.useState<Staff | null>(null);
  const [staffId, setStaffId] = React.useState<string | null>(null);
  const [slot, setSlot] = React.useState<Slot | null>(null);

  const canNext = React.useMemo(() => {
    if (step === 1) return !!service;
    if (step === 2) return true; // Any available allowed
    if (step === 3) return !!slot;
    return false;
  }, [step, service, slot]);

  const handleNext = () => setStep((s) => Math.min(4, s + 1) as 1 | 2 | 3 | 4);
  const handleBack = () => setStep((s) => Math.max(1, s - 1) as 1 | 2 | 3 | 4);

  return (
    <div className="min-h-screen bg-background">
      {/* Header minimal */}
      <header className="sticky top-0 z-30 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3 sm:px-6">
          <Link href="/" className="flex items-center gap-2" aria-label="Kembali ke beranda">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-semibold text-sm">FB</div>
            <span className="text-sm font-semibold tracking-tight">FlowBarber Studio</span>
            <span className="hidden sm:inline-flex ml-2 rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">Booking</span>
          </Link>
          <div className="flex items-center gap-2">
            <Link href="/track/sample" className="hidden sm:inline-flex text-xs text-muted-foreground hover:text-foreground underline-offset-4 hover:underline">
              Lacak booking
            </Link>
            <ThemeToggle />
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-6 sm:px-6 sm:py-8">
        {/* Stepper */}
        <nav aria-label="Langkah booking" className="mb-6 sm:mb-8">
          <ol className="flex items-center gap-2 sm:gap-3">
            {STEPS.map((s, idx) => {
              const active = step === s.id;
              const done = step > s.id;
              const Icon = s.icon;
              return (
                <React.Fragment key={s.id}>
                  <li className="flex items-center gap-2">
                    <div
                      aria-current={active ? "step" : undefined}
                      className={[
                        "flex h-8 w-8 items-center justify-center rounded-full border text-sm font-medium transition-colors",
                        done ? "bg-primary text-primary-foreground border-primary" : active ? "bg-primary text-primary-foreground border-primary" : "bg-card text-muted-foreground",
                      ].join(" ")}
                    >
                      {done ? <Check className="h-4 w-4" /> : <Icon className="h-4 w-4" />}
                    </div>
                    <span className={["hidden sm:inline text-sm", active ? "font-semibold text-foreground" : done ? "text-foreground" : "text-muted-foreground"].join(" ")}>{s.label}</span>
                    {active ? <span className="sr-only">Langkah aktif</span> : null}
                  </li>
                  {idx < STEPS.length - 1 ? <Separator className="flex-1 max-w-[40px] sm:max-w-none" /> : null}
                </React.Fragment>
              );
            })}
          </ol>
          <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-muted">
            <motion.div
              className="h-full bg-primary"
              initial={false}
              animate={{ width: `${(step / 4) * 100}%` }}
              transition={{ type: "spring", stiffness: 300, damping: 30 }}
            />
          </div>
        </nav>

        {/* Content */}
        <Card className="overflow-hidden">
          <CardContent className="p-4 sm:p-6 space-y-6">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h1 className="text-lg font-semibold tracking-tight">
                  {step === 1 && "Pilih layanan"}
                  {step === 2 && "Pilih staff"}
                  {step === 3 && "Pilih jadwal"}
                  {step === 4 && "Isi detail kontak"}
                </h1>
                <p className="text-sm text-muted-foreground">
                  {step === 1 && "Card menampilkan durasi & harga — klik untuk pilih."}
                  {step === 2 && "Avatar staff + opsi Any available."}
                  {step === 3 && "Kalender slot realtime Asia/Jakarta — polling 30 detik."}
                  {step === 4 && "Form validasi Zod → Review → Checkout."}
                </p>
              </div>
              {step > 1 ? (
                <Button variant="ghost" size="sm" onClick={handleBack} className="gap-1.5">
                  <ArrowLeft className="h-4 w-4" /> Kembali
                </Button>
              ) : null}
            </div>

            <AnimatePresence mode="wait">
              <motion.div
                key={step}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.2 }}
              >
                {step === 1 && (
                  <ServiceStep
                    selectedId={service?.id ?? null}
                    onSelect={(id, svc) => {
                      setService(svc);
                      setSlot(null); // reset slot when service changes
                    }}
                  />
                )}
                {step === 2 && (
                  <StaffStep
                    serviceId={service?.id ?? null}
                    selectedId={staffId}
                    onSelect={(id, st) => {
                      setStaffId(id);
                      setStaff(st);
                      setSlot(null);
                    }}
                  />
                )}
                {step === 3 && (
                  <CalendarStep serviceId={service?.id ?? null} staffId={staffId} selectedSlot={slot} onSelect={(s) => setSlot(s)} />
                )}
                {step === 4 && (
                  <FormStep
                    service={service}
                    staff={staff}
                    slot={slot}
                    onBack={handleBack}
                    onSuccess={(bookingId) => router.push(`/book/success?id=${bookingId}`)}
                  />
                )}
              </motion.div>
            </AnimatePresence>

            {/* Footer nav for steps 1-3 */}
            {step < 4 ? (
              <div className="flex items-center justify-between gap-4 border-t pt-4">
                <Button variant="outline" onClick={handleBack} disabled={step === 1} className="gap-2">
                  <ArrowLeft className="h-4 w-4" /> Kembali
                </Button>
                <div className="flex items-center gap-3">
                  {step === 3 && slot ? (
                    <span className="hidden sm:inline text-xs text-muted-foreground tabular-nums">
                      {slot.staffName} • {new Date(slot.startAt).toLocaleTimeString("id-ID", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit" })} WIB
                    </span>
                  ) : null}
                  <Button onClick={handleNext} disabled={!canNext} aria-disabled={!canNext} className="gap-2">
                    Lanjut <ArrowRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            ) : null}
          </CardContent>
        </Card>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          Butuh bantuan? <Link href="/" className="underline underline-offset-4 hover:text-foreground">Hubungi kami</Link> • Slot realtime • Asia/Jakarta
        </p>
      </main>
    </div>
  );
}

export default function BookPage() {
  return (
    <QueryProvider>
      <BookFlowInner />
    </QueryProvider>
  );
}
