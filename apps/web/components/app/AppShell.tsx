"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  LayoutDashboard,
  CalendarDays,
  ClipboardList,
  Scissors,
  UsersRound,
  Users,
  Settings,
  Menu,
  PanelLeftClose,
  PanelLeftOpen,
  Search,
  LogOut,
  Sun,
  Moon,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Separator } from "@/components/ui/separator";
import { ThemeToggle } from "@/components/ThemeToggle";
import { useSidebarStore } from "@/lib/store";
import { cn } from "@/lib/utils";
import { CommandPalette } from "./CommandPalette";
import { useTheme } from "next-themes";

const NAV = [
  { href: "/app", label: "Dashboard", icon: LayoutDashboard },
  { href: "/app/calendar", label: "Calendar", icon: CalendarDays },
  { href: "/app/bookings", label: "Bookings", icon: ClipboardList },
  { href: "/app/services", label: "Services", icon: Scissors },
  { href: "/app/staff", label: "Staff", icon: UsersRound },
  { href: "/app/customers", label: "Customers", icon: Users },
  { href: "/app/settings", label: "Settings", icon: Settings },
] as const;

function SidebarContent({ collapsed, onNavigate }: { collapsed: boolean; onNavigate?: () => void }) {
  const pathname = usePathname();
  return (
    <div className="flex h-full flex-col">
      <div className={cn("flex items-center gap-2.5 px-3 py-4", collapsed && "justify-center px-2")}>
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground font-semibold text-sm">
          FB
        </div>
        {!collapsed && (
          <div className="min-w-0 flex-1">
            <p className="text-sm font-semibold leading-none tracking-tight truncate">FlowBarber Studio</p>
            <p className="text-[11px] text-muted-foreground truncate">Booking & Scheduling</p>
          </div>
        )}
      </div>
      <Separator />
      <nav aria-label="App navigation" className="flex-1 space-y-1 p-2">
        {NAV.map((item) => {
          const active = pathname === item.href || (item.href !== "/app" && pathname.startsWith(item.href));
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={onNavigate}
              className={cn(
                "flex items-center gap-3 rounded-md px-2.5 py-2 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                active ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
                collapsed && "justify-center px-2"
              )}
              aria-current={active ? "page" : undefined}
              title={collapsed ? item.label : undefined}
            >
              <Icon className="h-[18px] w-[18px] shrink-0" aria-hidden="true" />
              {!collapsed && <span className="truncate">{item.label}</span>}
            </Link>
          );
        })}
      </nav>
      <div className="p-2">
        <Separator className="mb-2" />
        {!collapsed ? (
          <div className="rounded-lg bg-muted p-3">
            <p className="text-xs font-medium">Butuh bantuan?</p>
            <p className="text-xs text-muted-foreground leading-relaxed mt-1">
              Lihat <Link href="/book" className="underline underline-offset-4">/book</Link> untuk flow customer.
            </p>
          </div>
        ) : null}
        <p className={cn("mt-2 text-[11px] text-muted-foreground", collapsed && "text-center")}>
          {!collapsed ? "OKLCH violet 260 • Light 0.62 / Dark 0.68" : "260"}
        </p>
      </div>
    </div>
  );
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const { collapsed, toggle } = useSidebarStore();
  const [open, setOpen] = React.useState(false);
  const [cmdOpen, setCmdOpen] = React.useState(false);
  const router = useRouter();
  const { theme, setTheme } = useTheme();

  // Mirror localStorage token to cookie for middleware
  React.useEffect(() => {
    if (typeof window === "undefined") return;
    const tok = localStorage.getItem("flowbook_access");
    if (tok) document.cookie = `flowbook_access=${tok}; path=/; max-age=900; SameSite=Lax`;
  }, []);

  // ⌘K handler
  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || e.key === "/") {
        e.preventDefault();
        setCmdOpen((o) => !o);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const handleLogout = () => {
    if (typeof window !== "undefined") {
      localStorage.removeItem("flowbook_access");
      document.cookie = "flowbook_access=; path=/; max-age=0";
      document.cookie = "refresh_token=; path=/; max-age=0";
    }
    router.push("/login");
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Desktop sidebar */}
      <aside
        className={cn(
          "hidden md:fixed md:inset-y-0 md:flex md:flex-col border-r bg-card transition-all duration-300",
          collapsed ? "md:w-14" : "md:w-64"
        )}
        aria-label="Sidebar"
      >
        <SidebarContent collapsed={collapsed} />
      </aside>

      {/* Mobile drawer */}
      <Sheet open={open} onOpenChange={setOpen}>
        <div
          className={cn(
            "flex min-h-screen flex-col transition-all duration-300",
            collapsed ? "md:pl-14" : "md:pl-64"
          )}
        >
          {/* Header sticky backdrop-blur */}
          <header className="sticky top-0 z-30 flex h-14 items-center gap-2 border-b bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 px-3 sm:px-4">
            {/* Mobile menu */}
            <SheetTrigger asChild className="md:hidden">
              <Button variant="ghost" size="icon" aria-label="Buka menu">
                <Menu className="h-5 w-5" />
              </Button>
            </SheetTrigger>

            {/* Desktop collapse toggle */}
            <Button
              variant="ghost"
              size="icon"
              className="hidden md:inline-flex"
              onClick={toggle}
              aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            >
              {collapsed ? <PanelLeftOpen className="h-5 w-5" /> : <PanelLeftClose className="h-5 w-5" />}
            </Button>

            <Separator orientation="vertical" className="hidden md:block h-6 mx-1" />

            {/* Command K trigger */}
            <Button
              variant="outline"
              className="hidden sm:inline-flex items-center gap-2 text-sm text-muted-foreground justify-start flex-1 max-w-md"
              onClick={() => setCmdOpen(true)}
              aria-label="Buka Command palette"
            >
              <Search className="h-4 w-4" />
              <span className="flex-1 text-left">Cari...</span>
              <kbd className="hidden sm:inline-flex items-center gap-1 rounded border bg-muted px-1.5 py-0.5 text-xs font-mono">
                ⌘K
              </kbd>
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="sm:hidden"
              onClick={() => setCmdOpen(true)}
              aria-label="Command"
            >
              <Search className="h-5 w-5" />
            </Button>

            <div className="ml-auto flex items-center gap-1 sm:gap-2">
              <ThemeToggle />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="rounded-full" aria-label="User menu">
                    <Avatar className="h-8 w-8 border">
                      <AvatarImage src="" alt="Avatar owner" />
                      <AvatarFallback className="bg-primary text-primary-foreground text-xs font-semibold">OW</AvatarFallback>
                    </Avatar>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuLabel>
                    <p className="text-sm font-medium">Owner</p>
                    <p className="text-xs font-normal text-muted-foreground">owner@flowbook.test</p>
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => router.push("/app/settings")}>
                    <Settings className="mr-2 h-4 w-4" /> Settings
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
                  >
                    {theme === "dark" ? <Sun className="mr-2 h-4 w-4" /> : <Moon className="mr-2 h-4 w-4" />}
                    Toggle theme
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => router.push("/app/services")}>
                    <Plus className="mr-2 h-4 w-4" /> Add Service
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={handleLogout} className="text-destructive focus:text-destructive">
                    <LogOut className="mr-2 h-4 w-4" /> Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </header>

          {/* Content */}
          <main className="flex-1">
            <div className="mx-auto max-w-7xl p-4 sm:p-6 space-y-6">{children}</div>
          </main>

          <footer className="border-t bg-card/50">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 py-4 flex flex-col sm:flex-row items-center justify-between gap-2 text-xs text-muted-foreground">
              <span>© 2026 FlowBarber Studio • Dashboard • Asia/Jakarta</span>
              <span className="tabular-nums">v0.1.0 • OKLCH violet 260</span>
            </div>
          </footer>
        </div>

        <SheetContent side="left" className="p-0 w-72">
          <SheetHeader className="sr-only">
            <SheetTitle>Menu</SheetTitle>
          </SheetHeader>
          <SidebarContent collapsed={false} onNavigate={() => setOpen(false)} />
        </SheetContent>
      </Sheet>

      <CommandPalette open={cmdOpen} onOpenChange={setCmdOpen} />
    </div>
  );
}
