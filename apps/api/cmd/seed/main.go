package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"flowbook/api/internal/seed"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	minimal := flag.Bool("minimal", false, "seed only 10 rows for E2E beforeEach (150ms)")
	// also allow --minimal as alias (flag handles -minimal, --minimal is parsed as -minimal with extra dash? go flag supports '-' and '--')
	flag.Parse()

	// Support positional "minimal" arg for convenience: go run ./cmd/seed minimal
	for _, a := range flag.Args() {
		if a == "minimal" || a == "--minimal" || a == "-minimal" {
			*minimal = true
		}
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required — set postgres://...:6543/postgres?pgbouncer=true")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to create pgxpool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("db ping failed", "error", err)
		// continue anyway — seed may still attempt
	} else {
		slog.Info("db connected", "pooler", "6543")
	}

	start := time.Now()
	var inserted int
	if *minimal {
		slog.Info("seeding minimal (10 bookings) for E2E...")
		// For minimal as standalone, we clean bookings/payments/customers first to ensure idempotent 10 rows
		// But keep services/staff — SeedMinimal will ensure they exist
		// Use truncated minimal path: TRUNCATE bookings,payments,customers then seed 10
		// We do it here for go run --minimal idempotency, mirroring testhelpers
		// If pool can truncate, do it; otherwise seed will handle
		if _, err := pool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`); err != nil {
			slog.Warn("truncate before minimal failed (likely first run, ignoring)", "error", err)
		}
		inserted, err = seed.SeedMinimal(ctx, pool)
		if err != nil {
			slog.Error("seed minimal failed", "error", err)
			os.Exit(1)
		}
		slog.Info("seed minimal completed", "bookings", inserted, "elapsed", time.Since(start).String())
		// Verify 10 rows for AC
		if inserted < 10 {
			slog.Warn("minimal seed inserted less than 10 — check logic", "inserted", inserted)
		}
	} else {
		slog.Info("seeding full 2025-11-01 → 2026-08-24 ~1.500 bookings (weekend/musiman weighted)...")
		inserted, err = seed.SeedFull(ctx, pool)
		if err != nil {
			slog.Error("seed full failed", "error", err)
			os.Exit(1)
		}
		slog.Info("seed full completed", "bookings", inserted, "elapsed", time.Since(start).String())
		if inserted < 1300 || inserted > 1700 {
			slog.Warn("full seed count outside expected ~1.500 range", "inserted", inserted, "expected", "~1500")
		}
	}

	// Summary for tester
	var orgCount, svcCount, staffCount, custCount, bookingCount int64
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&orgCount)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM services`).Scan(&svcCount)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM staff`).Scan(&staffCount)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&custCount)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings`).Scan(&bookingCount)
	slog.Info("seed summary",
		"organizations", orgCount,
		"services", svcCount,
		"staff", staffCount,
		"customers", custCount,
		"bookings", bookingCount,
		"minimal", *minimal,
	)
}
