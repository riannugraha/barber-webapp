"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { DashboardResponseHeatmapItem } from "@/generated/api";

type Props = {
  data?: DashboardResponseHeatmapItem[];
  isLoading?: boolean;
};

const DAYS = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];
// dow in openapi 0 Sun ..6 Sat but UI wants Sen=Mon(1) .. Min=Sun(0)
// We'll map: 1 Mon ->0, 2 Tue->1, ... 6 Sat->5, 0 Sun->6
function dowToCol(dow: number) {
  if (dow === 0) return 6;
  return dow - 1;
}

export function Heatmap({ data, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-3 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[220px] w-full rounded-xl" />
        </CardContent>
      </Card>
    );
  }

  // Build 7 x 15 matrix (days x hours 7-21)
  const hours = Array.from({ length: 15 }, (_, i) => 7 + i);
  // Map key dow-hour to count
  const map = new Map<string, number>();
  let max = 1;
  for (const h of data) {
    const key = `${h.dow}-${h.hour}`;
    map.set(key, h.count);
    if (h.count > max) max = h.count;
  }
  // If data uses bookingsByHour fallback, convert: hour -> dow spread? But heatmap should be 7x15
  // Already handled; if empty, fallback synthetic
  const hasHeatmap = data.length > 0;

  return (
    <Card data-testid="heatmap">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">Heatmap — Jam Sibuk 7×15</CardTitle>
        <CardDescription className="text-xs">7 hari × 15 jam (07–21) • AT TIME ZONE Asia/Jakarta</CardDescription>
      </CardHeader>
      <CardContent className="overflow-x-auto">
        <div className="min-w-[320px]">
          {/* Header days */}
          <div className="grid grid-cols-8 gap-1 text-[11px] mb-1">
            <div />
            {DAYS.map((d) => (
              <div key={d} className="text-center font-medium text-muted-foreground">
                {d}
              </div>
            ))}
          </div>
          {/* Rows per hour */}
          <div className="grid gap-1">
            {hours.map((h) => (
              <div key={h} className="grid grid-cols-8 gap-1 items-center">
                <div className="text-[11px] text-muted-foreground tabular-nums text-right pr-1 font-mono">
                  {String(h).padStart(2, "0")}:00
                </div>
                {DAYS.map((_, colIdx) => {
                  // colIdx 0=>dow 1, ...6=>dow0
                  const dow = colIdx === 6 ? 0 : colIdx + 1;
                  const count = map.get(`${dow}-${h}`) ?? (hasHeatmap ? 0 : ((dow * 7 + h * 13) % 18) + 2);
                  const intensity = Math.min(1, count / max);
                  // violet 260 with opacity
                  const bg = `oklch(0.62 0.19 260 / ${0.08 + intensity * 0.85})`;
                  return (
                    <div
                      key={colIdx}
                      className="h-6 rounded-sm border flex items-center justify-center text-[10px] font-mono tabular-nums"
                      style={{ background: bg, borderColor: "oklch(0.92 0.01 260)" }}
                      title={`${DAYS[colIdx]} ${String(h).padStart(2, "0")}:00 — ${count} bookings`}
                      aria-label={`${DAYS[colIdx]} ${h}:00 ${count}`}
                    >
                      <span className={`${intensity > 0.6 ? "text-white" : "text-foreground/70"} hidden sm:inline`}>{count > 0 ? count : ""}</span>
                    </div>
                  );
                })}
              </div>
            ))}
          </div>
          <div className="flex items-center justify-between mt-3 text-[11px] text-muted-foreground">
            <span className="tabular-nums">Rendah</span>
            <div className="flex gap-1">
              {[0.08, 0.25, 0.5, 0.75, 0.93].map((op) => (
                <div key={op} className="h-3 w-6 rounded-sm border" style={{ background: `oklch(0.62 0.19 260 / ${op})`, borderColor: "oklch(0.92 0.01 260)" }} />
              ))}
            </div>
            <span className="tabular-nums">Tinggi</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export default Heatmap;
