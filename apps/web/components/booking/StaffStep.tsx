"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Users, Check } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { fetchStaff, type Staff } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { cn } from "@/lib/utils";

type Props = {
  serviceId: string | null;
  selectedId: string | null; // null = Any available
  onSelect: (id: string | null, staff: Staff | null) => void;
};

export function StaffStep({ serviceId, selectedId, onSelect }: Props) {
  const { data: staff, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.staff(serviceId ?? undefined),
    queryFn: () => fetchStaff(serviceId ?? undefined),
  });

  if (isLoading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2" aria-busy="true" aria-label="Memuat staff">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="flex items-center gap-4 p-4">
            <Skeleton className="h-12 w-12 rounded-full" />
            <div className="space-y-2 flex-1">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-32" />
            </div>
          </Card>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <Alert variant="destructive">
        <AlertDescription className="flex items-center justify-between">
          <span>Gagal memuat staff.</span>
          <button onClick={() => refetch()} className="underline text-sm font-medium">
            Retry
          </button>
        </AlertDescription>
      </Alert>
    );
  }

  const list = staff ?? [];

  if (list.length === 0) {
    return (
      <div className="rounded-xl border border-dashed bg-card p-10 text-center">
        <p className="text-sm text-muted-foreground">Belum ada staff untuk layanan ini.</p>
      </div>
    );
  }

  return (
    <div role="radiogroup" aria-label="Pilih staff" className="grid gap-3 sm:grid-cols-2">
      {/* Any available */}
      <Card
        role="radio"
        aria-checked={selectedId === null}
        aria-label="Any available — pilih staff terbaik yang tersedia"
        tabIndex={0}
        onClick={() => onSelect(null, null)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onSelect(null, null);
          }
        }}
        className={cn(
          "flex cursor-pointer items-center gap-4 p-4 transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          selectedId === null && "border-primary ring-1 ring-primary bg-primary/[0.04]"
        )}
      >
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary text-primary-foreground">
          <Users className="h-6 w-6" aria-hidden="true" />
        </div>
        <div className="flex-1">
          <p className="text-sm font-semibold">Any available</p>
          <p className="text-xs text-muted-foreground">Staff terbaik yang tersedia otomatis</p>
        </div>
        {selectedId === null ? (
          <Check className="h-5 w-5 text-primary" aria-hidden="true" />
        ) : (
          <span className="h-5 w-5 shrink-0 rounded-full border border-input" aria-hidden="true" />
        )}
      </Card>

      {list.map((s) => {
        const selected = selectedId === s.id;
        const initials = s.name
          .split(" ")
          .map((n) => n[0])
          .join("")
          .slice(0, 2)
          .toUpperCase();
        return (
          <Card
            key={s.id}
            role="radio"
            aria-checked={selected}
            aria-label={`Pilih ${s.name}`}
            tabIndex={0}
            onClick={() => onSelect(s.id, s)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelect(s.id, s);
              }
            }}
            className={cn(
              "flex cursor-pointer items-center gap-4 p-4 transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              selected && "border-primary ring-1 ring-primary bg-primary/[0.04]"
            )}
          >
            <Avatar className="h-12 w-12 border">
              <AvatarImage src={s.avatarUrl ?? undefined} alt={`Foto ${s.name}`} />
              <AvatarFallback className="bg-muted text-sm font-medium">{initials}</AvatarFallback>
            </Avatar>
            <div className="flex-1 min-w-0">
              <p className="truncate text-sm font-semibold">{s.name}</p>
              <p className="truncate text-xs text-muted-foreground">{s.email ?? "Staff FlowBarber"}</p>
            </div>
            {selected ? (
              <Check className="h-5 w-5 shrink-0 text-primary" aria-hidden="true" />
            ) : (
              <span className="h-5 w-5 shrink-0 rounded-full border border-input" aria-hidden="true" />
            )}
          </Card>
        );
      })}
    </div>
  );
}
