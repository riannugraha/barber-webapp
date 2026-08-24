"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Clock, BadgeCheck, Sparkles } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { fetchServices, formatPrice, formatDuration, type Service } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { cn } from "@/lib/utils";

type Props = {
  selectedId: string | null;
  onSelect: (id: string, service: Service) => void;
};

export function ServiceStep({ selectedId, onSelect }: Props) {
  const { data: services, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.services(),
    queryFn: fetchServices,
  });

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" aria-busy="true" aria-label="Memuat layanan">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i} className="p-6 space-y-3">
            <Skeleton className="h-5 w-3/4" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-6 w-1/2" />
          </Card>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <Alert variant="destructive">
        <AlertDescription className="flex items-center justify-between gap-4">
          <span>Gagal memuat layanan. Coba lagi.</span>
          <button onClick={() => refetch()} className="text-sm font-medium underline">
            Retry
          </button>
        </AlertDescription>
      </Alert>
    );
  }

  if (!services || services.length === 0) {
    return (
      <div className="rounded-xl border border-dashed bg-card p-10 text-center">
        <p className="text-sm text-muted-foreground">Belum ada layanan tersedia — hubungi admin untuk menambah layanan.</p>
        <p className="mt-2 text-xs text-muted-foreground">Belum ada booking hari ini — bagikan link /book</p>
      </div>
    );
  }

  return (
    <div role="radiogroup" aria-label="Pilih layanan" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {services.map((svc) => {
        const selected = selectedId === svc.id;
        const isFree = svc.priceCents === 0;
        return (
          <Card
            key={svc.id}
            role="radio"
            aria-checked={selected}
            aria-label={`${svc.name} ${formatDuration(svc.durationMinutes)} ${formatPrice(svc.priceCents)}`}
            tabIndex={0}
            onClick={() => onSelect(svc.id, svc)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelect(svc.id, svc);
              }
            }}
            className={cn(
              "group relative cursor-pointer transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              selected && "border-primary ring-1 ring-primary bg-primary/[0.04]"
            )}
          >
            {/* color strip */}
            <div className="absolute left-0 top-0 h-full w-1 rounded-l-xl" style={{ background: svc.color }} aria-hidden="true" />
            <CardHeader className="pb-3 pl-6">
              <div className="flex items-start justify-between gap-2">
                <CardTitle className="text-base leading-tight">{svc.name}</CardTitle>
                {selected ? (
                  <BadgeCheck className="h-5 w-5 shrink-0 text-primary" aria-hidden="true" />
                ) : isFree ? (
                  <Badge variant="secondary" className="shrink-0 gap-1">
                    <Sparkles className="h-3 w-3" /> Gratis
                  </Badge>
                ) : null}
              </div>
              {svc.description ? (
                <CardDescription className="line-clamp-2 text-xs leading-relaxed">{svc.description}</CardDescription>
              ) : null}
            </CardHeader>
            <CardContent className="flex flex-wrap items-center gap-2 pl-6">
              <Badge variant="outline" className="gap-1.5 font-normal tabular-nums">
                <Clock className="h-3 w-3" />
                {formatDuration(svc.durationMinutes)}
              </Badge>
              <span className="text-xs text-muted-foreground">+ {svc.bufferMinutes}m buffer</span>
              <span className="ml-auto text-sm font-semibold tabular-nums">
                {isFree ? <span className="text-primary">Gratis</span> : formatPrice(svc.priceCents)}
              </span>
            </CardContent>
            {/* price accent on hover */}
            <span className="sr-only">{selected ? "Terpilih" : "Pilih layanan ini"}</span>
          </Card>
        );
      })}
    </div>
  );
}
