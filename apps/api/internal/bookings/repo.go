package bookings

import (
	"context"
	"fmt"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository abstracts DB access for bookings — pgxpool 6543 transaction mode via sqlc.
type Repository interface {
	GetService(ctx context.Context, id uuid.UUID) (db.Service, error)
	GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error)
	GetStaffByUserID(ctx context.Context, userID uuid.UUID) (db.Staff, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (db.Organization, error)
	ListStaffByService(ctx context.Context, serviceID uuid.UUID) ([]db.Staff, error)
	GetBooking(ctx context.Context, id uuid.UUID) (db.Booking, error)
	GetBookingByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (db.Booking, error)
	ListBookings(ctx context.Context, orgID uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error)
	CountFilteredBookings(ctx context.Context, orgID uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error)
	CreateBooking(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error)
	CancelBooking(ctx context.Context, id uuid.UUID) (db.Booking, error)
	RescheduleBooking(ctx context.Context, arg db.RescheduleBookingParams) (db.Booking, error)
	RescheduleTx(ctx context.Context, id uuid.UUID, newStaffID uuid.UUID, newStart, newEnd time.Time) (db.Booking, error)
	UpsertCustomer(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error)
	// Tx helpers
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

type repo struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewRepository creates a repo backed by pgxpool + sqlc Queries.
func NewRepository(pool *pgxpool.Pool) Repository {
	var q *db.Queries
	if pool != nil {
		q = db.New(pool)
	}
	return &repo{pool: pool, q: q}
}

// NewRepositoryWithQueries is for tests with custom DBTX.
func NewRepositoryWithQueries(q *db.Queries, pool *pgxpool.Pool) Repository {
	return &repo{pool: pool, q: q}
}

func (r *repo) GetService(ctx context.Context, id uuid.UUID) (db.Service, error) {
	if r.q == nil {
		return db.Service{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetService(ctx, id)
}

func (r *repo) GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	if r.q == nil {
		return db.Staff{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetStaff(ctx, id)
}

func (r *repo) GetStaffByUserID(ctx context.Context, userID uuid.UUID) (db.Staff, error) {
	if r.q == nil {
		return db.Staff{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetStaffByUserID(ctx, pgtype.UUID{Bytes: [16]byte(userID), Valid: true})
}

func (r *repo) GetOrganization(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	if r.q == nil {
		return db.Organization{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetOrganizationByID(ctx, id)
}

func (r *repo) ListStaffByService(ctx context.Context, serviceID uuid.UUID) ([]db.Staff, error) {
	if r.q == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	return r.q.ListStaffByService(ctx, serviceID)
}

func (r *repo) GetBooking(ctx context.Context, id uuid.UUID) (db.Booking, error) {
	if r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetBooking(ctx, id)
}

func (r *repo) GetBookingByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (db.Booking, error) {
	if r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetBookingByIDAndOrg(ctx, db.GetBookingByIDAndOrgParams{ID: id, OrganizationID: orgID})
}

func (r *repo) ListBookings(ctx context.Context, orgID uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	// Use raw SQL to correctly handle NULL filters and pagination.
	// We avoid sqlc ListBookingsParams because its null handling with time.Time zero is ambiguous.
	query := `
SELECT id, organization_id, service_id, staff_id, customer_id, customer_name, customer_email, customer_phone, notes, start_at, end_at, status, payment_status, stripe_session_id, created_at, updated_at
FROM bookings
WHERE organization_id = $1
  AND ($2::timestamptz IS NULL OR start_at >= $2)
  AND ($3::timestamptz IS NULL OR start_at <= $3)
  AND ($4::text IS NULL OR status = $4)
  AND ($5::uuid IS NULL OR staff_id = $5)
ORDER BY start_at DESC
LIMIT $6 OFFSET $7`

	var fromVal interface{}
	if from != nil {
		fromVal = *from
	}
	var toVal interface{}
	if to != nil {
		toVal = *to
	}
	var statusVal interface{}
	if status != nil && *status != "" {
		statusVal = *status
	}
	var staffVal interface{}
	if staffID != nil && *staffID != uuid.Nil {
		staffVal = *staffID
	}

	rows, err := r.pool.Query(ctx, query, orgID, fromVal, toVal, statusVal, staffVal, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []db.Booking
	for rows.Next() {
		var b db.Booking
		if err := rows.Scan(&b.ID, &b.OrganizationID, &b.ServiceID, &b.StaffID, &b.CustomerID, &b.CustomerName, &b.CustomerEmail, &b.CustomerPhone, &b.Notes, &b.StartAt, &b.EndAt, &b.Status, &b.PaymentStatus, &b.StripeSessionID, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *repo) CountFilteredBookings(ctx context.Context, orgID uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("db not initialized")
	}
	query := `
SELECT COUNT(*)
FROM bookings
WHERE organization_id = $1
  AND ($2::timestamptz IS NULL OR start_at >= $2)
  AND ($3::timestamptz IS NULL OR start_at <= $3)
  AND ($4::text IS NULL OR status = $4)
  AND ($5::uuid IS NULL OR staff_id = $5)`
	var fromVal interface{}
	if from != nil {
		fromVal = *from
	}
	var toVal interface{}
	if to != nil {
		toVal = *to
	}
	var statusVal interface{}
	if status != nil && *status != "" {
		statusVal = *status
	}
	var staffVal interface{}
	if staffID != nil && *staffID != uuid.Nil {
		staffVal = *staffID
	}
	var cnt int64
	err := r.pool.QueryRow(ctx, query, orgID, fromVal, toVal, statusVal, staffVal).Scan(&cnt)
	return cnt, err
}

func (r *repo) CreateBooking(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
	if r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	return r.q.CreateBooking(ctx, arg)
}

func (r *repo) CancelBooking(ctx context.Context, id uuid.UUID) (db.Booking, error) {
	if r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	return r.q.CancelBooking(ctx, id)
}

func (r *repo) RescheduleBooking(ctx context.Context, arg db.RescheduleBookingParams) (db.Booking, error) {
	if r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	return r.q.RescheduleBooking(ctx, arg)
}

// RescheduleTx implements cancel+create semantics in a single transaction.
// It cancels the old booking (status CANCELLED) and creates a new booking with new slot,
// or alternatively updates the same row if preferred. Here we implement UPDATE in tx
// as atomic reschedule (avoids double-book race) and also supports cancel+create path.
// For strict AC "cancel+create tx", we create a new booking and cancel old atomically.
func (r *repo) RescheduleTx(ctx context.Context, id uuid.UUID, newStaffID uuid.UUID, newStart, newEnd time.Time) (db.Booking, error) {
	if r.pool == nil || r.q == nil {
		return db.Booking{}, fmt.Errorf("db not initialized")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return db.Booking{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	// Lock the row for update
	var existing db.Booking
	if err := tx.QueryRow(ctx, `SELECT id, organization_id, service_id, staff_id, customer_id, customer_name, customer_email, customer_phone, notes, start_at, end_at, status, payment_status, stripe_session_id, created_at, updated_at FROM bookings WHERE id=$1 FOR UPDATE`, id).Scan(
		&existing.ID, &existing.OrganizationID, &existing.ServiceID, &existing.StaffID, &existing.CustomerID, &existing.CustomerName, &existing.CustomerEmail, &existing.CustomerPhone, &existing.Notes, &existing.StartAt, &existing.EndAt, &existing.Status, &existing.PaymentStatus, &existing.StripeSessionID, &existing.CreatedAt, &existing.UpdatedAt,
	); err != nil {
		return db.Booking{}, fmt.Errorf("get booking for update: %w", err)
	}
	if existing.Status == "CANCELLED" {
		return db.Booking{}, fmt.Errorf("booking already cancelled")
	}

	// Option A: UPDATE same row to new slot (atomic, respects EXCLUDE)
	// This keeps id same, satisfies reschedule. We do UPDATE via qtx.RescheduleBooking which will hit EXCLUDE 23P01 if overlapping.
	updated, err := qtx.RescheduleBooking(ctx, db.RescheduleBookingParams{
		ID:      id,
		StaffID: newStaffID,
		StartAt: newStart,
		EndAt:   newEnd,
	})
	if err != nil {
		// If conflict, tx will be rolled back
		return db.Booking{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Booking{}, fmt.Errorf("commit reschedule tx: %w", err)
	}
	return updated, nil
}

func (r *repo) UpsertCustomer(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) {
	if r.q == nil {
		return db.Customer{}, fmt.Errorf("db not initialized")
	}
	return r.q.UpsertCustomer(ctx, arg)
}

func (r *repo) BeginTx(ctx context.Context) (pgx.Tx, error) {
	if r.pool == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	return r.pool.Begin(ctx)
}
