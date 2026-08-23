# TECH — Technical Architecture FlowBook

> Stack 2026 Stable Tanpa Bun | Next.js Vercel Hobby + Go Koyeb Eco + Supabase Free

## 1. Arsitektur High-Level

```
[Browser]
   |
   v
[Vercel: apps/web Next.js 15.3.5] -- ky + TanStack --REST /api/v1 + WS--> [Koyeb: apps/api Go Echo :8080]
   |  Node 20, pnpm 9, App Router                                         | Echo 4.13, pgx 5.7, sqlc 2.16
   |  SSR landing, client booking                                          | JWT, Stripe, Resend, OpenAI SSE
   |                                                                       | gorilla/websocket
   v                                                                       v
                                                            [Supabase Postgres 15]
                                                             pooler 6543, 500MB, pgvector
                                                             storage 1GB
```

**Monorepo:**
```
/ (pnpm-workspace.yaml, turbo.json)
  apps/web      -> Vercel root apps/web
  apps/api      -> Koyeb root apps/api (PORT=8080, /health)
  packages/shared-types (zod schemas, shared DTO)
  supabase/ -> symlink apps/api/migrations
```

- Vercel optimal untuk Next.js (ISR, Image), Koyeb optimal untuk Go (scale-to-zero, WS native)
- Web tidak pakai Server Actions untuk mutasi — semua via Go REST (JWT header `Authorization: Bearer`)

## 2. Versi Terkunci (Stable, Bukan Beta)

```
Web (Vercel Hobby)
  next 15.3.5, react 19.0, react-dom 19.0, typescript 5.6
  tailwindcss 4.0 (Oxide), shadcn latest, tw-animate-css, oklch tokens
  @tanstack/react-query 5.80, zustand 5, react-hook-form, zod 3.23
  orval 7, ky 1.8, lucide-react, framer-motion 12, recharts, next-themes
  pnpm 9, turbo, vitest, @testing-library/react, playwright 1.45

API (Koyeb Eco)
  go 1.23.8, echo 4.13, pgx/v5 5.7, sqlc 2.16, golang-migrate 4.18
  jwt/v5 5.2, validator/v10 10.20, slog stdlib, gorilla/websocket 1.5
  stripe-go 76, openai-go, joho/godotenv
  testify 1.9, testcontainers-go/modules/postgres 0.33

DB
  Supabase Postgres 15, pgvector, pooler 6543 transaction mode
```

Yang sengaja tidak dipakai: Next 16 canary, PPR, React Compiler, Bun runtime, Go 1.24 nightly, ent/Bun ORM.

## 3. Struktur Folder

```
apps/web/
  app/
    (marketing)/page.tsx           // landing
    (booking)/book/page.tsx        // 4-step
    (booking)/book/success/page.tsx
    (booking)/track/[id]/page.tsx
    (app)/app/page.tsx             // dashboard
    (app)/app/calendar/page.tsx
    (app)/app/bookings/page.tsx
    (app)/app/bookings/[id]/page.tsx
    (app)/app/services/page.tsx
    (app)/app/staff/page.tsx
    (app)/app/customers/page.tsx
    (app)/app/settings/page.tsx
    api/ping/route.ts              // cron ping Koyeb
    layout.tsx, globals.css
  components/ui/*                  // shadcn
  components/booking/*
  components/dashboard/*
  lib/api.ts, lib/queryKeys.ts
  generated/api.ts                 // orval
  e2e/                             // playwright
  vitest.config.ts, playwright.config.ts

apps/api/
  cmd/api/main.go                  // Echo init, pgxpool, migrate
  cmd/seed/main.go                 // seed Nov 2025-Agu 2026
  internal/
    config/
    middleware/ (jwt, cors, requestID, role)
    auth/ (handler, service, repo)
    availability/ (service engine, handler)
    bookings/ (handler, service, repo)
    payments/ (stripe handler, webhook)
    dashboard/ (aggregate query)
    health/
  migrations/ 0001_init.up.sql
  sql/queries/*.sql                // sqlc
  sqlc.yaml
  go.mod, Dockerfile (optional)
```

## 4. Data Model (Postgres)

```sql
-- organizations
CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL, slug TEXT UNIQUE NOT NULL,
  timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
  logo_url TEXT, created_at TIMESTAMPTZ DEFAULT now()
);
-- users
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID REFERENCES organizations(id),
  email TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL,
  name TEXT NOT NULL, role TEXT CHECK (role IN ('OWNER','STAFF','CUSTOMER')),
  created_at TIMESTAMPTZ DEFAULT now()
);
-- services, staff, availability, overrides, bookings, payments, refresh_tokens
-- bookings EXCLUDE anti double-book:
ALTER TABLE bookings ADD CONSTRAINT no_overlap
  EXCLUDE USING gist (staff_id WITH =, tstzrange(start_at, end_at) WITH &&)
  WHERE (status IN ('PENDING','CONFIRMED'));
CREATE INDEX ON bookings USING gist (tstzrange(start_at, end_at));
```

Simpan UTC di DB, render di `organization.timezone`. `pgx` type `timestamptz` native.

`sqlc` generate `db/models.go`, `db/queries/` dari `sql/queries/*.sql`.

## 5. Calendar Engine (Inti)

`internal/availability/service.go`:

1. Load `availability` mingguan + `availability_overrides` untuk tanggal target
2. Generate grid 15m dalam `org timezone`
3. Query `SELECT ... WHERE tstzrange(start_at,end_at) && $range` untuk clash
4. Filter slot muat `duration+buffer` tanpa overlap
5. Cache 30s memory

Unit test table-driven: DST, buffer, overnight, override libur.

## 6. API Contracts

Base: `https://flowbook-api-xxx.koyeb.app/api/v1`

```
POST   /auth/register, /auth/login, /auth/refresh, /auth/logout
GET    /services
POST   /services               (OWNER)    GET /staff  POST /staff (OWNER)
GET    /availability/slots?serviceId&staffId&date&tz
POST   /bookings               GET /bookings?from&to&status&staffId
GET    /bookings/:id           POST /bookings/:id/cancel  POST /bookings/:id/reschedule
POST   /payments/checkout-session  POST /payments/webhook (Stripe, verify signature)
GET    /dashboard?from&to&granularity&tz
GET    /health                 GET /ws (upgrade)
# test only (APP_ENV=test)
POST   /test/reset             POST /test/seed-full
GET    /openapi.yaml
```

Auth: `Authorization: Bearer <access 15m>`, refresh via httpOnly `refresh_token` cookie `7d`. CORS `allow: https://flowbook-xxx.vercel.app`.

Validation: `validator/v10` di Echo `BindAndValidate`, error 422 Zod-compatible. `orval` generate TS client di web.

## 7. Auth & RBAC

- Register hash `bcrypt`, login issue JWT pair.
- Middleware `JWTMiddleware` + `RequireRole(OWNER, STAFF)` di Go. Next `middleware.ts` redirect `/app/*` if 401.
- `refresh_tokens` table: `{id, user_id, token_hash, expires_at, revoked}`

## 8. Realtime

Go `gorilla/websocket` Hub: `hub.Broadcast(orgID, {type:"slot_taken", slot})`. Koyeb support WS native, no Pusher. Next `useWebSocket` subscribe `ws://api/ws?orgId=...` → invalidate TanStack `slots`.

## 9. Stripe & Email

- `stripe-go`: `CreateCheckoutSession` → redirect, `HandleWebhook` idempotent via `stripeEventId` unique.
- `Resend`: templates React Email `BookingConfirmed`, `Reminder H-1` (Vercel Cron `GET /api/ping` + Go cron tiap 15m), `Cancelled`.

## 10. AI Receptionist

`POST /chat` SSE `text/event-stream` proxy ke OpenAI/Mistral, tools:
- `checkAvailability({service,date})` → `availability.Service.GetSlots`
- `createBooking({service,staff,slot,customer})` → `bookings.Service.Create`

Vercel AI SDK tidak dipakai di Next (karena Go), SSE manual di Echo.

## 11. Env

```
# web .env (Vercel)
NEXT_PUBLIC_API_URL=https://flowbook-api-xxx.koyeb.app/api/v1
NEXT_PUBLIC_SUPABASE_URL=... (hanya jika pakai storage direct)
NEXT_PUBLIC_SUPABASE_ANON_KEY=...

# api .env (Koyeb Secrets)
DATABASE_URL=postgres://postgres.xxx:5432?pgbouncer=true  # pooler 6543
JWT_SECRET=openssl rand -hex 32
REFRESH_SECRET=...
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
RESEND_API_KEY=re_...
OPENAI_API_KEY=sk-...
PORT=8080
APP_ENV=production
TEST_SECRET=... # hanya untuk /test/* di QA
```

CORS: `AllowOrigins: [https://flowbook-xxx.vercel.app, http://localhost:3000]`.

## 12. Seed

`go run ./cmd/seed` loop 2025-11-01 → 2026-08-24, ~1.500 bookings, bobot weekend/musiman, 60 customers. Dipakai dashboard + `POST /test/seed-full`.

## 13. Observability

- Go `slog` JSON → Koyeb log exporter
- Metrics: `/metrics` optional (Prometheus) — tidak wajib free
- Health: `GET /health` → `{"status":"ok","db":"up"}` untuk Koyeb health check

## 14. Keputusan Arsitektur (ADR Ringkas)

- **Bun ditolak** → pnpm + Node 20 GA, Bun runtime masih beta (no source map).
- **Echo dipilih** → WS native, validator built-in, hireable; vs Gin/Chi benchmark beda tipis.
- **sqlc+pgx dipilih** → type-safe, EXCLUDE terlihat, vs GORM hidden query.
- **Vercel+Koyeb split** → Next ISR di Vercel paling cepat, Go WS di Koyeb paling hemat.
- **TRUNCATE Opsi A untuk E2E** → 150ms, workers 1 di CI, vs transaction tidak work multi-conn.
