import Link from "next/link";
import { Clock, BadgeCheck, Sparkles, ArrowRight, Star, MapPin, Phone, Mail, Scissors, Users, CalendarCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/components/ui/accordion";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";

export const dynamic = "force-static";

const featured = [
  {
    name: "Classic Cut",
    duration: "30m",
    buffer: "10m",
    price: "Rp 85.000",
    desc: "Potongan klasik presisi, cuci + styling. Cocok harian, cepat & rapi.",
    color: "#7c3aed",
  },
  {
    name: "Premium Fade",
    duration: "45m",
    buffer: "10m",
    price: "Rp 120.000",
    desc: "Fade detail skin to 0, razor line, styling premium. Best seller.",
    color: "#8b5cf6",
  },
  {
    name: "Hair Color",
    duration: "90m",
    buffer: "15m",
    price: "Rp 250.000",
    desc: "Pewarnaan rambut dengan konsultasi warna, khusus staff Bayu.",
    color: "#f59e0b",
  },
];

const pricing = [
  { name: "Classic Cut", dur: "30m", buf: "10m", price: "Rp 85.000", popular: false },
  { name: "Premium Fade", dur: "45m", buf: "10m", price: "Rp 120.000", popular: true },
  { name: "Cut + Beard", dur: "60m", buf: "15m", price: "Rp 150.000", popular: false },
  { name: "Beard Trim", dur: "20m", buf: "10m", price: "Rp 50.000", popular: false },
  { name: "Hair Color", dur: "90m", buf: "15m", price: "Rp 250.000", popular: false },
  { name: "Father & Son", dur: "60m", buf: "15m", price: "Rp 180.000", popular: false },
  { name: "Grooming Package", dur: "75m", buf: "15m", price: "Rp 200.000", popular: true },
  { name: "Konsultasi Style 15m", dur: "15m", buf: "5m", price: "Gratis", popular: false },
];

const faqs = [
  { q: "Bagaimana cara booking?", a: "Klik Book Now → pilih layanan → pilih staff (atau Any available) → pilih tanggal & slot di Asia/Jakarta → isi nama/email → konfirmasi. Slot polling realtime setiap 30 detik, hanya slot yang muat durasi+buffer yang aktif." },
  { q: "Apakah bisa reschedule atau cancel?", a: "Bisa. Buka /track/[id] dari link email, lalu pilih Reschedule (Dialog + Calendar) atau Cancel. Slot yang dibatalkan langsung kembali tersedia. Sistem mencegah double-booking via EXCLUDE constraint." },
  { q: "Bagaimana pembayaran?", a: "Jika gratis (Konsultasi 15m) langsung CONFIRMED tanpa Stripe. Jika berbayar, diarahkan ke Stripe Checkout test 4242 4242 4242 4242 exp 12/34 CVC 123. Webhook idempotent via stripeEventId." },
  { q: "Zona waktu apa yang dipakai?", a: "Semua jadwal disimpan UTC di database, ditampilkan dalam Asia/Jakarta. Kamu lihat jam WIB, tidak perlu konversi manual. Buffer 10–15 menit otomatis mencegah bentrok." },
  { q: "White-label untuk bisnis lain?", a: "Ganti seed 5 menit: klinik (Konsultasi 30m, Scaling 60m), studio (Sesi 60m), konsultan (Discovery 30m). Warna primary ganti hue 260 → 160 emerald atau 25 amber via token OKLCH." },
];

export default function MarketingPage() {
  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-3 sm:px-6">
          <Link href="/" className="flex items-center gap-2.5" aria-label="FlowBarber Studio beranda">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground font-semibold text-sm tracking-tight">FB</div>
            <span className="text-sm font-semibold tracking-tight">FlowBarber Studio</span>
            <Badge variant="secondary" className="hidden sm:inline-flex ml-1 text-[11px]">Demo</Badge>
          </Link>
          <nav aria-label="Navigasi utama" className="flex items-center gap-2">
            <Link href="#pricing" className="hidden sm:inline-flex text-sm text-muted-foreground hover:text-foreground">Pricing</Link>
            <Link href="#faq" className="hidden sm:inline-flex text-sm text-muted-foreground hover:text-foreground">FAQ</Link>
            <ThemeToggle />
            <Button asChild size="sm" className="gap-1.5">
              <Link href="/book" aria-label="Book Now — mulai booking">Book Now <ArrowRight className="h-4 w-4" /></Link>
            </Button>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="absolute inset-0 -z-10 bg-gradient-to-b from-primary/[0.08] via-transparent to-transparent" aria-hidden="true" />
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 sm:py-16 lg:py-20">
          <div className="grid gap-8 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
            <div className="space-y-5">
              <Badge variant="secondary" className="gap-1.5">
                <Sparkles className="h-3 w-3" /> Booking 4-step • Slot realtime • Asia/Jakarta
              </Badge>
              <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl lg:text-[2.6rem] leading-tight">
                Potongan premium,
                <br />
                <span className="text-primary">tanpa antre.</span>
              </h1>
              <p className="max-w-xl text-sm leading-relaxed text-muted-foreground sm:text-base">
                FlowBarber Studio — booking 90 detik, slot realtime per tanggal, legend available/buffer/taken. Hanya slot yang muat durasi+buffer yang aktif. Gratis skip Stripe, berbayar via 4242.
              </p>
              <div className="flex flex-wrap gap-3">
                <Button asChild size="lg" className="gap-2">
                  <Link href="/book" aria-label="Book Now — ke halaman booking">Book Now <ArrowRight className="h-4 w-4" /></Link>
                </Button>
                <Button asChild variant="outline" size="lg">
                  <Link href="#pricing" aria-label="Lihat pricing">Lihat layanan</Link>
                </Button>
              </div>
              <div className="flex flex-wrap items-center gap-4 pt-2 text-xs text-muted-foreground">
                <span className="inline-flex items-center gap-1.5"><Star className="h-3.5 w-3.5 text-amber-500" /> 4.9/5 (342 ulasan)</span>
                <span className="inline-flex items-center gap-1.5"><Users className="h-3.5 w-3.5" /> 3 staff • 8 layanan</span>
                <span className="inline-flex items-center gap-1.5"><CalendarCheck className="h-3.5 w-3.5" /> ~1.500 booking Nov–Agu</span>
              </div>
            </div>

            {/* Hero visual — CSS only, no img for Lighthouse */}
            <div className="relative">
              <Card className="overflow-hidden border-primary/20">
                <div className="bg-muted p-3 flex items-center gap-2 border-b">
                  <span className="h-3 w-3 rounded-full bg-red-400" aria-hidden="true" />
                  <span className="h-3 w-3 rounded-full bg-yellow-400" aria-hidden="true" />
                  <span className="h-3 w-3 rounded-full bg-green-400" aria-hidden="true" />
                  <span className="ml-2 text-xs font-medium text-muted-foreground">flowbook.example.com/book — Step 3</span>
                </div>
                <CardContent className="p-4 space-y-4">
                  <div className="grid grid-cols-3 gap-2">
                    {["09:00 Andi", "09:30 Bayu", "10:00 Sari"].map((t) => (
                      <div key={t} className="rounded-md bg-primary text-primary-foreground text-center py-2 text-xs font-medium tabular-nums">{t} ✓</div>
                    ))}
                    <div className="rounded-md bg-amber-500/20 border border-amber-500/30 text-center py-2 text-xs tabular-nums">10:30 buffer</div>
                    <div className="rounded-md bg-muted text-muted-foreground text-center py-2 text-xs tabular-nums">11:00 taken</div>
                    <div className="rounded-md bg-primary text-primary-foreground text-center py-2 text-xs font-medium">11:30</div>
                  </div>
                  <div className="flex items-center gap-2 text-xs">
                    <span className="h-2 w-2 rounded-full bg-primary" /> available
                    <span className="h-2 w-2 rounded-full bg-amber-500/60" /> buffer
                    <span className="h-2 w-2 rounded-full bg-muted border" /> taken
                  </div>
                  <div className="rounded-lg bg-primary/5 border border-primary/15 p-3 flex items-center justify-between">
                    <span className="text-sm font-medium">Classic Cut • 30m • Rp 85.000</span>
                    <Badge>Terpilih</Badge>
                  </div>
                  <p className="text-xs text-muted-foreground text-center">Asia/Jakarta • polling 30 detik via TanStack</p>
                </CardContent>
              </Card>
              <div className="absolute -z-10 -right-6 -bottom-6 h-32 w-32 rounded-full bg-primary/10 blur-3xl" aria-hidden="true" />
            </div>
          </div>
        </div>
      </section>

      {/* Featured 3 */}
      <section aria-labelledby="featured-heading" className="mx-auto max-w-7xl px-4 sm:px-6 py-10">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 id="featured-heading" className="text-xl font-semibold tracking-tight">3 layanan unggulan</h2>
            <p className="text-sm text-muted-foreground">Paling sering dibooking — durasi & harga transparan.</p>
          </div>
          <Link href="/book" className="hidden sm:inline-flex text-sm font-medium text-primary hover:underline">Book sekarang →</Link>
        </div>
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {featured.map((f) => (
            <Card key={f.name} className="group relative overflow-hidden hover:bg-muted/50 transition-colors">
              <div className="absolute left-0 top-0 h-full w-1" style={{ background: f.color }} aria-hidden="true" />
              <CardHeader className="pb-3 pl-6">
                <div className="flex items-start justify-between gap-2">
                  <CardTitle className="text-base">{f.name}</CardTitle>
                  {f.name === "Premium Fade" ? <Badge>Popular</Badge> : null}
                </div>
                <CardDescription className="text-xs leading-relaxed">{f.desc}</CardDescription>
              </CardHeader>
              <CardContent className="flex items-center gap-2 pl-6">
                <Badge variant="outline" className="gap-1 tabular-nums">
                  <Clock className="h-3 w-3" /> {f.duration}
                </Badge>
                <span className="text-xs text-muted-foreground">+ buffer</span>
                <span className="ml-auto text-sm font-semibold tabular-nums">{f.price}</span>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Pricing */}
      <section id="pricing" aria-labelledby="pricing-heading" className="mx-auto max-w-7xl px-4 sm:px-6 py-10">
        <h2 id="pricing-heading" className="text-xl font-semibold tracking-tight">Pricing transparan</h2>
        <p className="text-sm text-muted-foreground">8 layanan • durasi jujur, buffer mencegah bentrok, gratis tetap bisa dibooking.</p>
        <div className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {pricing.map((p) => (
            <Card key={p.name} className={["relative", p.popular ? "border-primary ring-1 ring-primary" : ""].join(" ")}>
              {p.popular ? <div className="absolute -top-3 left-4 rounded-full bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground">Popular</div> : null}
              <CardHeader className="pb-2">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Scissors className="h-4 w-4 text-muted-foreground" /> {p.name}
                </CardTitle>
                <CardDescription className="text-xs tabular-nums">{p.dur} + {p.buf} buffer</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-lg font-semibold tabular-nums">{p.price}</p>
                <Button asChild variant={p.popular ? "default" : "outline"} size="sm" className="mt-3 w-full">
                  <Link href="/book" aria-label={`Book ${p.name}`}>Book</Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
        <div className="mt-4 flex items-center gap-2 text-xs text-muted-foreground">
          <BadgeCheck className="h-4 w-4 text-primary" /> Harga final, tanpa biaya tersembunyi. Konsultasi 15m gratis langsung CONFIRMED.
        </div>
      </section>

      {/* FAQ */}
      <section id="faq" aria-labelledby="faq-heading" className="mx-auto max-w-3xl px-4 sm:px-6 py-10">
        <h2 id="faq-heading" className="text-xl font-semibold tracking-tight text-center">FAQ</h2>
        <p className="text-sm text-muted-foreground text-center">Jawaban cepat sebelum booking.</p>
        <Accordion type="single" collapsible className="mt-6">
          {faqs.map((f, i) => (
            <AccordionItem key={i} value={`item-${i}`}>
              <AccordionTrigger className="text-sm text-left">{f.q}</AccordionTrigger>
              <AccordionContent className="text-sm leading-relaxed text-muted-foreground">{f.a}</AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </section>

      {/* CTA */}
      <section aria-label="Call to action" className="mx-auto max-w-7xl px-4 sm:px-6 pb-10">
        <Card className="overflow-hidden bg-primary text-primary-foreground">
          <CardContent className="p-6 sm:p-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 className="text-lg font-semibold">Siap potong tanpa antre?</h3>
              <p className="text-sm opacity-90">Booking 90 detik, konfirmasi + .ics otomatis, bisa reschedule di /track.</p>
            </div>
            <Button asChild variant="secondary" size="lg" className="gap-2 shrink-0">
              <Link href="/book" aria-label="Book Now — call to action">Book Now <ArrowRight className="h-4 w-4" /></Link>
            </Button>
          </CardContent>
        </Card>
      </section>

      {/* Footer */}
      <footer className="border-t bg-card">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 py-8">
          <div className="grid gap-8 sm:grid-cols-3">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className="h-7 w-7 rounded-lg bg-primary text-primary-foreground grid place-items-center text-xs font-semibold">FB</div>
                <span className="text-sm font-semibold">FlowBarber Studio</span>
              </div>
              <p className="text-xs leading-relaxed text-muted-foreground max-w-xs">Barbershop premium di Jakarta. Booking online, slot realtime, anti double-book. Demo FlowBook — ganti brand 5 menit untuk klinik/studio/konsultan.</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium">Kontak</p>
              <ul className="space-y-1.5 text-xs text-muted-foreground">
                <li className="flex items-center gap-2"><MapPin className="h-3.5 w-3.5" /> Jl. Senopati No. 12, Jakarta Selatan</li>
                <li className="flex items-center gap-2"><Phone className="h-3.5 w-3.5" /> 0812-3456-7890</li>
                <li className="flex items-center gap-2"><Mail className="h-3.5 w-3.5" /> hello@flowbook.example.com</li>
                <li className="flex items-center gap-2"><Clock className="h-3.5 w-3.5" /> Buka 07:00–21:00 WIB</li>
              </ul>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium">Tautan</p>
              <ul className="space-y-1.5 text-xs">
                <li><Link href="/book" className="text-muted-foreground hover:text-foreground underline-offset-4 hover:underline">Book Now</Link></li>
                <li><Link href="/book" className="text-muted-foreground hover:text-foreground underline-offset-4 hover:underline">Lacak booking /track/[id]</Link></li>
                <li><Link href="#pricing" className="text-muted-foreground hover:text-foreground underline-offset-4 hover:underline">Pricing</Link></li>
                <li><Link href="#faq" className="text-muted-foreground hover:text-foreground underline-offset-4 hover:underline">FAQ</Link></li>
              </ul>
            </div>
          </div>
          <Separator className="my-6" />
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between text-xs text-muted-foreground">
            <p>© 2026 FlowBarber Studio — FlowBook Demo. Dibuat dengan Next 15 + Go + Supabase.</p>
            <p>OKLCH violet 260 • Light 0.62 / Dark 0.68 • tabular-nums</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
