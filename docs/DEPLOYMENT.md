# DEPLOYMENT — Free Tier Guide (Vercel + Koyeb + Supabase)

> Vercel Hobby + Koyeb Eco Free + Supabase Free | pnpm, tanpa Bun

## 1. Gambaran

```
GitHub -> Vercel (apps/web) -> preview + prod
       -> Koyeb  (apps/api)  -> .koyeb.app
       -> Supabase (DB + Storage)
```

2 project Supabase Free: `flowbook-qa` (E2E + preview) dan `flowbook-prod` (prod). Vercel env split: Preview → QA, Production → Prod.

## 2. Prasyarat

- Node 20, pnpm 9, Go 1.23, Docker (untuk testcontainers lokal)
- Akun Vercel, Koyeb, Supabase, Stripe (test), Resend, OpenAI

## 3. Supabase Setup

### 3.1 Buat 2 Project

1. supabase.com → New Project `flowbook-qa` region SG, save DB password
2. New Project `flowbook-prod` region SG

Untuk tiap project, ambil dari **Settings → API Keys** (new keys) — tab **Publishable and secret** (legacy di tab **Legacy**):
- `Project URL` (`https://[ref].supabase.co`)
- `Publishable key` (`sb_publishable_...`, legacy: `anon` `eyJ...`)
- `Secret key` (`sb_secret_...`, legacy: `service_role` `eyJ...`, bypasses RLS — server only)

Dan dari **Database → Connection string** (pooler):
```
postgres://postgres.[ref]:[pass]@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?pgbouncer=true
```

### 3.2 Migrasi

```bash
# lokal
go run ./cmd/migrate up
# atau via Goose/golang-migrate CLI
migrate -path apps/api/migrations -database "$DATABASE_URL" up
```

### 3.3 Free Tier Note

- 500MB DB, 1GB storage, pause 7 hari idle
- Mitigasi pause: cron `SELECT 1` tiap hari (lihat §6)

## 4. Koyeb — Go API

### 4.1 Deploy via Dashboard

1. app.koyeb.com → Create Service → GitHub → repo `rian/demo-fullstack` → Branch `main`
2. **Root directory:** `apps/api`
3. **Builder:** Buildpack (deteksi go.mod) — atau Dockerfile
4. **Port:** `8080:http`, Route `/:8080`
5. **Instance:** Eco Free (0.1 vCPU / 512MB) untuk QA, Small untuk prod jika butuh
6. **Health check:** `GET /health` → 200

### 4.2 Env (Koyeb Secrets)

```
DATABASE_URL=postgres://...:6543/postgres?pgbouncer=true
JWT_SECRET=openssl rand -hex 32
REFRESH_SECRET=openssl rand -hex 32
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
RESEND_API_KEY=re_...
OPENAI_API_KEY=sk-...
PORT=8080
APP_ENV=production
TEST_SECRET=... # hanya QA
```

### 4.3 Koyeb CLI Alt

```bash
koyeb app init flowbook --git github.com/rian/demo-fullstack --git-branch main --ports 8080:http --routes /:8080 --env PORT=8080
koyeb service logs flowbook/flowbook -t build
```

Cold start Eco ~250ms — first request setelah idle lambat 1s.

## 5. Vercel — Next.js Web

### 5.1 Import

1. vercel.com → Add New Project → Import GitHub `demo-fullstack`
2. **Root Directory:** `apps/web`
3. Framework: Next.js, Build `pnpm run build`, Install `pnpm install`
4. Node 20

### 5.2 Env (Vercel)

**Preview (QA):**
```
NEXT_PUBLIC_API_URL=https://flowbook-api-qa-xxx.koyeb.app/api/v1
NEXT_PUBLIC_SUPABASE_URL=https://xxx-qa.supabase.co
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY=sb_publishable_... # legacy: NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJ...
```
**Production:**
```
NEXT_PUBLIC_API_URL=https://flowbook-api-xxx.koyeb.app/api/v1
NEXT_PUBLIC_SUPABASE_URL=https://xxx-prod.supabase.co
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY=sb_publishable_... # legacy: NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJ...
```

### 5.3 Deployment Protection Bypass (untuk Playwright CI)

Jika Vercel Authentication ON, buat bypass secret:

```bash
openssl rand -hex 16
# Tambah ke Vercel Env `VERCEL_AUTOMATION_BYPASS_SECRET` (32 char) — Preview + Production
# Di playwright.config.ts, header `x-vercel-protection-bypass: $VERCEL_AUTOMATION_BYPASS_SECRET`
```

Atau matikan protection untuk project demo.

## 6. Cron Keep-Warm

### Vercel Cron — ping Koyeb

`apps/web/app/api/ping/route.ts` → fetch `GET $API_URL/health` tiap 5m saat jam demo:

```json
// vercel.json
{ "crons": [{ "path": "/api/ping", "schedule": "*/5 * * * *" }] }
```

### Supabase anti-pause

GitHub Actions daily `SELECT 1`:

```yaml
# .github/workflows/keep-warm.yml
on: { schedule: [{ cron: "0 9 * * *" }] }
jobs:
  ping:
    runs-on: ubuntu-latest
    steps:
      - run: psql "$DATABASE_URL" -c "SELECT 1"
        env: { DATABASE_URL: ${{ secrets.QA_DATABASE_URL }} }
```

## 7. Stripe Test

1. dashboard.stripe.com → Developers → API keys → `sk_test` + `pk_test`
2. Vercel/Koyeb env `pk_test` di web, `sk_test` di api
3. Lokal webhook:
```bash
stripe listen --forward-to localhost:8080/api/v1/payments/webhook
stripe trigger checkout.session.completed
```
4. Kartu test: `4242 4242 4242 4242` exp `12/34` CVC `123`

## 8. Lokal Dev

```bash
pnpm install
# terminal 1: Go
DATABASE_URL=postgres://...:6543/postgres?pgbouncer=true go run ./apps/api/cmd/api/main.go
# terminal 2: Web
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1 pnpm --filter web dev
# atau turbo
pnpm dev
```

Seed:

```bash
go run ./apps/api/cmd/seed/main.go        # full 1.500 Nov-Agu
go run ./apps/api/cmd/seed/main.go --minimal # 10 rows untuk E2E
```

## 9. Batasan Free Tier Recap

| Layanan | Limit Free | Mitigasi |
|---|---|---|
| Vercel Hobby | 100GB BW, 100k func, 10s timeout | Cukup, function jangan berat |
| Koyeb Eco | 0.1 vCPU, 512MB, scale-to-zero | ping cron, /health |
| Supabase Free | 500MB DB, pause 7d, 50k MAU | 2 project, daily ping, <500MB |

## 10. Upgrade Path

Client bayar → Vercel Pro $20 + Koyeb Small $29 (0.5 vCPU) + Supabase Pro $25 — tanpa rewrite, ganti instance size.
