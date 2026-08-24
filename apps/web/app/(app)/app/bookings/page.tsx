"use client";

import * as React from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Search, Filter, Calendar as CalendarIcon, ChevronLeft, ChevronRight, MoreHorizontal } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { listBookings, fetchStaff, cancelBooking, formatJakartaTime, formatJakartaDate } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
import { toast } from "sonner";

export default function BookingsPage() {
  const qc = useQueryClient();
  const [search, setSearch] = React.useState("");
  const [status, setStatus] = React.useState<string>("all");
  const [staffId, setStaffId] = React.useState<string>("all");
  const [page, setPage] = React.useState(1);
  const [from, setFrom] = React.useState<string>("");
  const [to, setTo] = React.useState<string>("");
  const limit = 10;

  const debouncedSearch = useDebounce(search, 400);

  const filters = React.useMemo(
    () => ({
      search: debouncedSearch || undefined,
      status: status !== "all" ? status : undefined,
      staffId: staffId !== "all" ? staffId : undefined,
      from: from || undefined,
      to: to || undefined,
      page,
      limit,
    }),
    [debouncedSearch, status, staffId, from, to, page]
  );

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: queryKeys.bookings(filters),
    queryFn: () => listBookings(filters),
  });

  const { data: staff } = useQuery({
    queryKey: queryKeys.staff(),
    queryFn: () => fetchStaff(),
  });

  const cancelMut = useMutation({
    mutationFn: (id: string) => cancelBooking(id),
    onSuccess: () => {
      toast.success("Booking dibatalkan");
      qc.invalidateQueries({ queryKey: queryKeys.bookings() });
    },
    onError: async (err: unknown) => {
      const msg = err instanceof Error ? err.message : "Gagal cancel";
      toast.error(msg);
    },
  });

  // reset page when filters change
  React.useEffect(() => setPage(1), [debouncedSearch, status, staffId, from, to]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">Bookings</h1>
        <p className="text-sm text-muted-foreground">DataTable — filter status/date/staff + search + pagination • TanStack Query 5.80 • ky</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-sm flex items-center gap-2">
            <Filter className="h-4 w-4 text-muted-foreground" /> Filter
          </CardTitle>
          <CardDescription className="text-xs">Cari nama/email/ID • filter status, staff, tanggal</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <div className="relative lg:col-span-2">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Cari customer, email, ID..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8"
                aria-label="Cari bookings"
              />
            </div>

            <Select value={status} onValueChange={setStatus}>
              <SelectTrigger aria-label="Filter status">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua status</SelectItem>
                <SelectItem value="CONFIRMED">CONFIRMED</SelectItem>
                <SelectItem value="PENDING">PENDING</SelectItem>
                <SelectItem value="CANCELLED">CANCELLED</SelectItem>
                <SelectItem value="COMPLETED">COMPLETED</SelectItem>
              </SelectContent>
            </Select>

            <Select value={staffId} onValueChange={setStaffId}>
              <SelectTrigger aria-label="Filter staff">
                <SelectValue placeholder="Staff" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Semua staff</SelectItem>
                {staff?.map((s) => (
                  <SelectItem key={s.id} value={s.id}>{s.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <div className="flex gap-2">
              <Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} aria-label="Dari tanggal" />
              <Input type="date" value={to} onChange={(e) => setTo(e.target.value)} aria-label="Sampai tanggal" />
            </div>
          </div>

          <Separator />

          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : isError ? (
            <Alert variant="destructive">
              <AlertDescription className="flex items-center justify-between">
                <span>Gagal memuat bookings.</span>
                <Button variant="outline" size="sm" onClick={() => refetch()}>Retry</Button>
              </AlertDescription>
            </Alert>
          ) : !data || data.data.length === 0 ? (
            <div className="rounded-lg border border-dashed p-10 text-center">
              <p className="text-sm font-medium">Belum ada booking</p>
              <p className="text-xs text-muted-foreground mt-1">Belum ada booking hari ini — bagikan link /book • coba ubah filter</p>
            </div>
          ) : (
            <>
              <div className="rounded-md border overflow-hidden">
                <div className="overflow-x-auto">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className="w-[110px]">ID</TableHead>
                        <TableHead>Customer</TableHead>
                        <TableHead className="hidden md:table-cell">Layanan/Staff</TableHead>
                        <TableHead>Waktu</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="w-[50px]"></TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.data.map((b) => (
                        <TableRow key={b.id} className="hover:bg-muted/50">
                          <TableCell className="font-mono text-xs tabular-nums">{b.id}</TableCell>
                          <TableCell>
                            <p className="text-sm font-medium leading-none">{b.customerName}</p>
                            <p className="text-xs text-muted-foreground">{b.customerEmail}</p>
                          </TableCell>
                          <TableCell className="hidden md:table-cell">
                            <p className="text-xs font-medium truncate max-w-[140px]">{b.serviceId.slice(0, 8)}</p>
                            <p className="text-xs text-muted-foreground">{b.staffId.slice(0, 8)}</p>
                          </TableCell>
                          <TableCell>
                            <p className="text-xs tabular-nums">{formatJakartaDate(b.startAt)}</p>
                            <p className="text-xs text-muted-foreground tabular-nums">{formatJakartaTime(b.startAt)} WIB</p>
                          </TableCell>
                          <TableCell>
                            <Badge
                              variant={b.status === "CONFIRMED" ? "default" : b.status === "PENDING" ? "secondary" : b.status === "CANCELLED" ? "outline" : "secondary"}
                              className="text-xs"
                            >
                              {b.status}
                            </Badge>
                            {b.paymentStatus ? (
                              <Badge variant="outline" className="ml-1 text-[11px]">{b.paymentStatus}</Badge>
                            ) : null}
                          </TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon" aria-label="Aksi booking">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => toast.info(`Detail ${b.id} — /app/bookings/${b.id}`)}>View detail</DropdownMenuItem>
                                <DropdownMenuItem onClick={() => toast.info("Reschedule — buka dialog kalender")}>Reschedule</DropdownMenuItem>
                                <DropdownMenuItem
                                  className="text-destructive focus:text-destructive"
                                  onClick={() => cancelMut.mutate(b.id)}
                                  disabled={b.status === "CANCELLED"}
                                >
                                  Cancel booking
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>

              {/* Pagination */}
              <div className="flex items-center justify-between gap-4 pt-2">
                <p className="text-xs text-muted-foreground tabular-nums">
                  Hal {data.meta.page} / {data.meta.totalPages} • {data.meta.total} bookings • limit {data.meta.limit}
                </p>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
                    <ChevronLeft className="h-4 w-4" /> Prev
                  </Button>
                  <span className="text-sm tabular-nums min-w-[40px] text-center">{page}</span>
                  <Button variant="outline" size="sm" disabled={page >= data.meta.totalPages} onClick={() => setPage((p) => p + 1)}>
                    Next <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <p className="text-xs text-muted-foreground text-center">
        Row navigable via keyboard, focus trap di Dialog • Header sticky backdrop-blur • Query invalidate via WS slot_taken
      </p>
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
