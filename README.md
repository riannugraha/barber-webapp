# FlowBook — Booking & Scheduling Platform (Demo)

> Full-stack demo untuk impress client freelance — **Next.js Vercel Hobby + Go Koyeb Eco + Supabase Free | Tanpa Bun | pnpm + turbo**

![Stack](https://img.shields.io/badge/Next.js-15.3.5-black) ![Go](https://img.shields.io/badge/Go-1.23-blue) ![Tailwind](https://img.shields.io/badge/Tailwind-4.0-06B6D4) ![Supabase](https://img.shields.io/badge/Supabase-Free-3ECF8E)

Demo barbershop yang bisa di-white-label ke klinik, studio, konsultan, rental dalam 5 menit. Tampilkan dalam 2 menit: booking → Stripe test (`4242`) → dashboard revenue 10 bulan.

## ✨ Highlights

- **Calendar engine** timezone-aware + buffer + `EXCLUDE` anti double-book
- **Realtime slot** via Go WebSocket (Koyeb native)
- **Stripe Test** + Resend email + AI Receptionist SSE
- **Dashboard minimalis** — 4 KPI + Area Chart Nov 2025→Agu 2026 (~1.500 bookings seed)
- **Light/Dark OKLCH** — `next-themes`, violet `260`
- **TDD + E2E Opsi A** — `TRUNCATE + seedMinimal` per `beforeEach`, Supabase QA terpisah, 8 critical Playwright tests

## 📚 Dokumen

- [PRD](docs/PRD.md) — persona, layanan 8, journey, halaman
- [TECH](docs/TECH.md) — arsitektur, versi terkunci, data model, API
- [DESIGN](docs/DESIGN.md) — design system, tokens, inventory UI, layout 5 row
- [TESTING](docs/TESTING.md) — piramida, Go testcontainers, Vitest, Playwright Opsi A
- [DEPLOYMENT](docs/DEPLOYMENT.md) — Vercel + Koyeb + Supabase free step + cron keep-warm

## 🚀 Quick Start (Lokal)

```bash
pnpm install
# 1. Supabase: buat project QA, copy pooler 6543 DATABASE_URL
# 2. Go API
DATABASE_URL=postgres://...:6543/postgres?pgbouncer=true go run ./apps/api/cmd/api/main.go # :8080
# 3. Web
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1 pnpm --filter web dev # :3000
# atau turbo
pnpm dev

# Seed
go run ./apps/api/cmd/seed/main.go              # full 1.5k Nov-Agu
go run ./apps/api/cmd/seed/main.go --minimal    # 10 rows untuk E2E
```

## 🧪 Test

```bash
pnpm go:test -- -cover          # Go unit + integration (testcontainers)
pnpm web:test -- --coverage     # Vitest
pnpm web:e2e                    # Playwright chromium
supabase test db                # pgTAP (optional)
```

## 🌗 Stack Terkunci 2026 Stable

`next 15.3.5 / react 19 / tailwind 4 / shadcn / tanstack-query 5.80 / zustand 5` + `go 1.23.8 / echo 4.13 / pgx 5.7 / sqlc 2.16` + `Supabase 15`

Tanpa Bun — `pnpm 9` + Node 20 GA. Lihat [TECH.md](docs/TECH.md).

## 🔒 Public Repo — Secret Handling

> Repo ini public — **jangan commit `.env`**. Semua secret via dashboard.

```bash
cp .env.example .env.local
cp apps/web/.env.example apps/web/.env.local
cp apps/api/.env.example apps/api/.env
# isi dari Vercel/Koyeb/Supabase dashboard, bukan dari repo
```
- `.env` & `apps/*/.env` sudah di `.gitignore` — hanya `.env.example` yang boleh
- Pre-commit hook block jika `.env` ter-staged + `gitleaks protect --staged`
- GitHub Action `gitleaks` scan tiap push/PR — aktifkan **Push protection** di Settings → Code security
- Jika terlanjur push secret: revoke di dashboard, `git filter-repo`, force push

Lihat [SECURITY.md](SECURITY.md) dan [DEPLOYMENT.md](docs/DEPLOYMENT.md).

## 📦 Deploy Free

Vercel `apps/web` + Koyeb `apps/api` (`PORT=8080 /health`) + Supabase pooler. Lihat [DEPLOYMENT.md](docs/DEPLOYMENT.md).

## 📄 Lisensi

Demo portfolio — bebas fork untuk pitch client.
