"use client";

import * as React from "react";
import { motion, Reorder } from "framer-motion";
import { ChevronLeft, ChevronRight, Clock, Plus, Trash2, Calendar as CalendarIcon } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useQuery } from "@tanstack/react-query";
import { fetchSlots, fetchStaff, type Slot } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { cn } from "@/lib/utils";

function startOfWeek(d: Date) {
  const day = d.getDay(); // 0 Sun
  const diff = (day + 6) % 7; // Mon as start? Use Sun as 0
  const copy = new Date(d);
  copy.setDate(d.getDate() - diff);
  copy.setHours(0, 0, 0, 0);
  return copy;
}

function addDays(d: Date, n: number) {
  const c = new Date(d);
  c.setDate(c.getDate() + n);
  return c;
}

function toJakartaDateStr(d: Date) {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit", day: "2-digit" }).format(d);
}

function toJakartaDayLabel(d: Date) {
  return new Intl.DateTimeFormat("id-ID", { timeZone: "Asia/Jakarta", weekday: "short", day: "numeric", month: "short" }).format(d);
}

const HOURS = Array.from({ length: 14 }, (_, i) => 7 + i); // 07-20 inclusive => slots until 21

export default function CalendarPage() {
  const [weekStart, setWeekStart] = React.useState(() => startOfWeek(new Date()));
  const [selectedStaff, setSelectedStaff] = React.useState<string | undefined>(undefined);
  const [blocks, setBlocks] = React.useState<Array<{ id: string; day: number; hour: number; dur: number; title: string }>>([
    { id: "b1", day: 1, hour: 9, dur: 1, title: "Libur — Bayu" },
    { id: "b2", day: 3, hour: 13, dur: 2, title: "Training" },
  ]);
  const [detailSlot, setDetailSlot] = React.useState<Slot | null>(null);

  const { data: staff } = useQuery({ queryKey: queryKeys.staff(), queryFn: () => fetchStaff() });

  const weekDays = React.useMemo(() => Array.from({ length: 7 }).map((_, i) => addDays(weekStart, i)), [weekStart]);

  // Demo: fetch slots for first day to showcase TanStack cache for slots (30s)
  const firstDayStr = toJakartaDateStr(weekDays[0]);
  const { data: slotsData } = useQuery({
    queryKey: queryKeys.slots({ serviceId: "svc-classic", staffId: selectedStaff, date: firstDayStr }),
    queryFn: () => fetchSlots({ serviceId: "svc-classic", staffId: selectedStaff, date: firstDayStr, tz: "Asia/Jakarta" }),
    staleTime: 30_000,
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Calendar</h1>
          <p className="text-sm text-muted-foreground">Week view 07–21 • 7 kolom • drag blok libur (Framer Motion) • klik slot detail</p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={selectedStaff ?? "all"} onValueChange={(v) => setSelectedStaff(v === "all" ? undefined : v)}>
            <SelectTrigger className="w-[160px]">
              <SelectValue placeholder="Staff" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All staff</SelectItem>
              {staff?.map((s) => (
                <SelectItem key={s.id} value={s.id}>{s.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" onClick={() => setWeekStart(addDays(weekStart, -7))} aria-label="Minggu sebelumnya">
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={() => setWeekStart(startOfWeek(new Date()))} aria-label="Hari ini">
            <CalendarIcon className="h-4 w-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={() => setWeekStart(addDays(weekStart, 7))} aria-label="Minggu berikutnya">
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between gap-2">
            <CardTitle className="text-sm flex items-center gap-2">
              <Clock className="h-4 w-4 text-muted-foreground" />
              {toJakartaDayLabel(weekStart)} — {toJakartaDayLabel(addDays(weekStart, 6))} • Asia/Jakarta
            </CardTitle>
            <Badge variant="secondary" className="tabular-nums">{slotsData?.slots.filter((s) => s.available).length ?? 0} slot tersedia (day 1)</Badge>
          </div>
          <CardDescription className="text-xs">Drag blok libur untuk reschedule (Framer Motion) • 07:00–21:00 • 7 kolom grid • Slot polling 30s via TanStack</CardDescription>
        </CardHeader>
        <CardContent className="overflow-x-auto">
          <div className="min-w-[760px]">
            {/* Header 7 kolom */}
            <div className="grid grid-cols-[60px_repeat(7,1fr)] gap-px bg-border rounded-t-lg overflow-hidden">
              <div className="bg-muted p-2 text-xs font-medium text-muted-foreground">Jam</div>
              {weekDays.map((d) => {
                const isToday = toJakartaDateStr(d) === toJakartaDateStr(new Date());
                return (
                  <div key={d.toISOString()} className={cn("p-2 text-center", isToday ? "bg-primary text-primary-foreground" : "bg-muted")}>
                    <p className="text-xs font-medium">{toJakartaDayLabel(d)}</p>
                    <p className="text-[11px] opacity-80 tabular-nums">{toJakartaDateStr(d)}</p>
                  </div>
                );
              })}
            </div>

            {/* Grid hours */}
            <div className="grid grid-cols-[60px_repeat(7,1fr)] gap-px bg-border rounded-b-lg overflow-hidden border-t-0">
              {HOURS.map((hour) => (
                <React.Fragment key={hour}>
                  <div className="bg-card p-2 text-xs tabular-nums text-muted-foreground border-r">
                    {String(hour).padStart(2, "0")}:00
                  </div>
                  {weekDays.map((day, dayIdx) => {
                    const dayBlocks = blocks.filter((b) => b.day === dayIdx && b.hour === hour);
                    // slots for this day/hour demo — mock booked
                    const hasMockBooking = (dayIdx + hour) % 5 === 0;
                    return (
                      <div
                        key={`${hour}-${dayIdx}`}
                        className="relative bg-card min-h-[48px] p-1"
                      >
                        {hasMockBooking ? (
                          <Popover>
                            <PopoverTrigger asChild>
                              <button
                                className="w-full rounded-md bg-primary text-primary-foreground text-xs p-1.5 text-left hover:bg-primary/90 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                                aria-label={`Booking ${String(hour).padStart(2, "0")}:00 ${toJakartaDayLabel(day)}`}
                              >
                                <p className="font-medium truncate">Classic Cut</p>
                                <p className="text-[11px] opacity-90 truncate">Andi • CONFIRMED</p>
                              </button>
                            </PopoverTrigger>
                            <PopoverContent className="w-72">
                              <p className="text-sm font-semibold">Booking detail</p>
                              <p className="text-xs text-muted-foreground mt-1">{toJakartaDayLabel(day)} {String(hour).padStart(2, "0")}:00 WIB • 30m + 10m buffer</p>
                              <Separator className="my-2" />
                              <div className="flex gap-2">
                                <Button size="sm" variant="outline" onClick={() => setDetailSlot({ startAt: day.toISOString(), endAt: day.toISOString(), available: true, staffName: "Andi", staffId: "staff-andi", reason: null })}>View slot</Button>
                                <Button size="sm">Reschedule</Button>
                              </div>
                            </PopoverContent>
                          </Popover>
                        ) : (
                          <div className="h-full flex items-center justify-center">
                            <span className="text-[11px] text-muted-foreground/50">—</span>
                          </div>
                        )}

                        {/* Draggable blocks */}
                        {dayBlocks.map((b) => (
                          <motion.div
                            key={b.id}
                            drag
                            dragMomentum={false}
                            dragElastic={0.2}
                            whileDrag={{ scale: 1.02, zIndex: 10, boxShadow: "0 8px 24px rgba(0,0,0,0.15)" }}
                            className="absolute inset-1 rounded-md bg-amber-500/90 text-amber-950 dark:text-amber-50 border border-amber-600/20 p-1.5 flex items-center justify-between gap-1 cursor-grab active:cursor-grabbing"
                          >
                            <span className="text-xs font-medium truncate">{b.title}</span>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-6 w-6 shrink-0 hover:bg-amber-600/20"
                              onClick={() => setBlocks((prev) => prev.filter((x) => x.id !== b.id))}
                              aria-label="Hapus blok"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </motion.div>
                        ))}
                      </div>
                    );
                  })}
                </React.Fragment>
              ))}
            </div>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              onClick={() => setBlocks((prev) => [...prev, { id: `b${Date.now()}`, day: Math.floor(Math.random() * 7), hour: 9 + Math.floor(Math.random() * 6), dur: 1, title: "Blok libur baru" }])}
              className="gap-1.5"
            >
              <Plus className="h-4 w-4" /> Tambah blok libur (drag)
            </Button>
            <span className="text-xs text-muted-foreground">Drag blok kuning untuk geser jam/hari — Framer Motion drag tanpa lib external calendar.</span>
          </div>
        </CardContent>
      </Card>

      {/* Legend + polling info */}
      <Card>
        <CardContent className="p-4 flex flex-wrap gap-4 text-xs">
          <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-sm bg-primary" /> available — slot bisa dibooking</span>
          <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-sm bg-amber-500/60" /> buffer — tidak bisa</span>
          <span className="inline-flex items-center gap-1.5"><span className="h-3 w-3 rounded-sm bg-muted border" /> taken — sudah terisi</span>
          <span className="ml-auto text-muted-foreground tabular-nums">Polling 30s • TanStack staleTime 30s • cache slots</span>
        </CardContent>
      </Card>

      <Dialog open={!!detailSlot} onOpenChange={(o) => !o && setDetailSlot(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Slot detail</DialogTitle>
            <DialogDescription>Asia/Jakarta • {detailSlot?.startAt ? new Date(detailSlot.startAt).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" }) : "—"}</DialogDescription>
          </DialogHeader>
          <div className="rounded-lg bg-muted p-3 text-sm">
            <p>Staff: {detailSlot?.staffName ?? "—"}</p>
            <p>Status: {detailSlot?.available ? "available" : detailSlot?.reason}</p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDetailSlot(null)}>Tutup</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
