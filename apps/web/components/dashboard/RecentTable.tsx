"use client";

import * as React from "react";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { Booking } from "@/generated/api";

type Props = {
  data?: Booking[];
  isLoading?: boolean;
};

function statusVariant(status: string) {
  switch (status) {
    case "CONFIRMED":
    case "COMPLETED":
      return "default" as const;
    case "PENDING":
      return "secondary" as const;
    case "CANCELLED":
    case "NO_SHOW":
      return "outline" as const;
    default:
      return "secondary" as const;
  }
}

export function RecentTable({ data, isLoading }: Props) {
  if (isLoading || !data) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-4 w-32" />
          <Skeleton className="h-3 w-40" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full rounded-xl" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card data-testid="recent-table">
      <CardHeader className="pb-3 flex-row items-center justify-between space-y-0 gap-2">
        <div>
          <CardTitle className="text-sm">Recent 10 Bookings</CardTitle>
          <CardDescription className="text-xs">Status badge + jam WIB • @/app/bookings</CardDescription>
        </div>
        <Link href="/app/bookings" className="text-xs font-medium text-primary hover:underline shrink-0" data-testid="recent-see-all">
          Lihat semua
        </Link>
      </CardHeader>
      <CardContent className="p-0 sm:p-6 sm:pt-0">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[110px] text-xs">ID</TableHead>
                <TableHead className="text-xs">Customer</TableHead>
                <TableHead className="text-xs">Status</TableHead>
                <TableHead className="text-right text-xs">Jam WIB</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.slice(0, 10).map((b) => (
                <TableRow key={b.id} className="hover:bg-muted/50">
                  <TableCell className="font-mono text-xs tabular-nums max-w-[110px] truncate" title={b.id}>
                    {b.id.slice(0, 8)}
                  </TableCell>
                  <TableCell className="text-sm">
                    <div className="flex flex-col">
                      <span className="font-medium truncate max-w-[140px]">{b.customerName}</span>
                      <span className="text-xs text-muted-foreground truncate max-w-[140px] hidden sm:inline">{b.customerEmail}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(b.status)} className="text-xs tabular-nums">
                      {b.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right text-xs tabular-nums font-mono">
                    {new Date(b.startAt).toLocaleTimeString("id-ID", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit" })} WIB
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {data.length === 0 && <p className="text-sm text-muted-foreground text-center py-8">Belum ada booking</p>}
        </div>
      </CardContent>
    </Card>
  );
}

export default RecentTable;
