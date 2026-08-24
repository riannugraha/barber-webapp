"use client";

import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from "recharts";
import type { DashboardResponseTopServicesItem } from "@/generated/api";

type Props = {
  data?: DashboardResponseTopServicesItem[];
  isLoading?: boolean;
};

const COLORS = [
  "oklch(0.62 0.19 260)",
  "oklch(0.55 0.16 260)",
  "oklch(0.48 0.14 260)",
  "oklch(0.70 0.12 260)",
  "oklch(0.45 0.10 260)",
  "oklch(0.60 0.14 260)",
];

export function ServicePie({ data, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-40" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[200px] w-full rounded-xl" />
        </CardContent>
      </Card>
    );
  }

  // Ensure sorted by bookingsCount desc, compute percentage if missing (~35% Classic Cut)
  const total = data.reduce((s, d) => s + (d.bookingsCount ?? 0), 0) || 1;
  const chartData = data.map((d) => ({
    name: d.name ?? "Layanan",
    value: d.bookingsCount ?? 0,
    percentage: d.percentage ?? Math.round(((d.bookingsCount ?? 0) / total) * 100),
    revenueCents: d.revenueCents ?? 0,
  }));

  const classic = chartData.find((d) => d.name.toLowerCase().includes("classic")) ?? chartData[0];
  const classicPct = classic?.percentage ?? 35;

  return (
    <Card data-testid="service-pie">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm">Layanan • Classic Cut {classicPct}%</CardTitle>
        <CardDescription className="text-xs">Pie — proporsi bookings • 1 warna primary</CardDescription>
      </CardHeader>
      <CardContent className="h-[220px] p-2">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={chartData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="45%"
              outerRadius={72}
              innerRadius={0}
              labelLine={false}
              label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
              isAnimationActive={true}
              animationDuration={650}
            >
              {chartData.map((_, i) => (
                <Cell key={i} fill={COLORS[i % COLORS.length]} stroke="white" strokeWidth={1} />
              ))}
            </Pie>
            <Tooltip
              formatter={(value: number, name: string, props: unknown) => {
                const payload = (props as { payload?: { percentage?: number } })?.payload;
                const pct = payload?.percentage ?? Math.round((value / total) * 100);
                return [`${value} bookings (${pct}%)`, name];
              }}
              contentStyle={{ borderRadius: "0.625rem", fontSize: 12 }}
            />
            <Legend verticalAlign="bottom" height={24} iconType="circle" wrapperStyle={{ fontSize: 11 }} />
          </PieChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

export default ServicePie;
