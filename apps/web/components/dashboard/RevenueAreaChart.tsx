"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";
import type { DashboardResponseRevenueSeriesItem } from "@/generated/api";

type Granularity = "day" | "week" | "month";

type Props = {
  data?: DashboardResponseRevenueSeriesItem[];
  granularity: Granularity;
  onGranularityChange: (g: Granularity) => void;
  isLoading?: boolean;
};

const MONTH_LABELS = ["Nov", "Des", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu"];

function toChartData(series: DashboardResponseRevenueSeriesItem[]) {
  return series.map((r, i) => {
    // Prefer label from API, fallback to MONTH_LABELS for 10 titik, else format period
    const label = r.label ?? MONTH_LABELS[i] ?? new Date(r.period).toLocaleDateString("id-ID", { month: "short" });
    return {
      name: label,
      revenue: r.revenueCents,
      bookings: r.bookingsCount,
      period: r.period,
    };
  });
}

export function RevenueAreaChart({ data, granularity, onGranularityChange, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-64" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[260px] w-full rounded-xl" />
        </CardContent>
      </Card>
    );
  }

  const chartData = toChartData(data);

  return (
    <Card data-testid="revenue-area">
      <CardHeader className="pb-2">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <CardTitle className="text-base">Revenue per Bulan — 10 titik</CardTitle>
            <CardDescription className="text-xs">Single primary oklch violet 260, grid halus, tooltip Rp • Recharts isAnimationActive</CardDescription>
          </div>
          <div
            role="group"
            aria-label="Granularity toggle"
            className="inline-flex items-center rounded-lg border p-1 bg-muted/50 shrink-0"
          >
            {([
              { value: "day" as const, label: "Harian" },
              { value: "week" as const, label: "Mingguan" },
              { value: "month" as const, label: "Bulanan" },
            ]).map((opt) => (
              <Button
                key={opt.value}
                variant={granularity === opt.value ? "secondary" : "ghost"}
                size="sm"
                className={`h-7 px-3 text-xs font-medium ${granularity === opt.value ? "bg-card shadow-sm border" : ""}`}
                onClick={() => onGranularityChange(opt.value)}
                aria-pressed={granularity === opt.value}
                data-testid={`granularity-${opt.value}`}
              >
                {opt.label}
              </Button>
            ))}
          </div>
        </div>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <div className="min-w-[560px] h-[260px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ left: 8, right: 16, top: 8, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" vertical={false} />
              <XAxis dataKey="name" tick={{ fontSize: 12 }} axisLine={false} tickLine={false} dy={8} />
              <YAxis
                tick={{ fontSize: 12 }}
                tickFormatter={(v: number) => `Rp ${(v / 1_000_000).toFixed(1)}jt`}
                width={90}
                axisLine={false}
                tickLine={false}
                className="tabular-nums"
              />
              <Tooltip
                contentStyle={{ borderRadius: "0.625rem", borderColor: "oklch(0.92 0.01 260)", background: "oklch(1 0 0)" }}
                labelStyle={{ fontWeight: 600 }}
                formatter={(value: number, name: string) => {
                  if (name === "revenue") {
                    return [new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value as number), "Revenue"];
                  }
                  return [value, name];
                }}
                cursor={{ stroke: "oklch(0.62 0.19 260 / 0.2)", strokeWidth: 1, strokeDasharray: "3 3" }}
              />
              <Area
                type="monotone"
                dataKey="revenue"
                stroke="oklch(0.62 0.19 260)"
                fill="oklch(0.62 0.19 260 / 0.15)"
                strokeWidth={2.5}
                isAnimationActive={true}
                animationDuration={600}
                dot={{ r: 3, strokeWidth: 2, fill: "oklch(0.62 0.19 260)" }}
                activeDot={{ r: 5, strokeWidth: 2, fill: "oklch(0.62 0.19 260)", stroke: "white" }}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
        <p className="text-[11px] text-muted-foreground mt-2 text-center tabular-nums">
          {granularity === "month" ? "10 bulan • Nov 2025 → Agu 2026" : granularity === "week" ? "Mingguan • 10 minggu terakhir" : "Harian • 30 hari terakhir"} • &lt;800ms render • 1 warna primary
        </p>
      </CardContent>
    </Card>
  );
}

export default RevenueAreaChart;
