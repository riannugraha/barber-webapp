import type { Metadata } from "next";
import { AppShell } from "@/components/app/AppShell";
import { QueryProvider } from "@/components/providers";

export const metadata: Metadata = {
  title: "FlowBook — App",
};

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <QueryProvider>
      <AppShell>{children}</AppShell>
    </QueryProvider>
  );
}
