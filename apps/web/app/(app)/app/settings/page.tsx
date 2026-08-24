"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage, FormDescription } from "@/components/ui/form";
import { Skeleton } from "@/components/ui/skeleton";
import { Upload, Building2, Globe, Image as ImageIcon, Save } from "lucide-react";
import { fetchOrganization, updateOrganization, uploadLogo, parseApiError } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

const schema = z.object({
  name: z.string().min(2, "Nama org minimal 2 karakter").max(60),
  timezone: z.string().min(1, "Pilih timezone"),
});

type FormValues = z.infer<typeof schema>;

const TIMEZONES = ["Asia/Jakarta", "Asia/Makassar", "Asia/Jayapura", "Asia/Singapore", "UTC"];

export default function SettingsPage() {
  const qc = useQueryClient();
  const [logoPreview, setLogoPreview] = React.useState<string | null>(null);
  const [logoFile, setLogoFile] = React.useState<File | null>(null);

  const { data: org, isLoading } = useQuery({
    queryKey: queryKeys.organization(),
    queryFn: fetchOrganization,
  });

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: "", timezone: "Asia/Jakarta" },
  });

  React.useEffect(() => {
    if (org) {
      form.reset({ name: org.name, timezone: org.timezone });
      if (org.logoUrl) setLogoPreview(org.logoUrl);
    }
  }, [org, form]);

  const saveMut = useMutation({
    mutationFn: (vals: FormValues) => updateOrganization(vals),
    onSuccess: () => {
      toast.success("Pengaturan organisasi disimpan");
      qc.invalidateQueries({ queryKey: queryKeys.organization() });
    },
    onError: async (err) => {
      const p = await parseApiError(err);
      toast.error(p.message);
    },
  });

  const logoMut = useMutation({
    mutationFn: (file: File) => uploadLogo(file),
    onSuccess: (res) => {
      toast.success("Logo berhasil diupload — Supabase Storage 1GB");
      setLogoPreview(res.logoUrl);
      qc.invalidateQueries({ queryKey: queryKeys.organization() });
    },
    onError: async (err) => {
      const p = await parseApiError(err);
      toast.error(p.message);
    },
  });

  const onSubmit = (vals: FormValues) => saveMut.mutate(vals);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    if (f.size > 2 * 1024 * 1024) {
      toast.error("File terlalu besar — maks 2MB");
      return;
    }
    setLogoFile(f);
    const url = URL.createObjectURL(f);
    setLogoPreview(url);
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-3xl">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">Org name, timezone, logo upload • Supabase Storage 1GB • ky multipart • rhf+zod</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm flex items-center gap-2">
            <Building2 className="h-4 w-4 text-muted-foreground" /> Organisasi
          </CardTitle>
          <CardDescription className="text-xs">Timezone memengaruhi render slots (UTC → org timezone). Default Asia/Jakarta.</CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-5">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Nama organisasi</FormLabel>
                    <FormControl>
                      <Input placeholder="FlowBarber Studio" {...field} />
                    </FormControl>
                    <FormDescription className="text-xs">Tampil di header, invoice, dan email.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="timezone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="flex items-center gap-1.5"><Globe className="h-3.5 w-3.5" /> Timezone</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Pilih timezone" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {TIMEZONES.map((tz) => (
                          <SelectItem key={tz} value={tz}>{tz}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormDescription className="text-xs">Semua start_at disimpan UTC, dirender sesuai timezone ini.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Separator />

              <div className="space-y-3">
                <Label className="flex items-center gap-1.5"><ImageIcon className="h-3.5 w-3.5" /> Logo organisasi — Supabase Storage</Label>
                <div className="flex gap-4 items-start">
                  <div className="h-20 w-20 rounded-lg border bg-muted flex items-center justify-center overflow-hidden shrink-0">
                    {logoPreview ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={logoPreview} alt="Logo preview" className="h-full w-full object-cover" />
                    ) : (
                      <span className="text-xs text-muted-foreground">No logo</span>
                    )}
                  </div>
                  <div className="flex-1 space-y-2">
                    <Input type="file" accept="image/*" onChange={handleFileChange} aria-label="Upload logo" />
                    <p className="text-xs text-muted-foreground">PNG/JPG maks 2MB • Storage bucket organisasi • 1GB free tier.</p>
                    {logoFile ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="secondary"
                        onClick={() => logoFile && logoMut.mutate(logoFile)}
                        disabled={logoMut.isPending}
                        className="gap-1.5"
                      >
                        <Upload className="h-4 w-4" /> {logoMut.isPending ? "Mengupload..." : "Upload logo"}
                      </Button>
                    ) : null}
                  </div>
                </div>
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <Button type="submit" disabled={saveMut.isPending} className="gap-1.5">
                  <Save className="h-4 w-4" /> {saveMut.isPending ? "Menyimpan..." : "Simpan pengaturan"}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>

      <Card className="bg-muted/30">
        <CardContent className="p-4 text-xs leading-relaxed text-muted-foreground">
          <p className="font-medium text-foreground">Info Free Tier</p>
          <p>Supabase Storage 1GB — logo disimpan di bucket <code className="rounded bg-background px-1 py-0.5">org-logos</code>. Supabase pooler 6543 transaction mode. Keep-warm via <code className="rounded bg-background px-1 py-0.5">/api/ping</code> tiap 5 menit.</p>
        </CardContent>
      </Card>
    </div>
  );
}
