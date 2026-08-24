"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Plus, Pencil, Trash2, Palette, Clock, DollarSign } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage, FormDescription } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchServices, createService, updateService, deleteService, formatPrice, formatDuration, parseApiError, type Service } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

const schema = z.object({
  name: z.string().min(2, "Nama minimal 2 karakter").max(60),
  description: z.string().max(200).optional().or(z.literal("")),
  durationMinutes: z.coerce.number().min(5, "Min 5").max(480, "Max 480"),
  bufferMinutes: z.coerce.number().min(0).max(60),
  priceCents: z.coerce.number().min(0, "Harga tidak boleh negatif"),
  color: z.string().min(4, "Warna hex mis #7c3aed").regex(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/, "Format hex salah"),
  isActive: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

export default function ServicesPage() {
  const qc = useQueryClient();
  const [open, setOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<Service | null>(null);
  const [deleteId, setDeleteId] = React.useState<string | null>(null);

  const { data: services, isLoading } = useQuery({
    queryKey: queryKeys.services(),
    queryFn: fetchServices,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", description: "", durationMinutes: 30, bufferMinutes: 10, priceCents: 85000, color: "#7c3aed", isActive: true },
  });

  React.useEffect(() => {
    if (editing) {
      form.reset({
        name: editing.name,
        description: editing.description ?? "",
        durationMinutes: editing.durationMinutes,
        bufferMinutes: editing.bufferMinutes,
        priceCents: editing.priceCents,
        color: editing.color,
        isActive: editing.isActive,
      });
    } else {
      form.reset({ name: "", description: "", durationMinutes: 30, bufferMinutes: 10, priceCents: 85000, color: "#7c3aed", isActive: true });
    }
  }, [editing, form, open]);

  const createMut = useMutation({
    mutationFn: (vals: FormValues) => createService(vals as unknown as Parameters<typeof createService>[0]),
    onSuccess: () => {
      toast.success("Layanan dibuat");
      qc.invalidateQueries({ queryKey: queryKeys.services() });
      setOpen(false);
      setEditing(null);
    },
    onError: async (err) => {
      const p = await parseApiError(err);
      toast.error(p.message);
      if (p.details) p.details.forEach((d) => form.setError(d.field as keyof FormValues, { message: d.message }));
    },
  });

  const updateMut = useMutation({
    mutationFn: (vals: FormValues) => updateService(editing!.id, vals),
    onSuccess: () => {
      toast.success("Layanan diperbarui");
      qc.invalidateQueries({ queryKey: queryKeys.services() });
      setOpen(false);
      setEditing(null);
    },
    onError: async (err) => {
      const p = await parseApiError(err);
      toast.error(p.message);
    },
  });

  const delMut = useMutation({
    mutationFn: (id: string) => deleteService(id),
    onSuccess: () => {
      toast.success("Layanan dihapus");
      qc.invalidateQueries({ queryKey: queryKeys.services() });
      setDeleteId(null);
    },
    onError: async (err) => {
      const p = await parseApiError(err);
      toast.error(p.message);
    },
  });

  const onSubmit = (vals: FormValues) => {
    if (editing) updateMut.mutate(vals);
    else createMut.mutate(vals);
  };

  const isPending = createMut.isPending || updateMut.isPending;

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Services</h1>
          <p className="text-sm text-muted-foreground">CRUD layanan — nama/durasi/buffer/harga/warna/active • Dialog + rhf+zod • ky • Toaster 422/409</p>
        </div>
        <Button
          onClick={() => {
            setEditing(null);
            setOpen(true);
          }}
          className="gap-2"
        >
          <Plus className="h-4 w-4" /> Tambah layanan
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm flex items-center gap-2"><Palette className="h-4 w-4 text-muted-foreground" /> Daftar layanan — 8 seed PRD §3</CardTitle>
          <CardDescription className="text-xs">Durasi jujur + buffer mencegah double-booking • warna untuk strip kartu</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}</div>
          ) : (
            <div className="rounded-md border overflow-hidden">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Warna</TableHead>
                      <TableHead>Nama</TableHead>
                      <TableHead>Durasi</TableHead>
                      <TableHead>Harga</TableHead>
                      <TableHead>Active</TableHead>
                      <TableHead className="text-right">Aksi</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {services?.map((s) => (
                      <TableRow key={s.id} className="hover:bg-muted/50">
                        <TableCell>
                          <span className="inline-block h-6 w-6 rounded-md border" style={{ background: s.color }} title={s.color} />
                        </TableCell>
                        <TableCell>
                          <p className="text-sm font-medium">{s.name}</p>
                          <p className="text-xs text-muted-foreground line-clamp-1 max-w-[220px]">{s.description}</p>
                        </TableCell>
                        <TableCell className="tabular-nums text-sm">
                          <span className="inline-flex items-center gap-1.5">
                            <Clock className="h-3.5 w-3.5 text-muted-foreground" /> {formatDuration(s.durationMinutes)}
                          </span>
                          <span className="ml-1 text-xs text-muted-foreground">+{s.bufferMinutes}m buffer</span>
                        </TableCell>
                        <TableCell className="tabular-nums text-sm font-medium">{formatPrice(s.priceCents)}</TableCell>
                        <TableCell>
                          {s.isActive ? <Badge>Active</Badge> : <Badge variant="outline">Off</Badge>}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              aria-label={`Edit ${s.name}`}
                              onClick={() => {
                                setEditing(s);
                                setOpen(true);
                              }}
                            >
                              <Pencil className="h-4 w-4" />
                            </Button>
                            <Button variant="ghost" size="icon" aria-label={`Hapus ${s.name}`} onClick={() => setDeleteId(s.id)}>
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create/Edit Dialog */}
      <Dialog open={open} onOpenChange={(o) => { setOpen(o); if (!o) setEditing(null); }}>
        <DialogContent className="sm:max-w-[520px]">
          <DialogHeader>
            <DialogTitle>{editing ? "Edit layanan" : "Tambah layanan baru"}</DialogTitle>
            <DialogDescription>Validasi zod — durasi 5-480 menit, buffer 0-60, harga ≥0. Error 422 ditampilkan via Toaster.</DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Nama</FormLabel>
                    <FormControl>
                      <Input placeholder="Classic Cut" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Deskripsi</FormLabel>
                    <FormControl>
                      <Input placeholder="Potongan klasik presisi" {...field} />
                    </FormControl>
                    <FormDescription className="text-xs">Maks 200 karakter — tampil di Card.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="grid grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="durationMinutes"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="gap-1.5 flex items-center"><Clock className="h-3.5 w-3.5" /> Durasi (menit)</FormLabel>
                      <FormControl><Input type="number" min={5} max={480} {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="bufferMinutes"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Buffer (menit)</FormLabel>
                      <FormControl><Input type="number" min={0} max={60} {...field} /></FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="priceCents"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="flex items-center gap-1.5"><DollarSign className="h-3.5 w-3.5" /> Harga (cents → Rp)</FormLabel>
                      <FormControl><Input type="number" min={0} {...field} /></FormControl>
                      <FormDescription className="text-xs">0 = Gratis skip Stripe.</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="color"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel className="flex items-center gap-1.5"><Palette className="h-3.5 w-3.5" /> Warna hex</FormLabel>
                      <FormControl>
                        <div className="flex gap-2">
                          <Input placeholder="#7c3aed" {...field} />
                          <span className="h-9 w-9 rounded-md border shrink-0" style={{ background: field.value }} aria-hidden="true" />
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="isActive"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-3">
                    <div>
                      <FormLabel>Active</FormLabel>
                      <FormDescription className="text-xs">Jika off, tidak muncul di /book Step1.</FormDescription>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} aria-label="Toggle active" />
                    </FormControl>
                  </FormItem>
                )}
              />
              <DialogFooter className="gap-2">
                <Button type="button" variant="outline" onClick={() => { setOpen(false); setEditing(null); }}>Batal</Button>
                <Button type="submit" disabled={isPending}>{isPending ? "Menyimpan..." : editing ? "Simpan perubahan" : "Buat layanan"}</Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* Delete confirm */}
      <Dialog open={!!deleteId} onOpenChange={(o) => !o && setDeleteId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Hapus layanan?</DialogTitle>
            <DialogDescription>Tindakan ini tidak dapat dibatalkan. Pastikan tidak ada booking aktif untuk layanan ini.</DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setDeleteId(null)}>Batal</Button>
            <Button variant="destructive" onClick={() => deleteId && delMut.mutate(deleteId)} disabled={delMut.isPending}>
              {delMut.isPending ? "Menghapus..." : "Hapus"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
