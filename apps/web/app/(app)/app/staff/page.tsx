"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Pencil, Clock, CalendarDays, Plus, Trash2, Save } from "lucide-react";
import { fetchStaff, fetchStaffDetail, updateStaffAvailability, type Availability, type AvailabilityInput } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

const DAYS = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

export default function StaffPage() {
  const qc = useQueryClient();
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [editOpen, setEditOpen] = React.useState(false);
  const [availabilityDraft, setAvailabilityDraft] = React.useState<AvailabilityInput[]>([]);
  const [overrideDate, setOverrideDate] = React.useState<Date | undefined>(new Date());
  const [overrideClosed, setOverrideClosed] = React.useState(true);
  const [overrideStart, setOverrideStart] = React.useState("09:00");
  const [overrideEnd, setOverrideEnd] = React.useState("17:00");

  const { data: staff, isLoading } = useQuery({
    queryKey: queryKeys.staff(),
    queryFn: () => fetchStaff(),
  });

  const { data: detail, isLoading: detailLoading } = useQuery({
    queryKey: queryKeys.staffDetail(selectedId ?? ""),
    queryFn: () => fetchStaffDetail(selectedId!),
    enabled: !!selectedId,
  });

  React.useEffect(() => {
    if (detail?.availability) {
      setAvailabilityDraft(
        detail.availability.map((a) => ({ dayOfWeek: a.dayOfWeek, startTime: a.startTime, endTime: a.endTime }))
      );
    } else if (selectedId) {
      // fallback demo
      setAvailabilityDraft([
        { dayOfWeek: 1, startTime: "09:00", endTime: "18:00" },
        { dayOfWeek: 2, startTime: "09:00", endTime: "18:00" },
        { dayOfWeek: 3, startTime: "09:00", endTime: "18:00" },
        { dayOfWeek: 4, startTime: "09:00", endTime: "18:00" },
        { dayOfWeek: 5, startTime: "09:00", endTime: "18:00" },
      ]);
    }
  }, [detail, selectedId]);

  const saveMut = useMutation({
    mutationFn: () => updateStaffAvailability(selectedId!, availabilityDraft),
    onSuccess: () => {
      toast.success("Jadwal mingguan disimpan");
      qc.invalidateQueries({ queryKey: queryKeys.staff() });
      qc.invalidateQueries({ queryKey: queryKeys.staffDetail(selectedId!) });
      setEditOpen(false);
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "Gagal simpan jadwal"),
  });

  const selectedStaff = staff?.find((s) => s.id === selectedId) ?? null;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Staff</h1>
        <p className="text-sm text-muted-foreground">List staff + availability editor mingguan + override tanggal • TanStack + ky • Toaster 422</p>
      </div>

      <div className="grid gap-4 lg:grid-cols-[360px_1fr]">
        {/* List */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-sm">Staff — 3 seed (Andi, Bayu, Sari)</CardTitle>
            <CardDescription className="text-xs">Klik untuk edit availability mingguan</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-16 w-full" />)
            ) : (
              staff?.map((s) => {
                const initials = s.name.slice(0, 2).toUpperCase();
                const active = selectedId === s.id;
                return (
                  <button
                    key={s.id}
                    onClick={() => setSelectedId(s.id)}
                    className={[
                      "w-full flex items-center gap-3 rounded-lg border p-3 text-left transition-colors",
                      active ? "bg-primary text-primary-foreground border-primary" : "bg-card hover:bg-muted",
                    ].join(" ")}
                  >
                    <Avatar className="h-10 w-10 border">
                      <AvatarImage src={s.avatarUrl ?? undefined} alt={s.name} />
                      <AvatarFallback className={active ? "bg-primary-foreground text-primary" : "bg-muted"}>{initials}</AvatarFallback>
                    </Avatar>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">{s.name}</p>
                      <p className={["text-xs truncate", active ? "text-primary-foreground/80" : "text-muted-foreground"].join(" ")}>{s.email ?? "—"}</p>
                    </div>
                    {s.isActive ? <Badge variant={active ? "secondary" : "default"} className="text-xs">Active</Badge> : <Badge variant="outline">Off</Badge>}
                  </button>
                );
              })
            )}
          </CardContent>
        </Card>

        {/* Detail */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-start justify-between gap-2">
              <div>
                <CardTitle className="text-base flex items-center gap-2">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  {selectedStaff ? `Jadwal ${selectedStaff.name}` : "Pilih staff"}
                </CardTitle>
                <CardDescription className="text-xs">Availability mingguan 07:00–21:00 • override tanggal libur</CardDescription>
              </div>
              {selectedStaff ? (
                <Button size="sm" onClick={() => setEditOpen(true)} className="gap-1.5">
                  <Pencil className="h-4 w-4" /> Edit jadwal
                </Button>
              ) : null}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {!selectedId ? (
              <div className="rounded-lg border border-dashed p-10 text-center">
                <p className="text-sm text-muted-foreground">Pilih staff di kiri untuk melihat jadwal.</p>
              </div>
            ) : detailLoading ? (
              <Skeleton className="h-64 w-full" />
            ) : (
              <>
                {/* Availability table read-only */}
                <div className="rounded-md border overflow-hidden">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Hari</TableHead>
                        <TableHead>Jam</TableHead>
                        <TableHead>Status</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {DAYS.map((day, idx) => {
                        const slot = availabilityDraft.find((a) => a.dayOfWeek === idx);
                        return (
                          <TableRow key={day}>
                            <TableCell className="font-medium text-sm">{day}</TableCell>
                            <TableCell className="tabular-nums text-sm">{slot ? `${slot.startTime}–${slot.endTime}` : "—"}</TableCell>
                            <TableCell>
                              {slot ? <Badge variant="secondary">Buka</Badge> : <Badge variant="outline">Libur</Badge>}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>

                <Separator />

                {/* Override */}
                <div className="rounded-lg border bg-muted/30 p-4 space-y-3">
                  <p className="text-sm font-medium flex items-center gap-2">
                    <CalendarDays className="h-4 w-4 text-muted-foreground" /> Override tanggal — libur / jam khusus
                  </p>
                  <div className="grid gap-3 sm:grid-cols-[1.2fr_0.8fr]">
                    <div>
                      <Label className="text-xs">Tanggal override</Label>
                      <Popover>
                        <PopoverTrigger asChild>
                          <Button variant="outline" className="w-full justify-start font-normal mt-1">
                            <CalendarDays className="mr-2 h-4 w-4" />
                            {overrideDate ? overrideDate.toLocaleDateString("id-ID") : "Pilih tanggal"}
                          </Button>
                        </PopoverTrigger>
                        <PopoverContent className="w-auto p-0">
                          <Calendar mode="single" selected={overrideDate} onSelect={setOverrideDate} />
                        </PopoverContent>
                      </Popover>
                    </div>
                    <div className="space-y-2">
                      <Label className="text-xs">Status</Label>
                      <div className="flex items-center gap-2 mt-1">
                        <Switch checked={overrideClosed} onCheckedChange={setOverrideClosed} aria-label="Tutup libur" />
                        <span className="text-sm">{overrideClosed ? "Libur (0 slot)" : "Buka jam khusus"}</span>
                      </div>
                      {!overrideClosed ? (
                        <div className="flex gap-2 mt-2">
                          <Input value={overrideStart} onChange={(e) => setOverrideStart(e.target.value)} placeholder="09:00" className="tabular-nums" />
                          <Input value={overrideEnd} onChange={(e) => setOverrideEnd(e.target.value)} placeholder="17:00" className="tabular-nums" />
                        </div>
                      ) : null}
                    </div>
                  </div>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      if (!overrideDate) return toast.error("Pilih tanggal");
                      toast.success(overrideClosed ? `Override libur ${overrideDate.toLocaleDateString("id-ID")} — 0 slot` : `Override ${overrideDate.toLocaleDateString("id-ID")} ${overrideStart}–${overrideEnd}`);
                    }}
                    className="gap-1.5"
                  >
                    <Plus className="h-4 w-4" /> Simpan override
                  </Button>
                  <p className="text-xs text-muted-foreground leading-relaxed">
                    Override libur → GetSlots return 0 slot untuk tanggal itu. Handle PRD: override libur, buffer, overnight, skill filter.
                  </p>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Edit dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="sm:max-w-[620px]">
          <DialogHeader>
            <DialogTitle>Edit availability mingguan — {selectedStaff?.name}</DialogTitle>
            <DialogDescription>Set jam buka per hari (0=Minggu ... 6=Sabtu). Kosongkan untuk libur. Simpan via ky PUT /staff/{"{id}"}/availability.</DialogDescription>
          </DialogHeader>

          <div className="space-y-3 max-h-[60vh] overflow-y-auto pr-1">
            {DAYS.map((day, idx) => {
              const entry = availabilityDraft.find((a) => a.dayOfWeek === idx);
              const isOpen = !!entry;
              return (
                <div key={day} className="flex items-center gap-3 rounded-lg border p-3">
                  <Switch
                    checked={isOpen}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        setAvailabilityDraft((prev) => [...prev, { dayOfWeek: idx, startTime: "09:00", endTime: "17:00" }]);
                      } else {
                        setAvailabilityDraft((prev) => prev.filter((a) => a.dayOfWeek !== idx));
                      }
                    }}
                    aria-label={`Toggle ${day}`}
                  />
                  <span className="w-20 text-sm font-medium">{day}</span>
                  {isOpen ? (
                    <>
                      <Input
                        value={entry?.startTime ?? "09:00"}
                        onChange={(e) => setAvailabilityDraft((prev) => prev.map((a) => (a.dayOfWeek === idx ? { ...a, startTime: e.target.value } : a)))}
                        className="tabular-nums"
                        placeholder="09:00"
                      />
                      <span className="text-sm text-muted-foreground">—</span>
                      <Input
                        value={entry?.endTime ?? "17:00"}
                        onChange={(e) => setAvailabilityDraft((prev) => prev.map((a) => (a.dayOfWeek === idx ? { ...a, endTime: e.target.value } : a)))}
                        className="tabular-nums"
                        placeholder="17:00"
                      />
                    </>
                  ) : (
                    <span className="text-xs text-muted-foreground flex-1">Libur — tidak ada slot</span>
                  )}
                </div>
              );
            })}
          </div>

          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setEditOpen(false)}>Batal</Button>
            <Button onClick={() => saveMut.mutate()} disabled={saveMut.isPending} className="gap-1.5">
              <Save className="h-4 w-4" /> {saveMut.isPending ? "Menyimpan..." : "Simpan jadwal"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
