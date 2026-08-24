"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { CalendarDays, ClipboardList, Scissors, Users, Settings, Moon, Sun, LayoutDashboard, Plus } from "lucide-react";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { useTheme } from "next-themes";

export function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const router = useRouter();
  const { theme, setTheme } = useTheme();

  const run = (fn: () => void) => {
    onOpenChange(false);
    fn();
  };

  return (
    <CommandDialog open={open} onOpenChange={onOpenChange}>
      <CommandInput placeholder="Cari halaman atau aksi... (⌘K)" />
      <CommandList>
        <CommandEmpty>Tidak ditemukan.</CommandEmpty>
        <CommandGroup heading="Navigasi">
          <CommandItem onSelect={() => run(() => router.push("/app"))}>
            <LayoutDashboard className="mr-2 h-4 w-4" /> Dashboard
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/calendar"))}>
            <CalendarDays className="mr-2 h-4 w-4" /> Calendar — week 07-21
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/bookings"))}>
            <ClipboardList className="mr-2 h-4 w-4" /> Bookings — DataTable filter
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/services"))}>
            <Scissors className="mr-2 h-4 w-4" /> Services — CRUD
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/staff"))}>
            <Users className="mr-2 h-4 w-4" /> Staff — availability
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/customers"))}>
            <Users className="mr-2 h-4 w-4" /> Customers — list
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/settings"))}>
            <Settings className="mr-2 h-4 w-4" /> Settings — timezone/logo
          </CommandItem>
        </CommandGroup>
        <CommandSeparator />
        <CommandGroup heading="Aksi">
          <CommandItem onSelect={() => run(() => router.push("/app/services"))}>
            <Plus className="mr-2 h-4 w-4" /> Add Service
          </CommandItem>
          <CommandItem onSelect={() => run(() => router.push("/app/bookings"))}>Go to Bookings</CommandItem>
          <CommandItem
            onSelect={() => run(() => setTheme(theme === "dark" ? "light" : "dark"))}
          >
            {theme === "dark" ? <Sun className="mr-2 h-4 w-4" /> : <Moon className="mr-2 h-4 w-4" />}
            Toggle theme — {theme === "dark" ? "Light" : "Dark"}
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
