"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import { api, parseApiError } from "@/lib/api";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";

const schema = z.object({
  email: z.string().email("Email tidak valid"),
  password: z.string().min(8, "Password minimal 8 karakter"),
});

type Values = z.infer<typeof schema>;

function LoginInner() {
  const router = useRouter();
  const sp = useSearchParams();
  const next = sp.get("next") ?? "/app";

  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { email: "owner@flowbook.test", password: "password123" },
  });

  const [loading, setLoading] = React.useState(false);

  const onSubmit = async (vals: Values) => {
    setLoading(true);
    try {
      const res = await api.post("auth/login", { json: vals }).json<{ accessToken: string; user?: { id: string } }>();
      if (res.accessToken) {
        localStorage.setItem("flowbook_access", res.accessToken);
        document.cookie = `flowbook_access=${res.accessToken}; path=/; max-age=900; SameSite=Lax`;
        document.cookie = `refresh_token=dummy; path=/; max-age=604800; SameSite=Lax`;
        toast.success("Login berhasil — redirect ke app");
        router.push(next);
        router.refresh();
      } else {
        toast.error("Login gagal — token hilang");
      }
    } catch (err) {
      const p = await parseApiError(err);
      if (p.status === 401) toast.error("Email atau password salah (401)");
      else toast.error(p.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Login — FlowBook App</CardTitle>
        <CardDescription>
          Masuk untuk akses <code className="rounded bg-muted px-1 py-0.5">/app/*</code> — protected via middleware (401 → redirect). Test: owner@flowbook.test / password123
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input type="email" placeholder="owner@flowbook.test" autoComplete="email" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder="••••••••" autoComplete="current-password" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" className="w-full gap-2" disabled={loading}>
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {loading ? "Memproses..." : "Login"}
            </Button>
          </form>
        </Form>

        <Separator className="my-4" />

        <div className="text-xs text-muted-foreground space-y-1">
          <p>
            Demo credentials: <span className="font-mono">owner@flowbook.test / password123</span>
          </p>
          <p>
            Middleware cek cookie <code className="rounded bg-muted px-1 py-0.5">refresh_token</code> atau{" "}
            <code className="rounded bg-muted px-1 py-0.5">flowbook_access</code> — jika 401, redirect ke /login?next=/app.
          </p>
          <p>Toaster menampilkan 422 (validasi) & 409 (slot conflict) otomatis via ky afterResponse.</p>
        </div>

        <div className="mt-4 flex justify-between text-xs">
          <Link href="/" className="underline underline-offset-4 hover:text-foreground">
            ← Beranda
          </Link>
          <Link href="/book" className="underline underline-offset-4 hover:text-foreground">
            Ke /book (public)
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}

export default function LoginPage() {
  return (
    <div className="min-h-screen bg-background flex flex-col">
      <header className="border-b bg-background/80 backdrop-blur sticky top-0 z-30">
        <div className="mx-auto max-w-5xl px-4 py-3 flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2">
            <div className="h-8 w-8 rounded-lg bg-primary text-primary-foreground grid place-items-center text-sm font-semibold">FB</div>
            <span className="text-sm font-semibold tracking-tight">FlowBarber Studio</span>
          </Link>
          <ThemeToggle />
        </div>
      </header>

      <main className="flex-1 flex items-center justify-center p-4">
        <React.Suspense fallback={<div className="w-full max-w-md rounded-lg border bg-card p-6 animate-pulse h-80" />}>
          <LoginInner />
        </React.Suspense>
      </main>
    </div>
  );
}
