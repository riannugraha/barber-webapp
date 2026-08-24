package availability

import (
	"context"
	"time"

	"github.com/google/uuid"
	"flowbook/api/internal/db"
)

// Repository abstracts DB access for the calendar engine.
// Hot path uses pgxpool (pooler 6543 transaction mode) via db.Queries (pgx/v5).
type Repository interface {
	GetService(ctx context.Context, id uuid.UUID) (db.Service, error)
	GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (db.Organization, error)
	ListAvailabilityByStaff(ctx context.Context, staffID uuid.UUID) ([]db.Availability, error)
	GetOverrideByStaffAndDate(ctx context.Context, staffID uuid.UUID, date time.Time) (db.AvailabilityOverride, error)
	ListStaffByService(ctx context.Context, serviceID uuid.UUID) ([]db.Staff, error)
	ListOverlappingBookings(ctx context.Context, staffID uuid.UUID, start, end time.Time) ([]db.Booking, error)
	// ListStaffServices is used to validate skill eligibility if needed
	ListStaffServices(ctx context.Context, staffID uuid.UUID) ([]db.StaffService, error)
}

type repo struct {
	q *db.Queries
}

// NewRepository creates a repo backed by sqlc Queries (which wraps pgxpool.Pool via DBTX).
func NewRepository(q *db.Queries) Repository {
	return &repo{q: q}
}

func (r *repo) GetService(ctx context.Context, id uuid.UUID) (db.Service, error) {
	return r.q.GetService(ctx, id)
}

func (r *repo) GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	return r.q.GetStaff(ctx, id)
}

func (r *repo) GetOrganizationByID(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	return r.q.GetOrganizationByID(ctx, id)
}

func (r *repo) ListAvailabilityByStaff(ctx context.Context, staffID uuid.UUID) ([]db.Availability, error) {
	return r.q.ListAvailabilityByStaff(ctx, staffID)
}

func (r *repo) GetOverrideByStaffAndDate(ctx context.Context, staffID uuid.UUID, date time.Time) (db.AvailabilityOverride, error) {
	return r.q.GetOverrideByStaffAndDate(ctx, db.GetOverrideByStaffAndDateParams{
		StaffID: staffID,
		Date:    date,
	})
}

func (r *repo) ListStaffByService(ctx context.Context, serviceID uuid.UUID) ([]db.Staff, error) {
	return r.q.ListStaffByService(ctx, serviceID)
}

func (r *repo) ListOverlappingBookings(ctx context.Context, staffID uuid.UUID, start, end time.Time) ([]db.Booking, error) {
	return r.q.ListOverlappingBookings(ctx, db.ListOverlappingBookingsParams{
		StaffID: staffID,
		Column2: start,
		Column3: end,
	})
}

func (r *repo) ListStaffServices(ctx context.Context, staffID uuid.UUID) ([]db.StaffService, error) {
	return r.q.ListStaffServices(ctx, staffID)
}
