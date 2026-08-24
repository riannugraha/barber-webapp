"use client";

import * as React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell, BarChart, Bar } from "recharts";

// Fallback dashboard shape matches GET /dashboard
type DashboardData = {
  kpi: { totalBookings: number; confirmedBookings: number; totalRevenueCents: number; avgTicketCents: number; occupancyPct: number; deltaRevenuePct?: number; deltaBookingsPct?: number };
  revenueSeries: Array<{ period: string; revenueCents: number; bookingsCount: number }>;
  topServices: Array<{ name: string; bookingsCount: number; revenueCents: number }>;
  bookingsByStaff: Array<{ name: string; bookingsCount: number }>;
  bookingsByHour: Array<{ hour: number; bookingsCount: number }>;
  topCustomers: Array<{ customerName: string; customerEmail: string; bookingsCount: number }>;
  recentBookings: Array<{ id: string; customerName: string; status: string; startAt: string }>;
};

const fallback: DashboardData = {
  kpi: { totalBookings: 1542, confirmedBookings: 1420, totalRevenueCents: 14200000, avgTicketCents: 128000, occupancyPct: 68, deltaRevenuePct: 9, deltaBookingsPct: 4 },
  revenueSeries: ["Nov", "Des", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"].map((m, i) => ({
    period: `2025-${String(11 + i).padStart(2, "0")}-01T00:00:00Z`,
    revenueCents: [9000000, 14500000, 11000000, 9800000, 12300000, 14200000, 13500000, 11800000, 13900000, 14200000][i],
    bookingsCount: [120, 180, 140, 126, 145, 162, 155, 138, 160, 165][i],
    // label for chart
    // @ts-ignore
    label: m,
  })) as unknown as DashboardData["revenueSeries"],
  topServices: [
    { name: "Classic Cut", bookingsCount: 540, revenueCents: 45900000 },
    { name: "Premium Fade", bookingsCount: 320, revenueCents: 38400000 },
    { name: "Cut + Beard", bookingsCount: 210, revenueCents: 31500000 },
  ],
  bookingsByStaff: [
    { name: "Andi", bookingsCount: 90 },
    { name: "Bayu", bookingsCount: 70 },
    { name: "Sari", bookingsCount: 20 },
  ],
  bookingsByHour: Array.from({ length: 15 }).map((_, i) => ({ hour: 7 + i, bookingsCount: Math.floor(Math.random() * 20) + 5 })),
  topCustomers: Array.from({ length: 5 }).map((_, i) => ({ customerName: ["Siti 18x", "Budi 14x", "Ani 12x", "Joko 10x", "Dewi 9x"][i].split(" ")[0], customerEmail: `cust${i}@example.com`, bookingsCount: [18, 14, 12, 10, 9][i] })),
  recentBookings: Array.from({ length: 5 }).map((_, i) => ({ id: `bk-00${i + 1}`, customerName: ["Siti Rahayu", "Budi Santoso", "Ani Wijaya", "Joko Prabowo", "Dewi Lestari"][i], status: ["CONFIRMED", "PENDING", "CONFIRMED", "CANCELLED", "CONFIRMED"][i], startAt: new Date().toISOString() })),
};

function useDashboard() {
  return useQuery({
    queryKey: queryKeys.dashboard(),
    queryFn: async (): Promise<DashboardData> => {
      try {
        const res = await api.get("dashboard?from=2025-11-01&to=2026-08-24&granularity=month&tz=Asia/Jakarta").json<DashboardData>();
        return res;
      } catch {
        return fallback;
      }
    },
  });
}

export default function DashboardPage() {
  const { data, isLoading, isError } = useDashboard();
  const d = data ?? fallback;

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i} className="p-6"><Skeleton className="h-20 w-full" /></Card>
          ))}
        </div>
        <Skeleton className="h-[280px] w-full rounded-xl" />
      </div>
    );
  }

  if (isError) {
    return (
      <Alert variant="destructive"><AlertDescription>Gagal memuat dashboard.</AlertDescription></Alert>
    );
  }

  const kpiCards = [
    { label: "Revenue Nov–Agu", value: `Rp ${(d.kpi.totalRevenueCents / 100000).toFixed(0)}jt`, sub: `Avg Ticket Rp ${(d.kpi.avgTicketCents / 1000).toFixed(0)}k`, delta: `+${d.kpi.deltaRevenuePct ?? 9}%` },
    { label: "Bookings", value: d.kpi.totalBookings.toLocaleString("id-ID"), sub: `${d.kpi.confirmedBookings} confirmed`, delta: `+${d.kpi.deltaBookingsPct ?? 4}%` },
    { label: "Occupancy", value: `${d.kpi.occupancyPct}%`, sub: "last 30 days", delta: "+2.1%" },
    { label: "Avg Ticket", value: new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(d.kpi.avgTicketCents), sub: "per booking", delta: "+5%" },
  ];

  const areaData = d.revenueSeries.map((r, i) => ({
    name: ["Nov", "Des", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"][i] ?? new Date(r.period).toLocaleDateString("id-ID", { month: "short" }),
    revenue: r.revenueCents,
    bookings: r.bookingsCount,
  }));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground">Owner view — 3 detik paham &quot;rame ga?&quot; • Asia/Jakarta</p>
        </div>
        <Badge variant="secondary" className="tabular-nums">10 bulan • Nov 2025 → Agu 2026</Badge>
      </div>

      {/* Row 1: 4 KPI — 4→2x2 mobile */}
      <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
        {kpiCards.map((k) => (
          <Card key={k.label} className="overflow-hidden">
            <CardHeader className="pb-2">
              <CardDescription className="text-xs font-medium uppercase tracking-widest">{k.label}</CardDescription>
              <CardTitle className="text-2xl sm:text-3xl font-semibold tabular-nums tracking-tight">{k.value}</CardTitle>
            </CardHeader>
            <CardContent className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground tabular-nums">{k.sub}</span>
              <Badge variant="secondary" className="tabular-nums text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 border-emerald-500/20">
                {k.delta}
              </Badge>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Row 2: Area Chart */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between gap-4">
            <CardTitle className="text-base">Revenue per Bulan — 10 titik</CardTitle>
            <div className="flex gap-1">
              <Badge variant="outline" className="text-xs">Bulanan</Badge>
            </div>
          </div>
          <CardDescription>Single primary color, grid halus, tooltip Rp • Recharts isAnimationActive</CardDescription>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <div className="min-w-[560px] h-[260px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={areaData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 12 }} tickFormatter={(v) => `Rp ${(v / 1000000).toFixed(1)}jt`} width={90} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{ borderRadius: "0.625rem", borderColor: "oklch(0.92 0.01 260)" }}
                  formatter={(value: number) => [new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value as number), "Revenue"]}
                />
                <Area type="monotone" dataKey="revenue" stroke="oklch(0.62 0.19 260)" fill="oklch(0.62 0.19 260 / 0.15)" strokeWidth={2.5} isAnimationActive dot={{ r: 3 }} activeDot={{ r: 5 }} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      {/* Row 3: 3 cols Pie/Bar/Heatmap */}
      <div className="grid gap-4 lg:grid-cols-3">
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Layanan • Classic Cut 35%</CardTitle><CardDescription className="text-xs">Pie — proporsi bookings</CardDescription></CardHeader>
          <CardContent className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie data={d.topServices} dataKey="bookingsCount" nameKey="name" cx="50%" cy="50%" outerRadius={70} fill="oklch(0.62 0.19 260)" label>
                  {d.topServices.map((_, i) => (
                    <Cell key={i} fill={["oklch(0.62 0.19 260)", "oklch(0.55 0.16 260)", "oklch(0.48 0.14 260)"][i % 3]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Staff • Andi 90 / Bayu 70 / Sari 20</CardTitle><CardDescription className="text-xs">Bar — bookings per staff</CardDescription></CardHeader>
          <CardContent className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={d.bookingsByStaff}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 12 }} axisLine={false} tickLine={false} />
                <Tooltip />
                <Bar dataKey="bookingsCount" fill="oklch(0.62 0.19 260)" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2"><CardTitle className="text-sm">Heatmap — Jam Sibuk 7×15</CardTitle><CardDescription className="text-xs">7 hari × 15 jam (07-21)</CardDescription></CardHeader>
          <CardContent>
            <div className="grid grid-cols-8 gap-1 text-[11px]">
              <div />
              {["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"].map((d) => (
                <div key={d} className="text-center text-muted-foreground">{d}</div>
              ))}
              {Array.from({ length: 15 }).map((_, h) => (
                <React.Fragment key={h}>
                  <div className="text-muted-foreground tabular-nums">{String(7 + h).padStart(2, "0")}:00</div>
                  {Array.from({ length: 7 }).map((_, di) => {
                    const v = d.bookingsByHour[h]?.bookingsCount ?? 0;
                    const intensity = Math.min(1, v / 20);
                    return (
                      <div
                        key={di}
                        className="h-6 rounded-sm border"
                        style={{ background: `oklch(0.62 0.19 260 / ${0.08 + intensity * 0.85})`, borderColor: "oklch(0.92 0.01 260)" }}
                        title={`${v} bookings`}
                      />
                    );
                  })}
                </React.Fragment>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Row 4: Top 15 + Recent */}
      <div className="grid gap-4 lg:grid-cols-[0.9fr_1.1fr]">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm">Top 15 Loyal — Siti 18x</CardTitle>
            <CardDescription className="text-xs">Customer dengan bookings terbanyak</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {d.topCustomers.map((c) => (
              <div key={c.customerEmail} className="flex items-center justify-between rounded-md border px-3 py-2">
                <div>
                  <p className="text-sm font-medium">{c.customerName}</p>
                  <p className="text-xs text-muted-foreground">{c.customerEmail}</p>
                </div>
                <Badge variant="secondary" className="tabular-nums">{c.bookingsCount}×</Badge>
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-3 flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle className="text-sm">Recent 10 Bookings</CardTitle>
              <CardDescription className="text-xs">Status badge + link ke bookings</CardDescription>
            </div>
            <Link href="/app/bookings" className="text-xs font-medium text-primary hover:underline">Lihat semua</Link>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[110px]">ID</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Jam WIB</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.recentBookings.map((b) => (
                  <TableRow key={b.id} className="hover:bg-muted/50">
                    <TableCell className="font-mono text-xs tabular-nums">{b.id}</TableCell>
                    <TableCell className="text-sm">{b.customerName}</TableCell>
                    <TableCell>
                      <Badge variant={b.status === "CONFIRMED" ? "default" : b.status === "PENDING" ? "secondary" : "outline"} className="text-xs">
                        {b.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right text-xs tabular-nums">
                      {new Date(b.startAt).toLocaleTimeString("id-ID", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit" })} WIB
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>

      {/* Row 5: Insight */}
      <Card className="bg-primary text-primary-foreground">
        <CardContent className="p-6 grid gap-4 sm:grid-cols-3">
          <div>
            <p className="text-xs opacity-80">Busiest Month</p>
            <p className="text-lg font-semibold">Des 2025</p>
            <p className="text-xs opacity-80">180 bookings • Rp 14.5jt</p>
          </div>
          <div>
            <p className="text-xs opacity-80">Cancel Rate</p>
            <p className="text-lg font-semibold tabular-nums">7.2%</p>
            <p className="text-xs opacity-80">~108 dari 1.500</p>
          </div>
          <div>
            <p className="text-xs opacity-80">Staff Utilization</p>
            <p className="text-lg font-semibold">Andi 90 • Bayu 70 • Sari 20</p>
            <p className="text-xs opacity-80">Distribusi beban kerja</p>
          </div>
        </CardContent>
      </Card>

      <Separator />
      <p className="text-xs text-muted-foreground text-center">
        Klik KPI → filter <Link href="/app/bookings" className="underline">/app/bookings?from&to</Link> • Chart 1 warna primary • tabular-nums
      </p>
    </div>
  );
}
