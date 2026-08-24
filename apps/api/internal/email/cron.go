package email

import (
	"context"
	"log/slog"
	"time"

	"flowbook/api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StartReminderCron starts a Go cron loop every 15m to send Reminder H-1.
// It queries bookings where start_at BETWEEN now+60m and now+75m and status IN ('PENDING','CONFIRMED') but only sends for CONFIRMED.
// H-1 window 60-75 ensures each booking is hit exactly once (15m window sliding).
// It runs immediately on start, then every 15m. Caller should run it as `go StartReminderCron(ctx, pool, client)`.
// Context cancellation stops the loop. Pool may be nil (no-op).
func StartReminderCron(ctx context.Context, pool *pgxpool.Pool, client *Client) {
	if pool == nil || client == nil {
		slog.Info("email: reminder cron disabled (no pool or no client)")
		return
	}
	slog.Info("email: reminder cron started", "interval", "15m", "window", "60-75m", "query", "start_at BETWEEN now+60m AND now+75m")

	// Run once after 10s startup delay to allow DB ready, then ticker
	initialDelay := 10 * time.Second
	select {
	case <-time.After(initialDelay):
		runReminderTick(ctx, pool, client)
	case <-ctx.Done():
		slog.Info("email: reminder cron stopped before first tick", "reason", ctx.Err())
		return
	}

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("email: reminder cron stopped", "reason", ctx.Err())
			return
		case t := <-ticker.C:
			slog.Info("email: reminder cron tick", "at", t.Format(time.RFC3339))
			runReminderTick(ctx, pool, client)
		}
	}
}

// runReminderTick executes one cron iteration.
func runReminderTick(ctx context.Context, pool *pgxpool.Pool, client *Client) {
	// Use a bounded context for DB query + sends
	tickCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Query upcoming bookings H-1
	// Note: store UTC, so NOW() is UTC; start_at is timestamptz UTC — direct compare works.
	// We join services/staff/organizations to render template without N+1, but for simplicity fetch bookings then lookup per booking (small window: <~20 bookings per tick).
	rows, err := pool.Query(tickCtx, `
		SELECT b.id, b.organization_id, b.service_id, b.staff_id, b.customer_id, b.customer_name, b.customer_email, b.customer_phone, b.notes, b.start_at, b.end_at, b.status, b.payment_status, b.stripe_session_id, b.created_at, b.updated_at
		FROM bookings b
		WHERE b.status = 'CONFIRMED'
		  AND b.start_at BETWEEN NOW() + INTERVAL '60 minutes' AND NOW() + INTERVAL '75 minutes'
		ORDER BY b.start_at ASC
		LIMIT 100
	`)
	if err != nil {
		slog.Error("email: reminder cron query failed", "error", err)
		return
	}
	defer rows.Close()

	type bookingRow struct {
		b db.Booking
	}
	var toSend []db.Booking
	for rows.Next() {
		var b db.Booking
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.ServiceID, &b.StaffID, &b.CustomerID, &b.CustomerName, &b.CustomerEmail, &b.CustomerPhone, &b.Notes, &b.StartAt, &b.EndAt, &b.Status, &b.PaymentStatus, &b.StripeSessionID, &b.CreatedAt, &b.UpdatedAt); err != nil {
			slog.Error("email: reminder scan failed", "error", err)
			continue
		}
		toSend = append(toSend, b)
	}
	if err := rows.Err(); err != nil {
		slog.Error("email: reminder rows error", "error", err)
	}
	if len(toSend) == 0 {
		slog.Info("email: reminder cron — no bookings in H-1 window")
		return
	}

	slog.Info("email: reminder cron — found bookings", "count", len(toSend))

	// For each booking, fetch service/staff/org and send
	q := db.New(pool)
	for _, b := range toSend {
		// Per-booking context with timeout
		bCtx, bCancel := context.WithTimeout(tickCtx, 10*time.Second)
		svc, err := q.GetService(bCtx, b.ServiceID)
		if err != nil {
			slog.Error("email: reminder get service failed", "bookingId", b.ID.String(), "error", err)
			bCancel()
			continue
		}
		st, err := q.GetStaff(bCtx, b.StaffID)
		if err != nil {
			slog.Error("email: reminder get staff failed", "bookingId", b.ID.String(), "error", err)
			bCancel()
			continue
		}
		org, err := q.GetOrganizationByID(bCtx, b.OrganizationID)
		if err != nil {
			// fallback to default org
			org = db.Organization{ID: b.OrganizationID, Name: "FlowBarber Studio", Timezone: "Asia/Jakarta"}
		}
		// Send reminder — best effort, log error but continue
		if err := client.SendReminder(bCtx, b, svc, st, org); err != nil {
			slog.Error("email: reminder send failed", "bookingId", b.ID.String(), "to", b.CustomerEmail, "error", err)
		} else {
			slog.Info("email: reminder sent", "bookingId", b.ID.String(), "to", b.CustomerEmail, "startAt", b.StartAt.Format(time.RFC3339))
		}
		bCancel()
		// Small delay to avoid Resend rate limit (2 req/s free tier)
		select {
		case <-time.After(300 * time.Millisecond):
		case <-tickCtx.Done():
			slog.Info("email: reminder tick cancelled during sends")
			return
		}
	}
}

// RunReminderOnce is exported for tests / manual trigger — runs single tick synchronously and returns count.
func RunReminderOnce(ctx context.Context, pool *pgxpool.Pool, client *Client) (int, error) {
	if pool == nil {
		return 0, nil
	}
	// Reuse query logic but return count
	rows, err := pool.Query(ctx, `
		SELECT b.id, b.organization_id, b.service_id, b.staff_id, b.customer_id, b.customer_name, b.customer_email, b.customer_phone, b.notes, b.start_at, b.end_at, b.status, b.payment_status, b.stripe_session_id, b.created_at, b.updated_at
		FROM bookings b
		WHERE b.status = 'CONFIRMED'
		  AND b.start_at BETWEEN NOW() + INTERVAL '60 minutes' AND NOW() + INTERVAL '75 minutes'
		ORDER BY b.start_at ASC
		LIMIT 100
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		var b db.Booking
		_ = rows.Scan(&b.ID, &b.OrganizationID, &b.ServiceID, &b.StaffID, &b.CustomerID, &b.CustomerName, &b.CustomerEmail, &b.CustomerPhone, &b.Notes, &b.StartAt, &b.EndAt, &b.Status, &b.PaymentStatus, &b.StripeSessionID, &b.CreatedAt, &b.UpdatedAt)
	}
	return count, rows.Err()
}
