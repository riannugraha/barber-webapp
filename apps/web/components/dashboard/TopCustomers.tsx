"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import type { DashboardResponseTopCustomersItem } from "@/generated/api";

type Props = {
  data?: DashboardResponseTopCustomersItem[];
  isLoading?: boolean;
};

export function TopCustomers({ data, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-40" />
        </CardHeader>
        <CardContent className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-md" />
          ))}
        </CardContent>
      </Card>
    );
  }

  // Ensure sorted desc, highlight Siti 18x top
  const sorted = [...data].sort((a, b) => (b.bookingsCount ?? 0) - (a.bookingsCount ?? 0));

  return (
    <Card data-testid="top-customers">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Top 15 Loyal — {sorted[0]?.customerName ?? "Siti"} {sorted[0]?.bookingsCount ?? 18}×</CardTitle>
        <CardDescription className="text-xs">Customer dengan bookings terbanyak • GROUP BY email</CardDescription>
      </CardHeader>
      <CardContent className="space-y-2 max-h-[420px] overflow-y-auto pr-1">
        {sorted.slice(0, 15).map((c, idx) => {
          const isTop = idx === 0;
          const initials = (c.customerName ?? "U").split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase();
          return (
            <div
              key={c.customerEmail ?? `${c.customerName}-${idx}`}
              className={`flex items-center justify-between rounded-md border px-3 py-2 transition-colors hover:bg-muted/50 ${isTop ? "bg-primary/5 border-primary/20" : ""}`}
            >
              <div className="flex items-center gap-3 min-w-0">
                <Avatar className="h-7 w-7 border shrink-0">
                  <AvatarFallback className={`text-[11px] font-semibold ${isTop ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}>
                    {initials}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{c.customerName}</p>
                  <p className="text-xs text-muted-foreground truncate">{c.customerEmail}</p>
                </div>
                {isTop && (
                  <Badge className="ml-1 bg-primary text-primary-foreground text-[10px] px-1.5 py-0 hidden sm:inline-flex">Top</Badge>
                )}
              </div>
              <div className="flex items-center gap-2 shrink-0">
                {c.totalSpentCents ? (
                  <span className="hidden sm:inline text-xs text-muted-foreground tabular-nums font-mono">
                    Rp {(c.totalSpentCents / 1000).toFixed(0)}k
                  </span>
                ) : null}
                <Badge variant={isTop ? "default" : "secondary"} className="tabular-nums font-mono">
                  {c.bookingsCount}×
                </Badge>
              </div>
            </div>
          );
        })}
        {sorted.length === 0 && <p className="text-sm text-muted-foreground text-center py-8">Belum ada customer loyal</p>}
      </CardContent>
    </Card>
  );
}

export default TopCustomers;
