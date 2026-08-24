"use client";

import * as React from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { DashboardResponseInsights, DashboardResponseKpi } from "@/generated/api";
import { TrendingUp, AlertTriangle, Users } from "lucide-react";

type Props = {
  insights?: DashboardResponseInsights;
  kpi?: DashboardResponseKpi;
  isLoading?: boolean;
};

export function InsightRow({ insights, kpi, isLoading }: Props) {
  if (isLoading || (!insights && !kpi)) {
    return <Skeleton className="h-28 w-full rounded-xl" />;
  }

  const busiestMonth = insights?.busiestMonth ?? "Des 2025";
  const busiestCount = insights?.busiestMonthCount ?? 180;
  const busiestRevenue = insights?.busiestMonthRevenue ?? 14_500_000;
  const cancelRate = insights?.cancelRate ?? 7.2;
  const utilization = insights?.utilization ?? kpi?.occupancyPct ?? 68;

  // Format revenue jt
  const revJt = (busiestRevenue / 1_000_000).toFixed(1).replace(".0", "");

  return (
    <div className="grid gap-4 sm:grid-cols-3" data-testid="insight-row">
      <Card className="bg-primary text-primary-foreground border-primary overflow-hidden">
        <CardContent className="p-6">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-xs opacity-80 flex items-center gap-1.5">
                <TrendingUp className="h-3.5 w-3.5" /> Busiest Month
              </p>
              <p className="text-lg font-semibold tracking-tight mt-1">{busiestMonth}</p>
              <p className="text-xs opacity-80 tabular-nums font-mono mt-1">
                {busiestCount} bookings • Rp {revJt}jt
              </p>
            </div>
            <div className="h-8 w-8 rounded-lg bg-white/15 flex items-center justify-center shrink-0">
              <TrendingUp className="h-4 w-4" />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <CardContent className="p-6">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-xs text-muted-foreground flex items-center gap-1.5">
                <AlertTriangle className="h-3.5 w-3.5" /> Cancel Rate
              </p>
              <p className="text-lg font-semibold tabular-nums font-mono tracking-tight mt-1">{cancelRate.toFixed(1)}%</p>
              <p className="text-xs text-muted-foreground tabular-nums font-mono mt-1">~{Math.round((cancelRate / 100) * (kpi?.totalBookings ?? 1500))} dari {kpi?.totalBookings ?? 1500}</p>
            </div>
            <div className="h-8 w-8 rounded-lg bg-amber-500/10 border border-amber-500/20 flex items-center justify-center shrink-0">
              <AlertTriangle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card className="overflow-hidden">
        <CardContent className="p-6">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-xs text-muted-foreground flex items-center gap-1.5">
                <Users className="h-3.5 w-3.5" /> Staff Utilization
              </p>
              <p className="text-lg font-semibold tracking-tight mt-1 tabular-nums">{utilization}%</p>
              <p className="text-xs text-muted-foreground tabular-nums font-mono mt-1">Andi 90 • Bayu 70 • Sari 20</p>
            </div>
            <div className="h-8 w-8 rounded-lg bg-primary/10 border border-primary/20 flex items-center justify-center shrink-0">
              <Users className="h-4 w-4 text-primary" />
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default InsightRow;
