# Security Policy — FlowBook (Public Repo)

> **Repo ini public** — jangan pernah commit secret.

## Apa yang tidak boleh di-commit

- `.env`, `.env.local`, `apps/*/.env` — semua sudah di `.gitignore`
- `DATABASE_URL`, `JWT_SECRET`, `STRIPE_SECRET_KEY`, `RESEND_API_KEY`, `OPENAI_API_KEY`
- Gunakan `.env.example` sebagai template — isi dari dashboard lokal, bukan dari repo

## Guard yang aktif

1. **`.gitignore` ketat** — `.env` dan `apps/*/.env` di-ignore, hanya `.env.example` yang boleh
2. **Husky pre-commit** — block jika `.env` ter-staged, dan jalankan `gitleaks protect --staged` jika ter-install
3. **GitHub Action `gitleaks`** — scan setiap push/PR (`.github/workflows/gitleaks.yml`)
4. **Push protection** — aktifkan di GitHub Settings → Code security → Secret scanning → Push protection (wajib centang)

## Jika terlanjur commit secret

1. **Revoke segera** di dashboard provider (Supabase, Stripe, Koyeb, Vercel)
2. Jangan cuma `git revert` — history masih ada. Pakai `git filter-repo` atau BFG, lalu force push
3. Rotate semua secret terkait

## Cara setup lokal (aman)

```bash
cp .env.example .env.local
cp apps/web/.env.example apps/web/.env.local
cp apps/api/.env.example apps/api/.env
# isi .env.local dari dashboard, jangan dari chat/log
```

Install gitleaks lokal (opsional tapi direkomendasikan):

```bash
go install github.com/gitleaks/gitleaks/v8@latest
gitleaks detect --source . --no-banner --redact
```

## Kontak

Laporkan temuan secret via private issue, bukan public comment.
