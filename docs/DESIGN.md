# DESIGN — Design System & UI/UX FlowBook

> Tailwind 4 + shadcn/ui + Radix | Minimalis Fungsional | Light/Dark OKLCH

## 1. Prinsip Minimalis (Linear/Stripe/Vercel 2026)

- **Hierarchy via tipografi, bukan warna** — `text-3xl/600` KPI, `text-xs/500 muted` label
- **Spacing 4/8px + whitespace purposeful** — `p-6 gap-6`, grid `gap-4`
- **Progressive disclosure** — default 4 KPI + 1 chart, drill-down di Sheet/Dialog
- **No shadow di light, lightness di dark** — dark elevasi via surface lebih terang, bukan shadow
- **Micro-interaction halus** — `hover:bg-muted/50 transition-colors`, `animate-pulse` untuk slot pending

User paham "hari ini rame ga?" dalam **3 detik**. Sisanya drill-down.

## 2. Tokens — `apps/web/app/globals.css`

```css
@import "tailwindcss";
@import "tw-animate-css";

@theme {
  /* Light */
  --color-background: oklch(0.99 0 0);
  --color-foreground: oklch(0.18 0.02 260);
  --color-card: oklch(1 0 0);
  --color-card-foreground: oklch(0.18 0.02 260);
  --color-muted: oklch(0.96 0.01 260);
  --color-muted-foreground: oklch(0.55 0.02 260);
  --color-border: oklch(0.92 0.01 260);
  --color-primary: oklch(0.62 0.19 260); /* violet barber — ganti hue untuk rebrand */
  --color-primary-foreground: oklch(0.98 0 0);
  --color-destructive: oklch(0.58 0.22 25);
  --radius: 0.625rem;
  --font-sans: "Geist Sans", sans-serif;
}

.dark {
  /* Dark — off-black, bukan #000, + elevation */
  --color-background: oklch(0.14 0.01 260); /* Level 0 */
  --color-card: oklch(0.19 0.015 260);      /* Level 1 elevated */
  --color-card-foreground: oklch(0.96 0.01 260);
  --color-muted: oklch(0.22 0.015 260);
  --color-muted-foreground: oklch(0.68 0.015 260);
  --color-border: oklch(0.24 0.015 260);
  --color-primary: oklch(0.68 0.16 260);    /* +15% luminance, tidak neon */
}

@layer base {
  * { @apply border-border outline-ring/50; }
  body { @apply bg-background text-foreground antialiased; }
}
```

**Setup:**
```tsx
// app/layout.tsx
<html suppressHydrationWarning><body>
  <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
    {children}
  </ThemeProvider>
</body></html>
```
```tsx
// components/ThemeToggle.tsx
const { theme, setTheme } = useTheme();
<Button variant="ghost" size="icon" onClick={() => setTheme(theme==="dark"?"light":"dark")}>
  <Sun className="dark:-rotate-90 dark:scale-0" /><Moon className="rotate-90 scale-0 dark:rotate-0 dark:scale-100" />
</Button>
```

**Kenapa OKLCH:** perceptually uniform, light/dark pair tidak muddy, P3 ready — standar shadcn v4 Jan 2026.

## 3. Komponen shadcn

```
npx shadcn@latest add button card input form select calendar dialog sheet dropdown-menu avatar badge table tabs popover separator skeleton switch textarea alert accordion chart sonner
```

~22 komponen di `components/ui`, own code, edit bebas.

Tambahan: `Recharts` via `shadcn/chart`, `lucide-react`, `framer-motion`.

## 4. Layout Global

```
[Sidebar 16rem -> 3.5rem collapsible] | [Header: Command K + ThemeToggle + Avatar]
-------------------------------------------------
Content max-w-7xl mx-auto p-6 space-y-6
```

- Sidebar `shadcn Sidebar` block, mobile jadi `Sheet` drawer
- Header sticky `backdrop-blur`
- `Toaster` sonner bottom-right
- AI Widget floating bottom-right di `/book`

## 5. Inventory Halaman & Komponen

### Public (Customer)

| Route | UI | Komponen |
|---|---|---|
| `/` Landing | Hero + 3 layanan unggulan + pricing + FAQ Accordion + footer + CTA Book Now | Button, Card, Badge, Accordion |
| `/book` Step1 | Pilih layanan — grid Card durasi/harga | Card, RadioGroup, Skeleton |
| Step2 | Pilih staff — Avatar + "Any available" | Avatar, Tabs |
| Step3 | Kalender — slot realtime per tanggal, legend available/buffer/taken, hanya slot muat durasi+buffer aktif | Calendar, Popover, Toggle, Badge |
| Step4 | Form customer + notes + review | Form (rhf+zod), Input, Textarea |
| `/book/success` | Konfirmasi + ics download | Card, Button, Separator |
| `/track/[id]` | Detail booking + reschedule/cancel | Dialog, Alert, Calendar |
| `/login` `/register` | Auth form | Form, Input, Button |

### App (Protected)

| Route | UI | Komponen |
|---|---|---|
| `/app` Dashboard | KPI 4 cards + Area Chart 10 bulan + Pie/Bar/Heatmap + Top Customers + Recent (5 row) | Card, Chart, Table, Badge |
| `/app/calendar` | Week view 7 kolom 07-21, drag block, klik slot detail | Sheet, Popover, framer-motion drag |
| `/app/bookings` | DataTable filter status/date/staff + search + pagination | DataTable, Input, Select, DropdownMenu |
| `/app/bookings/[id]` | Detail + timeline + reschedule/cancel + payment badge | Sheet, Timeline, Badge |
| `/app/services` | CRUD layanan (nama, durasi, harga, buffer, warna, active) | Dialog, Form, Switch |
| `/app/staff` | List staff + availability editor mingguan + override tanggal | Avatar, Table, Dialog, Calendar |
| `/app/customers` | List customer + history | Table, Avatar |
| `/app/settings` | Org name, timezone, logo upload | Form, Select, Input file |

### Branding Awal

- Nama: FlowBarber Studio (ganti via seed)
- Warna: violet `260` — premium tech, ganti `hue` untuk emerald `160` atau amber `25`
- Radius `0.625rem`, shadow-none di light
- Typography: Geist Sans, tabular-nums untuk angka KPI

## 6. Dashboard Detail — 5 Row (Seed Nov 2025 → Agu 2026)

```
ROW 1: 4 KPI Cards
  [Revenue Nov-Agu] Rp 142jt | [Bookings] 1.542 | [Occupancy] 68% | [Avg Ticket] Rp 128k
  + Δ vs bulan lalu (+9%, +4%)

ROW 2: Area Chart Revenue per Bulan (10 titik) — 1 warna primary, grid halus, toggle Harian/Mingguan/Bulanan

ROW 3: 3 kolom
  [Pie Layanan 35% Classic Cut] [Bar Staff Andi 90/Bayu 70/Sari 20] [Heatmap Jam Sibuk 7x15]

ROW 4: 2 kolom
  [Top 15 Loyal — Siti 18x] [Recent 10 Bookings + status Badge]

ROW 5: Insight
  [Busiest Month: Des 2025] [Cancel Rate 7.2%] [Staff Utilization]
```

- Angka `font-mono tabular-nums`
- Klik KPI → filter `/app/bookings?from&to`
- Chart `Recharts` Area, `isAnimationActive`, tooltip `Rp`

## 7. State Kosong & Loading

- `Skeleton` untuk slot grid
- Empty: ilustrasi + "Belum ada booking hari ini — bagikan link /book" (bukan "No data")
- Error: `Alert` destructive + retry

## 8. Responsif & Aksesibilitas

- Breakpoint: KPI 4→2x2 di mobile, chart scroll horizontal, sidebar drawer
- Kontras 7:1 AAA, `getByRole` untuk test, `prefers-reduced-motion`
- Keyboard: DataTable row navigable, Dialog focus trap

## 9. Checklist QA Visual

- 3-second test: Owner jawab "rame ga?" 3 detik
- Light & dark tidak pure black, aksen tidak neon
- 320px → 4K tidak break, dark mode konsisten semua komponen
- Command K `⌘K` — Go to Bookings, Add Service, Toggle theme
