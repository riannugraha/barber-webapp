# PRD — FlowBook Booking & Scheduling Platform

> **Status:** Locked — 24 Agu 2026 | Stack: Next.js Vercel Hobby + Go Koyeb Eco + Supabase Free | Tanpa Bun

## 1. Ringkasan Eksekutif

FlowBook adalah platform booking & scheduling untuk SME jasa (barbershop, klinik, studio, konsultan, rental). Demo ini dibangun untuk **impress client freelance dalam 2 menit klik** — terlihat mahal, white-label-ready, dan menunjukkan kemampuan full-stack end-to-end.

**Seed brand demo:** FlowBarber Studio. Ganti seed 5 menit untuk niche lain.

**Tujuan demo:**
- Client SME lihat FlowBook dan langsung membayangkan versi untuk bisnisnya
- Menunjukkan: Auth + RBAC + Calendar Engine + Realtime + Payment + Email + Dashboard + AI
- Shippable 14 hari, stack 2026 stable, deploy free tier

**Non-goal MVP:**
- Multi-tenant signup (schema siap `organizationId`, UI nanti)
- WhatsApp Gateway (hanya Email, WA upsell)
- Mobile native app (responsive web cukup)
- Role ADMIN (cukup OWNER/STAFF/CUSTOMER)

---

## 2. Persona

| Persona | Deskripsi | Kebutuhan | Akses |
|---|---|---|---|
| **OWNER** | Pemilik barbershop | Lihat revenue, kelola layanan/staff/jadwal, blok waktu | Full `/app` |
| **STAFF** | Barber (Andi, Bayu, Sari) | Lihat booking miliknya, blok jamnya, tidak lihat revenue total | `/app` terbatas |
| **CUSTOMER** | Pelanggan umum | Booking layanan, bayar, track status | `/`, `/book`, `/track/[id]` saja |

---

## 3. Layanan yang Bisa di-Book (Seed 8 Layanan)

CRUD oleh OWNER di `/app/services`. Seed default FlowBarber:

| Kategori | Layanan | Durasi | Buffer | Harga | Staff Eligible | Warna |
|---|---|---|---|---|---|---|
| Signature | Classic Cut | 30m | 10m | Rp 85k | All | biru |
| Signature | Premium Fade | 45m | 10m | Rp 120k | Andi, Bayu | violet |
| Signature | Cut + Beard | 60m | 15m | Rp 150k | Andi | indigo |
| Grooming | Beard Trim | 20m | 10m | Rp 50k | All | teal |
| Grooming | Hair Color | 90m | 15m | Rp 250k | Bayu | amber |
| Package | Father & Son | 60m | 15m | Rp 180k | Andi | emerald |
| Package | Grooming Package | 75m | 15m | Rp 200k | Andi, Bayu | rose |
| Konsultasi | Konsultasi Style 15m | 15m | 5m | Gratis | Bayu | slate |

**Aturan engine:**
- `durasi + buffer` = slot tidak bisa double-book
- Skill staff memfilter slot (Hair Color hanya Bayu)
- Gratis skip Stripe, langsung CONFIRMED

**White-label:** Ganti seed untuk klinik (Konsultasi Umum 30m, Scaling 60m), studio (Sesi Studio 60m), konsultan (Discovery 30m).

---

## 4. User Journeys

### 4.1 Customer — Booking (Critical Path)

1. `/` Landing → CTA "Book Now"
2. `/book` Step 1: Pilih layanan (Card durasi/harga)
3. Step 2: Pilih staff (Avatar + "Any available")
4. Step 3: Kalender slot realtime per tanggal, timezone `Asia/Jakarta`, legend available/buffer/taken — hanya slot muat `durasi+buffer` yang aktif
5. Step 4: Form nama/email/notes (Zod) → Review
6. Checkout: jika harga >0 → Stripe Checkout (test `4242`), jika gratis → langsung success
7. `/book/success` → detail + ics download + email Resend
8. `/track/[id]` → cek status, reschedule/cancel

### 4.2 Owner — Operasional

1. Login → `/app` Dashboard KPI + Chart 10 bulan
2. `/app/calendar` Week view 07-21, drag blok libur
3. `/app/bookings` DataTable filter status/tanggal/staff, klik detail → reschedule/cancel
4. `/app/services` CRUD layanan
5. `/app/staff` CRUD staff + editor availability mingguan + override tanggal
6. `/app/settings` Org timezone/logo

### 4.3 Staff

1. Login → `/app` hanya booking miliknya
2. `/app/calendar` blok jamnya sendiri
3. Tidak bisa CRUD layanan

### 4.4 AI Receptionist (Wow)

Floating widget di `/book`:
- User: "ada jadwal potong besok jam 3?"
- AI: cek `checkAvailability` → tawar slot → eksekusi `createBooking` → streaming SSE

---

## 5. Fitur Lengkap

### MVP (14 hari)

- Auth JWT (access 15m + refresh 7d httpOnly)
- RBAC OWNER/STAFF/CUSTOMER (Go middleware + Next middleware)
- Calendar engine timezone-aware (UTC simpan, render Asia/Jakarta)
- Realtime slot via WebSocket (Go gorilla/websocket, Koyeb native)
- Stripe Checkout test + webhook idempotent
- Resend email: BookingConfirmed, Reminder H-1, Cancelled + Cron
- Dashboard 5 row (KPI, Area Chart 10 bulan, Pie/Bar/Heatmap, Top Customers, Recent)
- Upload avatar layanan (Supabase Storage)
- Seed 1 Nov 2025 → 24 Agu 2026 ~1.500 bookings

### V2 (Post-MVP, tidak di PRD ini)

- Multi-tenant org signup
- WA reminder, Google Calendar sync
- Review/rating, promo code
- Export CSV, print invoice

---

## 6. Halaman Inventory

### Public
- `/` Landing (Hero, 3 layanan unggulan, pricing, FAQ Accordion, footer)
- `/book` (4 step), `/book/success`, `/track/[id]`
- `/login`, `/register`

### App (Protected)
- `/app` Dashboard
- `/app/calendar` Week view
- `/app/bookings` + `/app/bookings/[id]`
- `/app/services`
- `/app/staff`
- `/app/customers`
- `/app/settings`

Global: Sidebar collapsible, Topbar (Command K, ThemeToggle, Avatar), Toaster, AI Widget.

---

## 7. Metrik Sukses Demo

- Customer booking end-to-end < 90 detik
- Dashboard load < 1.5s, chart 10 bulan render < 800ms
- Double-booking 0 (EXCLUDE constraint)
- Lighthouse > 90, a11y 0 violation (axe)

---

## 8. Batasan Free Tier

- Vercel Hobby 100GB BW, Supabase 500MB DB / 1GB storage / pause 7 hari, Koyeb Eco 0.1 vCPU cold start 250ms
- Mitigasi: cron ping `/health` tiap 5m saat demo + daily `SELECT 1` ke Supabase

---

## 9. Open Questions — Resolved

- Tanpa Bun ✅ (pnpm), tanpa ADMIN ✅, Stripe test ✅, seed Nov-Agu ✅, dashboard kompleks ✅, violet 260 ✅, Opsi A TRUNCATE ✅ — semua locked 24 Agu 2026.
