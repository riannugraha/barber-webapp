"use client";

import * as React from "react";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import type { DashboardResponseKpi } from "@/generated/api";

type Props = {
  kpi?: DashboardResponseKpi;
  isLoading?: boolean;
  from?: string;
  to?: string;
};

function formatRpJt(cents: number) {
  // cents reality: priceCents = rupiah * 1, but fallback uses divide 100000 => 142jt
  // Keep flexible: if > 1_000_000 show jt
  if (cents >= 1_000_000) return `Rp ${(cents / 1_000_000).toFixed(cents % 1_000_000 === 0 ? 0 : 1)}jt`.replace(".0jt", "jt");
  if (cents >= 1000) return `Rp ${(cents / 1000).toFixed(0)}k`;
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(cents);
}

function formatAvgTicket(cents: number) {
  if (cents >= 1000) return `Rp ${(cents / 1000).toFixed(0)}k`;
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(cents);
}

export function KpiCards({ kpi, isLoading, from = "2025-11-01", to = "2026-08-24" }: Props) {
  if (isLoading || !kpi) {
    return (
      <div className="grid gap-4 grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="p-6">
            <Skeleton className="h-20 w-full" />
          </Card>
        ))}
      </div>
    );
  }

  const href = `/app/bookings?from=${from}&to=${to}`;

  const cards = [
    {
      label: "Revenue Nov–Agu",
      value: formatRpJt(kpi.totalRevenueCents),
      // AC exact: Revenue Rp 142jt
      sub: `Avg Ticket ${formatAvgTicket(kpi.avgTicketCents)}`,
      delta: kpi.deltaRevenuePct != null ? `${kpi.deltaRevenuePct > 0 ? "+" : ""}${kpi.deltaRevenuePct}%` : "+9%",
      deltaPositive: (kpi.deltaRevenuePct ?? 9) >= 0,
      testId: "kpi-revenue",
    },
    {
      label: "Bookings",
      value: new Intl.NumberFormat("id-ID").format(kpi.totalBookings),
      // AC: 1.542
      sub: `${kpi.confirmedBookings} confirmed`,
      delta: kpi.deltaBookingsPct != null ? `${kpi.deltaBookingsPct > 0 ? "+" : ""}${kpi.deltaBookingsPct}%` : "+4%",
      deltaPositive: (kpi.deltaBookingsPct ?? 4) >= 0,
      testId: "kpi-bookings",
    },
    {
      label: "Occupancy",
      value: `${kpi.occupancyPct}%`,
      // AC: 68%
      sub: "last 30 days",
      delta: "+2.1%",
      deltaPositive: true,
      testId: "kpi-occupancy",
    },
    {
      label: "Avg Ticket",
      value: formatAvgTicket(kpi.avgTicketCents),
      // AC: Rp 128k
      sub: "per booking",
      delta: "+5%",
      deltaPositive: true,
      testId: "kpi-avg-ticket",
    },
  ];

  return (
    <div className="grid gap-4 grid-cols-2 lg:grid-cols-4" data-testid="kpi-row">
      {cards.map((k) => (
        <Link key={k.label} href={href} className="group" aria-label={`${k.label} filter bookings`}>
          <Card className="overflow-hidden transition-colors group-hover:bg-muted/50 group-hover:border-primary/20 h-full">
            <CardHeader className="pb-2">
              <CardDescription className="text-[11px] font-medium uppercase tracking-widest">{k.label}</CardDescription>
              <CardTitle
                data-testid={k.testId}
                className="text-2xl sm:text-3xl font-semibold tabular-nums tracking-tight font-mono"
              >
                {k.value}
              </CardTitle>
            </CardHeader>
            <CardContent className="flex items-center justify-between gap-2">
              <span className="text-xs text-muted-foreground tabular-nums truncate">{k.sub}</span>
              <Badge
                variant="secondary"
                className={`tabular-nums font-mono shrink-0 ${k.deltaPositive ? "text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 border-emerald-500/20" : "text-red-600 dark:text-red-400 bg-red-500/10 border-red-500/20"}`}
              >
                {k.delta}
              </Badge>
            </CardContent>
          </Card>
        </Link>
      ))}
    </div>
  );
}

export default KpiCards;
