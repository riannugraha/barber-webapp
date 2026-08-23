# PLAN — FlowBook Booking & Scheduling Platform

> **Status:** Draft — 24 Agu 2026 | **Sprint:** 14 Hari (25 Agu → 08 Sep 2026)
> **Stack Locked:** Next 15.3.5 / React 19 / TS 5.6 / Tailwind 4 Oxide / OKLCH violet 260 / Go 1.23.8 / Echo 4.13 / pgx 5.7 / sqlc 2.16 / Supabase Free (pooler 6543) / Vercel Hobby / Koyeb Eco | pnpm 9 tanpa Bun
> **Pemilik:** `planner` owns `docs/` → delegasi via orchestrator ke `frontend`/`backend`/`tester` → gate `reviewer`
> **Sumber Kebenaran:** `docs/PRD.md` + `TECH.md` + `DESIGN.md` + `TESTING.md` + `DEPLOYMENT.md` (Locked 24 Agu 2026)

---

## 1. Guardrails & Locked Decisions

- **Design:** `apps/web/app/globals.css` `@theme` OKLCH — light `background oklch(0.99 0 0)` `primary oklch(0.62 0.19 260)` / dark `background oklch(0.14 0.01 260)` `card oklch(0.19 0.015 260)` `primary oklch(0.68 0.16 260)` (+15% luminance, bukan neon). Radius `0.625rem`, shadow-none di light, elevation via lightness di dark, `tabular-nums` untuk KPI.
- **DB Anti Double-Book:** `EXCLUDE USING gist (staff_id WITH =, tstzrange(start_at, end_at) WITH &&) WHERE (status IN ('PENDING','CONFIRMED'))` + `CREATE INDEX USING gist (tstzrange(start_at,end_at))`. Simpan UTC, render `organization.timezone` (`Asia/Jakarta`). `pgx` + `sqlc`, bukan `database/sql` generic.
- **Isolasi E2E Opsi A:** `TRUNCATE bookings,payments,customers RESTART IDENTITY CASCADE` + `seedMinimal(10)` per `beforeEach` via `POST /test/reset` (`x-test-secret`, hanya `APP_ENV=test`) — 150-250ms. `POST /test/seed-full` (1.500) hanya untuk chart. `workers:1` di CI (pooler 60 conns, Koyeb 0.1 vCPU). Project terpisah `flowbook-qa` vs `flowbook-prod`.
- **Seed:** `1 Nov 2025 → 24 Agu 2026 ~1.500 bookings` + 60 customers + 3 staff (Andi, Bayu, Sari) + 8 layanan PRD §3 (durasi+buffer, skill filter, gratis skip Stripe).
- **Dashboard 5 Row (DESIGN §6):** Row1 4 KPI + Row2 Area 10 bulan + Row3 Pie 35% / Bar staff / Heatmap 7x15 + Row4 Top 15 + Recent 10 + Row5 Insight (Busiest Des 2025, Cancel 7.2%).
- **Stripe:** test `4242 4242 4242 4242` exp `12/34` CVC `123`, `sk_test_*` + `whsec_*` webhook idempotent `stripeEventId UNIQUE`.
- **Deploy:** `DATABASE_URL=postgres://...:6543/postgres?pgbouncer=true` (transaction mode), `PORT=8080`, `GET /health`, cron `*/5 * * * *` `apps/web/app/api/ping/route.ts` → Koyeb, daily `SELECT 1` anti-pause Supabase.

---

## 2. Milestones 14 Hari

| Milestone | Hari | Goal | Exit Criteria |
|---|---|---|---|
| **M0 Foundation & Contract** | **D1–D2** (25–26 Agu) | DB + sqlc + openapi.yaml + tokens | `migrate up` OK di `flowbook-qa`, `sqlc generate` pass, `GET /health` + `GET /openapi.yaml` 200, `globals.css` build pass |
| **M1 Engine & Auth** | **D3–D6** (27–30 Agu) | Calendar engine + JWT/RBAC + seed | `TestGetSlots` 100% (DST, buffer 10/15m, overnight, override libur, Hair Color→Bayu), `EXCLUDE` 409, auth `15m+7d httpOnly` 401/403 pass, `go run ./cmd/seed` 1.500 rows |
| **M2 Booking & Dashboard** | **D7–D10** (31 Agu–03 Sep) | 4-step booking + dashboard 5 row + app shell | `/book` E2E mock Stripe → CONFIRMED + ics + Resend, dashboard Recharts Area 10 titik <800ms, `/app/*` CRUD services/staff |
| **M3 Realtime + Polish + Deploy** | **D11–D14** (04–08 Sep) | WS + Stripe webhook + AI SSE + a11y + deploy | 2-context slot realtime hilang, webhook idempotent, SSE `checkAvailability/createBooking`, `axe 0 violation`, `pnpm playwright` 8/8 green di Vercel Preview + Koyeb Eco |

```mermaid
gantt
    title FlowBook 14 Hari
    dateFormat  YYYY-MM-DD
    axisFormat %d %b
    section M0
    T01 DB+sqlc+openapi      :done, t01, 2026-08-25, 1.5d
    T09 Tokens               :active, t09, 2026-08-25, 1d
    T07 Seed+Health          :t07, 2026-08-26, 1d
    section M1
    T02 Auth                 :t02, 2026-08-26, 1.5d
    T03 Engine               :crit, t03, 2026-08-27, 2d
    T04 Bookings             :t04, 2026-08-29, 1.5d
    section M2
    T05 Payments             :t05, 2026-08-31, 1d
    T06 Dashboard API        :t06, 2026-08-31, 1d
    T10 Booking UI           :crit, t10, 2026-08-31, 2d
    T11 Dashboard 5Row       :crit, t11, 2026-09-02, 2d
    T12 App Shell            :t12, 2026-09-02, 1.5d
    section M3
    T08 WS+SSE               :t08, 2026-09-04, 1d
    T13 Go TDD               :t13, 2026-09-05, 1.5d
    T14 E2E Opsi A           :crit, t14, 2026-09-06, 2d
```

**Dependency:** `T01 (openapi.yaml+sqlc) ─┬─► T02►T04►T05►T08` dan `┬─► T03` dan `┬─► T07►T14` dan `└─► T09►T10►T11►T12`. Setelah **T01** merge, `frontend` dan `backend` jalan **parallel dalam 1 turn** (`task:frontend` + `task:backend`).

---

## 3. Tasks File-Level (14 Tasks)

### Ringkasan Eksekusi

| ID | Path Utama | Owner | Deps | Est | M |
|---|---|---|---|---|---|
| **T01** | `apps/api/migrations/0001_init.up.sql`, `apps/api/sql/queries/*.sql`, `apps/api/sqlc.yaml`, `apps/api/openapi.yaml` | **backend** | — | 1.5d | M0 |
| **T02** | `apps/api/internal/auth/{handler,service,repo}.go`, `apps/api/internal/middleware/jwt.go`, `role.go`, `apps/api/internal/config/config.go`, `apps/api/cmd/api/main.go` | **backend** | T01 | 1.5d | M1 |
| **T03** | `apps/api/internal/availability/service.go`, `handler.go` | **backend** | T01 | 2d | M1 |
| **T04** | `apps/api/internal/bookings/{handler,service,repo}.go`, `sql/queries/bookings.sql` | **backend** | T02,T03 | 1.5d | M1 |
| **T05** | `apps/api/internal/payments/{handler,service}.go`, `apps/api/internal/email/resend.go` | **backend** | T04 | 1d | M2 |
| **T06** | `apps/api/internal/dashboard/{handler,service}.go`, `sql/queries/dashboard.sql` | **backend** | T01,T07 | 1d | M2 |
| **T07** | `apps/api/cmd/seed/main.go`, `apps/api/internal/health/handler.go`, `apps/api/internal/testhelpers/reset.go` | **backend** | T01 | 1d | M0 |
| **T08** | `apps/api/internal/ws/hub.go`, `apps/api/internal/chat/handler.go` | **backend** | T03,T04 | 1d | M3 |
| **T09** | `apps/web/app/globals.css`, `app/layout.tsx`, `components/ui/**`, `components/ThemeToggle.tsx` | **frontend** | T01 | 1d | M0 |
| **T10** | `apps/web/app/(booking)/book/page.tsx`, `components/booking/**`, `app/(booking)/track/[id]/page.tsx`, `app/(booking)/book/success/page.tsx`, `app/(marketing)/page.tsx` | **frontend** | T01,T09 | 2d | M2 |
| **T11** | `apps/web/app/(app)/app/page.tsx`, `components/dashboard/**`, `lib/queryKeys.ts` | **frontend** | T01,T06,T09 | 2d | M2 |
| **T12** | `apps/web/app/(app)/app/{calendar,bookings,services,staff,customers,settings}/page.tsx`, `lib/api.ts`, `generated/api.ts`, `middleware.ts` | **frontend** | T01,T09 | 1.5d | M2 |
| **T13** | `apps/api/internal/availability/service_test.go`, `bookings/*_test.go`, `payments/webhook_test.go`, `testhelpers/postgres.go` | **tester** | T03,T04 | 1.5d | M3 |
| **T14** | `apps/web/e2e/fixtures.ts`, `e2e/pages/**`, `e2e/01-08*.spec.ts`, `vitest.config.ts`, `playwright.config.ts`, `**/*.test.tsx` | **tester** | T07,T10,T11 | 2d | M3 |

---

### T01 — DB Contract & OpenAPI (Backend) — *Unblock semua* ✅
**Files:** `migrations/0001_init.up.sql` (+ `.down.sql`), `sql/queries/*.sql` (services, staff, availability, bookings, payments, dashboard), `sqlc.yaml`, `openapi.yaml`, `cmd/api/main.go` health stub
**AC:**
- [x] `0001_init` buat `organizations, users, services, staff, staff_services, availability, availability_overrides, bookings, payments, refresh_tokens` — `bookings` punya `EXCLUDE USING gist (staff_id WITH =, tstzrange(start_at,end_at) WITH &&) WHERE (status IN ('PENDING','CONFIRMED'))` + `CREATE INDEX USING gist (tstzrange(start_at,end_at))` + `status CHECK (PENDING,CONFIRMED,CANCELLED)`.
- [x] `timestamptz` UTC, `organizations.timezone DEFAULT 'Asia/Jakarta'`, `pgx` native `timestamptz`.
- [x] `sqlc generate` clean → `db/models.go` + `db/queries/*.gen.go` (tanpa `database/sql`).
- [x] `openapi.yaml` v1.0.0 expose `GET /health`, `POST /auth/{register,login,refresh,logout}`, `GET /services`, `POST /services (OWNER)`, `GET /staff`, `GET /availability/slots?serviceId&staffId&date&tz`, `POST /bookings`, `GET /bookings*`, `POST /payments/{checkout-session,webhook}`, `GET /dashboard`, `GET /ws`, `GET /openapi.yaml` — dipakai `orval 7` tanpa edit manual.
- [x] `.env.example` hanya placeholder `DATABASE_URL=postgres://...:6543/postgres?pgbouncer=true` + `JWT_SECRET=[openssl rand -hex 32]` + `REFRESH_SECRET=[...]` — tidak ada secret hardcode.
**Status:** DONE — `internal/db` generated, `go vet` pass, `EXCLUDE` verified

### T02 — Auth & RBAC (Backend)
**Files:** `internal/auth/handler.go`, `service.go`, `repo.go`, `internal/middleware/jwt.go`, `role.go`, `cors.go`, `requestID.go`, `internal/config/config.go`
**AC:**
- [ ] `POST /auth/register` `bcrypt`, `POST /auth/login` issue `access 15m` (`jwt/v5`) + `refresh 7d httpOnly Secure SameSite=Lax` cookie (`refresh_tokens {id,user_id,token_hash,expires_at,revoked}`), `POST /auth/refresh`, `POST /auth/logout` revoke.
- [ ] `JWTMiddleware` validasi `Authorization: Bearer <access>`, `RequireRole(OWNER,STAFF)` → `403` jika `CUSTOMER` `POST /services`, `401` → Next `middleware.ts` redirect `/login`.
- [ ] `validator/v10` `BindAndValidate` error `422` Zod-compatible, `slog` JSON.
- [ ] `CORS AllowOrigins: [https://flowbook-xxx.vercel.app, http://localhost:3000]`.
- [ ] Unit `httptest` + `echo.New()` `201/403` pass.
**Deps:** T01

### T03 — Calendar Engine (Backend) — *Highest Risk*
**Files:** `internal/availability/service.go`, `handler.go`, `repo.go`, `sql/queries/availability.sql`
**AC:**
- [ ] `GetSlots(ctx, serviceId, staffId, date, tz)` implement TECH §5: load `availability` mingguan + `availability_overrides` tanggal target → generate grid `15m` dalam `org timezone` → query `WHERE tstzrange(start_at,end_at) && $range` → filter slot muat `duration+buffer` tanpa overlap → cache 30s memory.
- [ ] Handle: `buffer 10/15m` block next slot, `overnight` 21:00→02:00, `override libur` → 0 slot, `staff skill` (Hair Color hanya Bayu, Father & Son hanya Andi), `Any available` = union staff eligible.
- [ ] Timezone-aware: `2025-11-02 Asia/Jakarta` DST edge tidak off-by-hour; response `start_at` ISO `timestamptz`.
- [ ] Hanya `PENDING|CONFIRMED` block; `CANCELLED` tidak block. Legend `available/buffer/taken` konsisten.
- [ ] `GET /availability/slots?serviceId&staffId&date&tz` `200 {slots:[{start,end,available,staffId}]}`.
**Deps:** T01 | **Catatan:** TDD dulu (T13) sebelum code, target `100%` coverage service.

### T04 — Bookings Core (Backend)
**Files:** `internal/bookings/handler.go`, `service.go`, `repo.go`, `sql/queries/bookings.sql`
**AC:**
- [ ] `POST /bookings` validasi via `availability.Service.GetSlots`, insert `tstzrange`, jika overlap `409` dari `23P01 exclusion` — 0 double-booking.
- [ ] `GET /bookings?from&to&status&staffId` + pagination, `GET /bookings/:id`, `POST /bookings/:id/cancel`, `POST /bookings/:id/reschedule` (cancel+create dalam `tx`).
- [ ] RBAC: `STAFF` hanya `bookings` miliknya, `OWNER` full, `CUSTOMER` hanya `POST` + `GET /track`.
- [ ] `hub.Broadcast(orgID, {type:"slot_taken"})` setelah create/cancel untuk realtime.
**Deps:** T02, T03

### T05 — Stripe & Email (Backend)
**Files:** `internal/payments/handler.go`, `service.go`, `internal/email/resend.go` + templates `BookingConfirmed`, `Cancelled`, `Reminder H-1`
**AC:**
- [ ] `POST /payments/checkout-session` `stripe-go 76` `sk_test_*` redirect, `POST /payments/webhook` verify `whsec_*` signature, idempotent via `stripeEventId UNIQUE` → retry `200`.
- [ ] Harga `0` (Konsultasi Style 15m) skip Stripe → langsung `CONFIRMED`.
- [ ] `Resend` kirim `BookingConfirmed` + `ics` attach, `Cancelled`; cron Go tiap `15m` untuk `Reminder H-1`.
- [ ] Kartu test `4242` manual smoke pass di `flowbook-qa`.
**Deps:** T04

### T06 — Dashboard Aggregates (Backend)
**Files:** `internal/dashboard/handler.go`, `service.go`, `sql/queries/dashboard.sql`
**AC:**
- [ ] `GET /dashboard?from&to&granularity&tz` return 5-row: `kpi {revenue, bookings, occupancy, avgTicket, delta}`, `area 10 titik Nov 2025→Agu 2026`, `pie {Classic Cut 35%}`, `bar {Andi 90/Bayu 70/Sari 20}`, `heatmap 7x15`, `topCustomers 15 (Siti 18x)`, `recent 10`, `insight {busiestMonth: Des 2025, cancelRate 7.2%, utilization}`.
- [ ] Query `tstzrange` + `DATE_TRUNC` + `GROUP BY` di DB dengan index `start_at`, bukan JS.
- [ ] `OWNER` full, `STAFF` scoped miliknya. Harga `Rp` `tabular-nums`.
**Deps:** T01, T07 (butuh seed full)

### T07 — Seed & Health & Test Infra (Backend)
**Files:** `cmd/seed/main.go`, `internal/health/handler.go`, `internal/testhelpers/reset.go` (`POST /test/reset`, `/test/seed-full`), `supabase/` symlink `migrations`, `vercel.json` (`/api/ping`), `.github/workflows/keep-warm.yml`
**AC:**
- [ ] `go run ./cmd/seed` loop `2025-11-01 → 2026-08-24` bobot weekend/musiman → `~1.500 bookings` + 60 customers + 8 layanan + 3 staff; flag `--minimal` → 10 rows untuk `beforeEach` 150ms.
- [ ] `GET /health` → `{"status":"ok","db":"up"}` untuk Koyeb health check `PORT=8080`.
- [ ] `POST /test/reset` + `/test/seed-full` guard `APP_ENV=test && x-test-secret==TEST_SECRET` → `TRUNCATE bookings,payments,customers RESTART IDENTITY CASCADE` + `seedMinimal`; selain itu `404`. Tidak bypass `EXCLUDE`.
- [ ] Keep-warm: `vercel.json` `{"crons":[{"path":"/api/ping","schedule":"*/5 * * * *"}]}` → `fetch GET $API_URL/health`; GH Actions daily `psql "$DATABASE_URL" -c "SELECT 1"` anti-pause 7 hari.
**Deps:** T01

### T08 — Realtime WS & AI Receptionist SSE (Backend)
**Files:** `internal/ws/hub.go`, `client.go`, `handler.go` (`GET /ws`), `internal/chat/handler.go` (`POST /chat` SSE)
**AC:**
- [ ] `gorilla/websocket 1.5` Hub `Broadcast(orgID, {type:"slot_taken", slot})` — Koyeb native tanpa Pusher; Next `useWebSocket` `ws://api/ws?orgId=...` invalidate `queryKeys.slots`.
- [ ] `POST /chat` `text/event-stream` proxy `openai-go` tools `checkAvailability({service,date})` → `availability.Service.GetSlots` + `createBooking({service,staff,slot,customer})` → `bookings.Service.Create` streaming.
- [ ] Tidak pakai Vercel AI SDK (Go SSE manual via Echo).
**Deps:** T03, T04

### T09 — Design Tokens & shadcn (Frontend) — *Unblock UI*
**Files:** `apps/web/app/globals.css`, `app/layout.tsx`, `components/ui/**` (button, card, input, form, select, calendar, dialog, sheet, dropdown-menu, avatar, badge, table, tabs, popover, separator, skeleton, switch, textarea, alert, accordion, chart, sonner), `components/ThemeToggle.tsx`
**AC:**
- [ ] `globals.css` `@import "tailwindcss"; @import "tw-animate-css"; @theme { --color-primary: oklch(0.62 0.19 260) ... }` + `.dark { --color-background: oklch(0.14 0.01 260); --color-card: oklch(0.19 0.015 260); --color-primary: oklch(0.68 0.16 260) }` exact DESIGN §2.
- [ ] `ThemeProvider next-themes` `attribute="class" defaultTheme="system" enableSystem`, `ThemeToggle` `Sun/Moon` rotate `dark:-rotate-90`.
- [ ] `shadcn@latest add ...` 22 komponen, `pnpm build` pass, kontras 7:1 AAA, `prefers-reduced-motion`.
**Deps:** T01

### T10 — Booking Flow 4-Step Public (Frontend)
**Files:** `app/(booking)/book/page.tsx` (4-step), `components/booking/ServiceStep.tsx`, `StaffStep.tsx`, `CalendarStep.tsx`, `FormStep.tsx`, `app/(booking)/book/success/page.tsx`, `app/(booking)/track/[id]/page.tsx`, `app/(marketing)/page.tsx` (Hero + 3 layanan unggulan + pricing + FAQ Accordion + footer + CTA Book Now)
**AC:**
- [ ] Step1 Card layanan durasi/harga, Step2 Avatar staff + `Any available`, Step3 kalender slot realtime per tanggal `Asia/Jakarta` legend `available/buffer/taken` hanya slot muat `duration+buffer` aktif, Step4 `rhf+zod` nama/email/notes → Review → Checkout Stripe `4242` atau langsung success jika gratis.
- [ ] `/book/success` konfirmasi + `ics` download + email log; `/track/[id]` detail + reschedule/cancel `Dialog+Calendar`.
- [ ] Mutasi via `ky 1.8` ke `NEXT_PUBLIC_API_URL` (bukan Server Actions), TanStack invalidate via WS, `getByRole` selector, `Skeleton` + empty `Belum ada booking...` bukan `No data`.
- [ ] Landing SSR, booking client, `Lighthouse >90`.
**Deps:** T01, T09

### T11 — Dashboard 5 Row (Frontend) — *Wow*
**Files:** `app/(app)/app/page.tsx`, `components/dashboard/KpiCards.tsx`, `RevenueAreaChart.tsx`, `ServicePie.tsx`, `StaffBar.tsx`, `Heatmap.tsx`, `TopCustomers.tsx`, `RecentTable.tsx`, `InsightRow.tsx`, `lib/queryKeys.ts`
**AC:**
- [ ] Row1 4 KPI `Revenue Rp 142jt | Bookings 1.542 | Occupancy 68% | Avg Ticket Rp 128k` + `Δ vs bulan lalu +9%` `font-mono tabular-nums` klik → ` /app/bookings?from&to`.
- [ ] Row2 Area Chart Revenue 10 bulan 1 warna `primary` grid halus toggle `Harian/Mingguan/Bulanan` render `<800ms`, tooltip `Rp`.
- [ ] Row3 3 kolom `Pie Layanan 35% Classic Cut` + `Bar Andi 90/Bayu 70/Sari 20` + `Heatmap 7x15 Jam Sibuk`; Row4 `Top 15 Siti 18x` + `Recent 10 Badge`; Row5 `Busiest Dec 2025 | Cancel 7.2% | Utilization`.
- [ ] `Recharts` `isAnimationActive`, responsive KPI 4→2x2 mobile, chart scroll horizontal, data dari `GET /dashboard` via `orval`.
- [ ] 3-second test: Owner jawab "rame ga?" 3 detik, tidak pakai gradient slop / pie chart abuse (pie hanya 1x di dashboard).
**Deps:** T01, T06, T09

### T12 — App Shell & CRUD Protected (Frontend)
**Files:** `app/(app)/layout.tsx` (Sidebar collapsible `16rem→3.5rem` + Header `Command K + ThemeToggle + Avatar`), `app/(app)/app/{calendar,bookings,services,staff,customers,settings}/page.tsx`, `lib/api.ts`, `generated/api.ts`, `middleware.ts`
**AC:**
- [ ] Sidebar `shadcn Sidebar` mobile `Sheet` drawer, Header sticky `backdrop-blur`, `Toaster sonner` bottom-right, AI Widget floating di `/book`.
- [ ] `TanStack 5.80 + zustand 5 + rhf+zod + orval 7 + ky 1.8` — `pnpm gen` regenerate `generated/api.ts` dari `openapi.yaml`.
- [ ] `calendar` week 07-21 7 kolom drag blok libur `framer-motion`, `bookings` DataTable filter status/date/staff + search + pagination, `services` CRUD Dialog (nama/durasi/buffer/harga/warna/active `Switch`), `staff` availability editor mingguan + override tanggal, `customers` list + history, `settings` org timezone/logo upload Supabase Storage 1GB.
- [ ] `middleware.ts` redirect `/app/*` if `401`, `Command K ⌘K` Go to Bookings/Add Service/Toggle theme, `Toaster` untuk 422/409.
**Deps:** T01, T09

### T13 — Go TDD & Integration (Tester)
**Files:** `internal/availability/service_test.go`, `internal/bookings/service_test.go`, `internal/bookings/repo_test.go`, `internal/payments/webhook_test.go`, `testhelpers/postgres.go`
**AC:**
- [ ] Table-driven `TestGetSlots` 100% tanpa DB: `buffer 10m blocks next`, `DST Asia/Jakarta 2025-11-02`, `override libur 0 slot`, `Hair Color hanya Bayu`, `overnight`, `Any available` — TDD red-green sebelum T03.
- [ ] Integration `testcontainers-go 0.33+ postgres:16-alpine` 1 container per `TestMain` reuse, `golang-migrate 4.18` up, `TestCreateBooking_ExcludeOverlap` assert `23P01 exclusion`, `TestTransactionRollback` via `tx.Rollback` — tidak mock EXCLUDE.
- [ ] Handler `httptest` + `echo.New()` `201/403/409` pass, `go test ./... -cover` overall `>80%`, `availability 100%`, `go vet` pass.
**Deps:** T03, T04

### T14 — E2E Playwright Opsi A + Vitest (Tester)
**Files:** `e2e/fixtures.ts`, `e2e/pages/BookPage.ts`, `DashboardPage.ts`, `e2e/01-08*.spec.ts`, `vitest.config.ts` (`jsdom` + `setupFiles`), `playwright.config.ts`, `components/booking/CalendarSlots.test.tsx`
**AC:**
- [ ] `playwright.config.ts`: `testDir e2e`, `workers:1` di CI (`pooler 60, Koyeb 0.1 vCPU`) `fullyParallel` lokal, `retries:2` di CI, `trace:on-first-retry`, `screenshot:only-on-failure`, `webServer pnpm --filter web dev`, `chromium` only, `baseURL http://localhost:3000`, bypass `x-vercel-protection-bypass` jika aktif.
- [ ] `fixtures.ts` `beforeEach` → `POST ${API_URL}/test/reset` `x-test-secret` `TRUNCATE + seedMinimal(10)` 150-250ms; `POST /test/seed-full` hanya untuk `04-owner-dashboard`; auth via `POST /auth/login` `owner@flowbook.test`.
- [ ] 8 critical green (TESTING §4): `01-public-booking` (Stripe mock `page.route` → CONFIRMED+email) | `02-slot-realtime` 2 contexts slot hilang | `03-overlap-prevent` `409` | `04-owner-dashboard` KPI + chart 10 bulan | `05-staff-restrict` STAFF `POST /services →403` | `06-cancel-reschedule` `CANCELLED` | `07-theme` `html.dark` persist reload | `08-a11y` `@axe-core/playwright` 0 violation.
- [ ] Vitest `CalendarSlots` + `BookingForm` `>70%`, `pnpm vitest --coverage` + `pnpm playwright test` di `ci.yml` (lint→go test→vitest→playwright) artifact 7 hari. Project `flowbook-qa` only.
**Deps:** T07, T10, T11

---

## 4. Paralelisasi & Data Contract Strategy

**Independence (wajib parallel):**
- Setelah **T01** (D1 siang) merge, orchestrator kirim **1 message parallel** `task:frontend (T09+T10)` + `task:backend (T02+T03)`. Jangan serialize.
- `orval` — frontend stub `msw`/`ky mock` D1, swap ke `generated/api.ts` real D2 setelah `openapi.yaml` ready. `pnpm gen` di `apps/web/package.json`.
- `tester` T13 unit tanpa DB bisa bareng T03; integration butuh T04 selesai.

**Handoff Orchestrator (template `task` desc):**
```
Backend T03: "TECH §5 Calendar Engine — internal/availability/service.go GetSlots handles buffer+DST+override+skill, cache 30s, UTC→Asia/Jakarta. TDD dulu, 100% coverage. Pakai pgx pooler 6543, jangan database/sql. Handoff ke tester T13."

Frontend T11: "DESIGN §6 + PRD §5 — app/(app)/app/page.tsx 5 row OKLCH violet 260, Recharts Area 10 bulan, tabular-nums. Data dari GET /dashboard via orval. Mobile 2x2, <800ms."
```

---

## 5. Risiko & Mitigasi Free Tier

| Risiko | Dampak | Mitigasi | Watch File |
|---|---|---|---|
| **Pooler 6543 habis** | E2E flaky | `pgxpool MaxConns 20`, `workers:1` CI, `TRUNCATE` bukan container per test | `cmd/api/main.go: pgxpool.New` |
| **EXCLUDE typo `status`** | double-book | `repo_test.go` assert `23P01`, `WHERE IN ('PENDING','CONFIRMED')` | `migrations/0001_init.up.sql` |
| **Supabase pause 7 hari** | QA hilang demo | `keep-warm.yml` daily `SELECT 1` + Vercel Cron `*/5` `/api/ping` | `vercel.json`, `.github/workflows/keep-warm.yml` |
| **Koyeb cold start 250ms** | first req 1s | `/health` + retry 2 di CI, ping tiap 5m jam demo | `internal/health/handler.go` |
| **Stripe webhook retry** | double charge | `stripeEventId UNIQUE` idempotent | `internal/payments/handler.go` |
| **Chart 10 bulan berat** | >800ms | Aggregate di DB `DATE_TRUNC`, bukan JS, `Recharts` 1 warna | `dashboard/sql/queries` |

---

## 6. Definition of Done & Reviewer Gate

**DoD tiap PR:**
- [ ] `go test ./... -cover` >80% (availability 100%) + `pnpm vitest --coverage` >70% + `pnpm playwright test` 8/8 green (Chromium)
- [ ] `gitleaks detect --no-banner --redact` pass, `.env` tidak staged, `JWT_SECRET` placeholder saja
- [ ] Design tokens OKLCH violet 260 exact, no `#000` pure, no shadow di dark, `getByRole` selector, `axe` 0 violation
- [ ] `EXCLUDE` + `tstzrange` ada di `git diff`, `UTC` storage verified

**Reviewer (`task:reviewer` read-only → tapi sekarang "*": allow, trust prompt):**
- `LGTM` → merge ke `main` → `smoke.yml` wait Vercel preview → 1 smoke test prod
- `NEEDS CHANGES` (HIGH/MEDIUM/LOW `file:line`) → route balik `frontend`/`backend` → re-run `tester` → re-review. Jangan bypass.

---

## 7. Handoff ke Orchestrator (What to Watch)

**Sudah direncanakan:** 14 file-level tasks, M0-M3, contract-first (`T01` unblock), frontend/backend parallel, Opsi A TRUNCATE, EXCLUDE, pooler 6543, 5 row dashboard, 1.500 seed.

**Keputusan dibuat:**
- `orval 7 + ky 1.8 + TanStack 5.80` untuk contract (bukan Server Actions mutasi)
- `gorilla/websocket` native (bukan Pusher) — Koyeb support WS
- `testcontainers 0.33+` untuk integration (jangan mock EXCLUDE)
- `pnpm 9 + turbo` tanpa Bun (locked)

**Yang diawasi:**
- **EXCLUDE constraint** — paling kritis; jika hilang, demo double-book = gagal.
- **Pooler 6543 transaction mode** — pastikan `pgxpool` pakai `?pgbouncer=true`, jangan `database/sql`.
- **Opsi A TRUNCATE** — jangan ganti ke transaction rollback (tidak work multi-conn) atau per-test container (2-3s).
- **OKLCH violet 260** — jangan `hue` random; rebrand cukup ganti `hue` token.
- **Stripe 4242** — mock di E2E, real `sk_test` hanya smoke manual.

**Next Action (orchestrator):**
1. `Write` markdown ini ke `docs/PLAN.md` ✅ DONE
2. `task:backend` T01 + T07 (D1 pagi) — tunggu merge ✅ T01 DONE
3. Parallel `task:backend` T02,T03 + `task:frontend` T09 (D1 sore)
4. Seq T04→T05→T06 + T10→T11→T12, lalu `task:tester` T13→T14, akhiri `task:reviewer`

---

## 8. Lampiran Env & Commands

```bash
# .env.example (jangan commit .env)
DATABASE_URL=postgres://postgres.[ref]:[pass]@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?pgbouncer=true
JWT_SECRET=[openssl rand -hex 32]
REFRESH_SECRET=[openssl rand -hex 32]
TEST_SECRET=[random-32-for-qa-only]
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
RESEND_API_KEY=re_...
OPENAI_API_KEY=sk-...
PORT=8080
APP_ENV=production # test untuk QA

# Lokal dev
pnpm install
DATABASE_URL=... go run ./apps/api/cmd/api/main.go # :8080
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1 pnpm --filter web dev
go run ./apps/api/cmd/seed/main.go        # full 1.500 Nov-Agu
go run ./apps/api/cmd/seed/main.go --minimal # 10 rows E2E
pnpm go:test -- -cover; pnpm web:test -- --coverage; pnpm web:e2e
migrate -path apps/api/migrations -database "$DATABASE_URL" up
```

```
# apps/web vercel.json
{ "crons": [{ "path": "/api/ping", "schedule": "*/5 * * * *" }] }

# playwright.config.ts workers:1 di CI
workers: process.env.CI ? 1 : undefined
```

**White-label:** Ganti seed FlowBarber → klinik/studio/konsultan cukup ganti `services` seed (durasi/harga/warna) tanpa ubah engine.
