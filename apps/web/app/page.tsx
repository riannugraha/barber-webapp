import { ThemeToggle } from "@/components/ThemeToggle";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export default function Page() {
  return (
    <main className="mx-auto max-w-7xl p-6 space-y-6">
      <header className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold tracking-tight">FlowBook</h1>
        <ThemeToggle />
      </header>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <CardDescription className="text-xs font-medium text-muted-foreground">
                KPI {i}
              </CardDescription>
              <CardTitle className="text-3xl font-semibold tabular-nums">
                Rp 142jt
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Badge variant="secondary">+9%</Badge>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Area Chart placeholder</CardTitle>
          <CardDescription>Recharts akan di-mount di sini (T12)</CardDescription>
        </CardHeader>
        <CardContent>
          <Button>Book Now</Button>
        </CardContent>
      </Card>
    </main>
  );
}
