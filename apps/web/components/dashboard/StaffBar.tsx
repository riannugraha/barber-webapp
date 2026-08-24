"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from "recharts";
import type { DashboardResponseBookingsByStaffItem } from "@/generated/api";

type Props = {
  data?: DashboardResponseBookingsByStaffItem[];
  isLoading?: boolean;
};

const BAR_COLORS: Record<string, string> = {
  Andi: "oklch(0.62 0.19 260)",
  Bayu: "oklch(0.55 0.16 260)",
  Sari: "oklch(0.48 0.14 260)",
};

export function StaffBar({ data, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-36" />
          <Skeleton className="h-3 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[200px] w-full rounded-xl" />
        </CardContent>
      </Card>
    );
  }

  // Ensure order Andi 90 Bayu 70 Sari 20
  const sorted = [...data].sort((a, b) => (b.bookingsCount ?? 0) - (a.bookingsCount ?? 0));

  return (
    <Card data-testid="staff-bar">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">Staff • {sorted.map((s) => `${s.name} ${s.bookingsCount}`).join(" / ") || "Andi 90 / Bayu 70 / Sari 20"}</CardTitle>
        <CardDescription className="text-xs">Bar — bookings per staff • GROUP BY staff.id</CardDescription>
      </CardHeader>
      <CardContent className="h-[220px] p-2">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={sorted} margin={{ top: 8, right: 8, left: 8, bottom: 8 }}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" vertical={false} />
            <XAxis dataKey="name" tick={{ fontSize: 12 }} axisLine={false} tickLine={false} dy={6} />
            <YAxis tick={{ fontSize: 12 }} axisLine={false} tickLine={false} width={32} className="tabular-nums" />
            <Tooltip
              cursor={{ fill: "oklch(0.96 0.01 260 / 0.5)" }}
              contentStyle={{ borderRadius: "0.625rem", fontSize: 12 }}
              formatter={(value: number) => [`${value} bookings`, "Bookings"]}
            />
            <Bar dataKey="bookingsCount" radius={[6, 6, 0, 0]} isAnimationActive={true} animationDuration={600}>
              {sorted.map((entry, idx) => (
                <Cell key={entry.name ?? idx} fill={BAR_COLORS[entry.name ?? ""] ?? "oklch(0.62 0.19 260)"} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

export default StaffBar;
