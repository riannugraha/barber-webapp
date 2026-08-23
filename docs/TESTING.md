# TESTING — TDD & E2E Playwright (Opsi A)

> Stack Tanpa Bun | Opsi A: TRUNCATE + seedMinimal per beforeEach + Supabase QA terpisah

## 1. Piramida Testing

| Layer | Tool | Scope | Speed | Kapan |
|---|---|---|---|---|
| Unit Go | `go test + testify/suite` | availability engine, service logic tanpa DB | 0.2s | tiap push |
| Integration Go | `testcontainers-go/modules/postgres:16-alpine` + `pgx` + `golang-migrate` | sqlc queries, EXCLUDE constraint, repo | 4-6s | tiap PR |
| Unit Web | `Vitest + Testing Library` | CalendarSlots, BookingForm, formatSlot | 1s | tiap PR |
| E2E | `Playwright 1.45` Chromium | 8 critical flows, auto-wait, trace | 30-60s | lokal + CI preview |

Bukan Jest/Cypress — 2026 starter Next 15 semua ke Vitest + Playwright.

## 2. Go TDD — Red-Green-Refactor

### Struktur

```
apps/api/
  internal/availability/service_test.go      // unit murni
  internal/bookings/service_test.go
  internal/bookings/repo_test.go             // integration
  internal/payments/webhook_test.go
  testhelpers/postgres.go                    // reusable container
```

### Unit — Availability Engine (Inti)

Table-driven tanpa DB:

```go
func TestGetSlots(t *testing.T) {
  cases := []struct{name string; avail []Availability; bookings []Booking; want int}{
    {"buffer 10m blocks next", avail30m, bookings10_00, 3},
    {"DST Asia/Jakarta 2025-11-02", ..., 4},
    {"override libur", ..., 0},
    {"Hair Color hanya Bayu", ..., 2},
  }
  for _, c := range cases { t.Run(c.name, func(t *testing.T) {
    slots := svc.GetSlots(ctx, c.avail, c.bookings, "2025-11-15", "Asia/Jakarta")
    assert.Len(t, slots, c.want)
  })}
}
```

Target: DST, buffer, overnight, override, staff skill — harus TDD sebelum code.

### Integration — testcontainers

```go
// testhelpers/postgres.go
func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
  ctr, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("flowbook_test"),
    postgres.WithUsername("test"),
    postgres.WithPassword("test"),
    testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
  )
  // ConnectionString + migrate up
}

// repo_test.go — 1 container per TestMain, reuse
func TestMain(m *testing.M) {
  ctx := context.Background()
  ctr, _ := testhelpers.CreatePostgresContainer(ctx)
  connStr, _ := ctr.ConnectionString(ctx)
  pool, _ := pgxpool.New(ctx, connStr)
  migrate.Up(pool) // golang-migrate
  testPool = pool
  code := m.Run()
  ctr.Terminate(ctx)
  os.Exit(code)
}

func TestCreateBooking_ExcludeOverlap(t *testing.T) {
  // insert 10:00-10:30
  _, err := queries.CreateBooking(ctx, params1)
  require.NoError(t, err)
  // coba 10:15-10:45 same staff -> expect 23P01
  _, err = queries.CreateBooking(ctx, paramsOverlap)
  assert.ErrorContains(t, err, "exclusion")
}

func TestCreateBooking_TransactionRollback(t *testing.T) {
  tx, _ := testPool.Begin(ctx)
  defer tx.Rollback(ctx)
  // test dengan tx, tidak pollute DB lain
}
```

Lib: `testcontainers-go 0.33+`, `testify 1.9`, `pgx/v5`. Tidak mock — mock bohongi EXCLUDE.

### Handler

```go
func TestBookingsHandler_Auth(t *testing.T) {
  e := echo.New()
  req := httptest.NewRequest("POST", "/api/v1/bookings", body)
  req.Header.Set("Authorization", "Bearer "+ownerToken)
  rec := httptest.NewRecorder()
  e.ServeHTTP(rec, req)
  assert.Equal(t, 201, rec.Code)
}
```

## 3. Next.js Unit — Vitest

`apps/web/vitest.config.ts`:

```ts
export default defineConfig({
  test: { environment: "jsdom", setupFiles: "./vitest.setup.ts" },
  plugins: [tsconfigPaths()],
})
```

```tsx
// CalendarSlots.test.tsx
test("renders available slots", async () => {
  vi.mocked(ky.get).mockResolvedValue({ slots: [{ start:"10:00", available:true }] })
  render(<CalendarSlots date="2025-11-15" />)
  expect(await screen.findByText("10:00")).toBeVisible()
})
```

## 4. Playwright E2E — Setup

`apps/web/playwright.config.ts` (Next 15 official 2026):

```ts
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined, // Supabase pooler 60 conns, Koyeb 0.1 vCPU
  reporter: "html",
  use: { baseURL: "http://localhost:3000", trace: "on-first-retry", screenshot: "only-on-failure" },
  webServer: {
    command: "pnpm --filter web dev",
    url: "http://localhost:3000",
    reuseExistingServer: !process.env.CI,
    timeout: 120*1000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
})
```

Hanya Chromium di MVP.

### Fixtures — Auth & Isolation (Opsi A)

```ts
// e2e/fixtures.ts
import { test as base } from "@playwright/test"

export const test = base.extend<{ ownerToken: string }>({
  ownerToken: async ({ request }, use) => {
    const res = await request.post(`${API_URL}/auth/login`, { data: { email:"owner@flowbook.test", password:"ownerpass" } })
    const { accessToken } = await res.json()
    await use(accessToken)
  }
})

test.beforeEach(async ({ request }) => {
  // Opsi A: TRUNCATE + seedMinimal(10)
  await request.post(`${API_URL}/test/reset`, {
    headers: { "x-test-secret": process.env.TEST_SECRET! }
  })
})
```

Go helper (hanya `APP_ENV=test`):

```go
r.POST("/test/reset", func(c echo.Context) error {
  if c.Request().Header.Get("x-test-secret") != os.Getenv("TEST_SECRET") { return echo.ErrUnauthorized }
  db.Exec(ctx, "TRUNCATE bookings,payments,customers RESTART IDENTITY CASCADE")
  seedMinimal(db) // 3 staff, 8 layanan, 0 booking
  return c.NoContent(204)
}, testOnly)

r.POST("/test/seed-full", func(c echo.Context) error {
  seedFull(db) // 1.500 rows Nov-Agu, hanya untuk test chart
  return c.NoContent(204)
})
```

- **TRUNCATE 150-250ms**, vs transaction rollback tidak work multi-conn, vs per-test container 2-3s
- **Project terpisah:** `flowbook-qa` untuk E2E, `flowbook-prod` untuk prod — 2 project free tier. Jangan hit prod.

### Page Objects

```
e2e/
  fixtures.ts
  pages/BookPage.ts
  pages/DashboardPage.ts
  01-public-booking.spec.ts
  02-slot-realtime.spec.ts
  ...
```

```ts
// pages/BookPage.ts
export class BookPage {
  constructor(private page: Page) {}
  async selectService(name: string) { await this.page.getByRole("button", { name }).click() }
  async selectSlot(time: string) { await this.page.getByRole("button", { name: time }).click() }
}
```
Selector pakai `getByRole`, bukan css.

### 8 Critical E2E Tests

| File | Steps | Assert |
|---|---|---|
| `01-public-booking.spec.ts` | /book pilih Classic Cut → Any → slot besok → form → Stripe mock → success | Booking CONFIRMED + email |
| `02-slot-realtime.spec.ts` | 2 context: A book 10:00, B lihat slot hilang | slot disabled |
| `03-overlap-prevent.spec.ts` | booking 10:00, coba 10:15 same staff | 409 exclusion |
| `04-owner-dashboard.spec.ts` | login OWNER → /app | KPI + chart 10 bulan render |
| `05-staff-restrict.spec.ts` | login STAFF → POST /services | 403 |
| `06-cancel-reschedule.spec.ts` | /track/[id] cancel | status CANCELLED di dashboard |
| `07-theme.spec.ts` | toggle dark → reload | html.dark persist |
| `08-a11y.spec.ts` | @axe-core/playwright di / dan /app | 0 violation |

Mock Stripe di E2E: `page.route('**/payments/**', route => route.fulfill({ json:{status:"PAID"}}))` — real Stripe hanya di smoke manual.

## 5. CI — GitHub Actions

**On PR (`ci.yml`):**
```yaml
lint & typecheck (pnpm turbo)
→ go test ./... -cover (testcontainers)
→ pnpm vitest --coverage
→ pnpm playwright test (local dev + Supabase QA)
# artifact: playwright-report 7 hari
```

**On merge main (`smoke.yml`):**
```
wait Vercel preview → playwright test 01- smoke vs PROD_URL (1 test)
```

Vercel Hobby + Koyeb Eco: Playwright jalan lokal di CI (`npx playwright install --with-deps chromium`), tidak butuh browserless.

## 6. Coverage

- Go `availability` 100%, overall >80%
- Web Vitest >70%
- E2E 8 critical green = demo aman

## 7. Verifikasi Lokal

```bash
pnpm go:test -- -cover
pnpm web:test -- --coverage
pnpm web:e2e
supabase test db  # pgTAP kalau RLS
```

## 8. Catatan Free Tier

- Supabase QA pause 7 hari → cron daily `SELECT 1`
- Koyeb cold start → test pertama tunggu 1s, retry 2 di CI
- Vercel preview protection → bypass header `x-vercel-protection-bypass` jika aktif
