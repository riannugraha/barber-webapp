import { NextResponse } from "next/server";

// Vercel Cron every 5 minutes — keeps Koyeb Eco from cold start and Supabase from pausing.
// vercel.json: { "crons": [{ "path": "/api/ping", "schedule": "*/5 * * * *" }] }
// This route proxies to Go API GET /health (pooler 6543) so both Web and API stay warm.
export const dynamic = "force-dynamic";
export const revalidate = 0;

export async function GET() {
  // Prefer server-side API_URL, fallback to public URL, then local dev
  const raw =
    process.env.API_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080/api/v1";

  // Normalize: if raw ends with /api/v1, strip to base for /health (Koyeb expects GET /health at root)
  // Otherwise append /health
  let healthUrl: string;
  if (raw.includes("/api/v1")) {
    healthUrl = raw.replace(/\/api\/v1\/?$/, "") + "/health";
  } else {
    healthUrl = raw.replace(/\/$/, "") + "/health";
  }
  // Also try /api/v1/health as fallback
  const altHealthUrl = raw.replace(/\/$/, "") + "/health";

  const started = Date.now();
  try {
    // Try primary /health first (Koyeb root)
    let res = await fetch(healthUrl, {
      method: "GET",
      cache: "no-store",
      next: { revalidate: 0 },
    });

    // Fallback to /api/v1/health if first fails
    if (!res.ok && altHealthUrl !== healthUrl) {
      res = await fetch(altHealthUrl, { method: "GET", cache: "no-store" });
    }

    const elapsed = Date.now() - started;
    let data: unknown = null;
    try {
      data = await res.json();
    } catch {
      data = { raw: await res.text().catch(() => null) };
    }

    return NextResponse.json(
      {
        status: "ok",
        ping: "pong",
        api: healthUrl,
        apiStatus: res.status,
        apiData: data,
        elapsedMs: elapsed,
        timestamp: new Date().toISOString(),
      },
      { status: 200, headers: { "cache-control": "no-store" } },
    );
  } catch (err) {
    const elapsed = Date.now() - started;
    return NextResponse.json(
      {
        status: "error",
        ping: "failed",
        api: healthUrl,
        error: err instanceof Error ? err.message : String(err),
        elapsedMs: elapsed,
        timestamp: new Date().toISOString(),
      },
      { status: 200, headers: { "cache-control": "no-store" } },
    );
  }
}
