"use client";

import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Users, Eye, Mail, Phone } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { fetchCustomers, listBookings, type Customer } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";

export default function CustomersPage() {
  const [search, setSearch] = React.useState("");
  const debounced = useDebounce(search, 350);
  const [selected, setSelected] = React.useState<Customer | null>(null);

  const { data: customers, isLoading } = useQuery({
    queryKey: queryKeys.customers(debounced),
    queryFn: () => fetchCustomers(debounced || undefined),
  });

  const { data: history, isLoading: histLoading } = useQuery({
    queryKey: queryKeys.bookings({ customerEmail: selected?.email }),
    queryFn: () => listBookings({ search: selected?.email, limit: 10 }),
    enabled: !!selected,
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Customers</h1>
        <p className="text-sm text-muted-foreground">List customer + history • 60 seed • total spent tabular-nums • search</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle className="text-sm flex items-center gap-2">
              <Users className="h-4 w-4 text-muted-foreground" /> Customers — {customers?.length ?? 0}
            </CardTitle>
            <div className="relative w-full sm:w-72">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input placeholder="Cari nama/email..." value={search} onChange={(e) => setSearch(e.target.value)} className="pl-8" aria-label="Cari customer" />
            </div>
          </div>
          <CardDescription className="text-xs">Top loyal Siti 18x • klik Eye untuk history 10 booking terakhir</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}</div>
          ) : !customers || customers.length === 0 ? (
            <div className="rounded-lg border border-dashed p-10 text-center">
              <p className="text-sm text-muted-foreground">Belum ada customer — data kosong.</p>
            </div>
          ) : (
            <div className="rounded-md border overflow-hidden">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Nama</TableHead>
                      <TableHead className="hidden sm:table-cell">Kontak</TableHead>
                      <TableHead className="text-center">Bookings</TableHead>
                      <TableHead className="text-right hidden sm:table-cell">Total</TableHead>
                      <TableHead>Last</TableHead>
                      <TableHead className="w-[50px]"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {customers.map((c) => (
                      <TableRow key={c.id} className="hover:bg-muted/50">
                        <TableCell>
                          <div className="flex items-center gap-2.5">
                            <Avatar className="h-8 w-8 border">
                              <AvatarFallback className="text-xs font-medium bg-muted">
                                {c.name.slice(0, 2).toUpperCase()}
                              </AvatarFallback>
                            </Avatar>
                            <div>
                              <p className="text-sm font-medium leading-none">{c.name}</p>
                              <p className="text-xs text-muted-foreground">{c.email}</p>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className="hidden sm:table-cell">
                          <p className="text-xs flex items-center gap-1.5"><Mail className="h-3 w-3 text-muted-foreground" /> {c.email}</p>
                          <p className="text-xs text-muted-foreground flex items-center gap-1.5"><Phone className="h-3 w-3" /> {c.phone ?? "—"}</p>
                        </TableCell>
                        <TableCell className="text-center">
                          <Badge variant="secondary" className="tabular-nums">{c.bookingsCount}×</Badge>
                        </TableCell>
                        <TableCell className="text-right hidden sm:table-cell tabular-nums text-sm font-medium">
                          {new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(c.totalSpentCents)}
                        </TableCell>
                        <TableCell className="text-xs tabular-nums">{c.lastBookingAt ? new Date(c.lastBookingAt).toLocaleDateString("id-ID") : "—"}</TableCell>
                        <TableCell>
                          <Button variant="ghost" size="icon" aria-label={`History ${c.name}`} onClick={() => setSelected(c)}>
                            <Eye className="h-4 w-4" />
                          </Button>
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

      <Dialog open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>History — {selected?.name}</DialogTitle>
            <DialogDescription className="text-xs">{selected?.email} • {selected?.bookingsCount} bookings • total {selected ? new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(selected.totalSpentCents) : ""}</DialogDescription>
          </DialogHeader>
          <Separator />
          {histLoading ? (
            <div className="space-y-2"><Skeleton className="h-12 w-full" /><Skeleton className="h-12 w-full" /></div>
          ) : !history || history.data.length === 0 ? (
            <p className="text-sm text-muted-foreground">Belum ada booking untuk customer ini.</p>
          ) : (
            <div className="space-y-2 max-h-[320px] overflow-y-auto pr-1">
              {history.data.map((b) => (
                <div key={b.id} className="flex items-center justify-between rounded-md border p-3">
                  <div>
                    <p className="text-sm font-mono tabular-nums">{b.id}</p>
                    <p className="text-xs text-muted-foreground">{new Date(b.startAt).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })} WIB</p>
                  </div>
                  <Badge variant={b.status === "CONFIRMED" ? "default" : b.status === "PENDING" ? "secondary" : "outline"}>{b.status}</Badge>
                </div>
              ))}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function useDebounce<T>(value: T, delay: number) {
  const [debounced, setDebounced] = React.useState(value);
  React.useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(id);
  }, [value, delay]);
  return debounced;
}
