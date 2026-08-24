package bookings

import (
	"context"
	"testing"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetAndListMethods(t *testing.T) {
	require.NotNil(t, testPool)
	ctx := context.Background()
	orgID, serviceID, staffID := setupTestData(t)

	repo := NewRepository(testPool)
	// GetService
	svc, err := repo.GetService(ctx, serviceID)
	require.NoError(t, err)
	assert.Equal(t, serviceID, svc.ID)
	_, err = repo.GetService(ctx, uuid.New())
	assert.Error(t, err)

	// GetStaff
	st, err := repo.GetStaff(ctx, staffID)
	require.NoError(t, err)
	assert.Equal(t, staffID, st.ID)
	_, err = repo.GetStaff(ctx, uuid.New())
	assert.Error(t, err)

	// GetOrganization
	org, err := repo.GetOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, orgID, org.ID)

	// ListStaffByService
	list, err := repo.ListStaffByService(ctx, serviceID)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// UpsertCustomer
	cust, err := repo.UpsertCustomer(ctx, db.UpsertCustomerParams{OrganizationID: orgID, Email: "newcust@test.com", Name: "New Cust", Phone: pgtype.Text{Valid: false}})
	require.NoError(t, err)
	assert.Equal(t, "newcust@test.com", cust.Email)
	// upsert again same email should return same or updated
	cust2, err := repo.UpsertCustomer(ctx, db.UpsertCustomerParams{OrganizationID: orgID, Email: "newcust@test.com", Name: "New Cust Updated", Phone: pgtype.Text{String: "0812", Valid: true}})
	require.NoError(t, err)
	assert.Equal(t, cust.ID, cust2.ID)

	// CreateBooking via repo
	start := time.Now().Add(48 * time.Hour).UTC()
	end := start.Add(40 * time.Minute)
	b, err := repo.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID, ServiceID: serviceID, StaffID: staffID,
		CustomerID: pgtype.UUID{Bytes: cust.ID, Valid: true},
		CustomerName: "RepoTest", CustomerEmail: "repotest@test.com",
		StartAt: start, EndAt: end, Status: "CONFIRMED", PaymentStatus: "PAID",
		CustomerPhone: pgtype.Text{Valid: false}, Notes: pgtype.Text{Valid: false}, StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", b.Status)

	// GetBooking
	gb, err := repo.GetBooking(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, gb.ID)

	// GetBookingByIDAndOrg
	gb2, err := repo.GetBookingByIDAndOrg(ctx, b.ID, orgID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, gb2.ID)

	// CountFilteredBookings
	cnt, err := repo.CountFilteredBookings(ctx, orgID, nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cnt)

	// ListBookings
	listB, err := repo.ListBookings(ctx, orgID, nil, nil, nil, nil, 10, 0)
	require.NoError(t, err)
	assert.Len(t, listB, 1)

	// Filter by status
	status := "CONFIRMED"
	cnt2, err := repo.CountFilteredBookings(ctx, orgID, nil, nil, &status, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), cnt2)
	status2 := "CANCELLED"
	cnt3, err := repo.CountFilteredBookings(ctx, orgID, nil, nil, &status2, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), cnt3)

	// CancelBooking
	cb, err := repo.CancelBooking(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", cb.Status)

	// RescheduleBooking (direct)
	// Need a new booking to reschedule
	start2 := time.Now().Add(72 * time.Hour).UTC()
	end2 := start2.Add(40 * time.Minute)
	b2, err := repo.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID, ServiceID: serviceID, StaffID: staffID,
		CustomerID: pgtype.UUID{Valid: false}, CustomerName: "Resched", CustomerEmail: "resched@test.com",
		StartAt: start2, EndAt: end2, Status: "CONFIRMED", PaymentStatus: "PAID",
	})
	require.NoError(t, err)
	newStart := start2.Add(2 * time.Hour)
	newEnd := newStart.Add(40 * time.Minute)
	rb, err := repo.RescheduleBooking(ctx, db.RescheduleBookingParams{ID: b2.ID, StaffID: staffID, StartAt: newStart, EndAt: newEnd})
	require.NoError(t, err)
	assert.WithinDuration(t, newStart.UTC(), rb.StartAt.UTC(), time.Millisecond)

	// GetStaffByUserID - need a staff linked to user
	// Create user and link staff
	userID := uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO users (id, organization_id, email, password_hash, name, role) VALUES ($1,$2,$3,$4,$5,$6)`, userID, orgID, "staffuser@test.com", "hash", "Staff User", "STAFF")
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `UPDATE staff SET user_id=$1 WHERE id=$2`, userID, staffID)
	require.NoError(t, err)
	stByUser, err := repo.GetStaffByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, staffID, stByUser.ID)

	// ListStaffByService via repo (already)
	// BeginTx
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	require.NotNil(t, tx)
	_ = tx.Rollback(ctx)
}

func TestRepository_NewRepositoryWithQueries(t *testing.T) {
	qs := db.New(testPool)
	repo := NewRepositoryWithQueries(qs, testPool)
	assert.NotNil(t, repo)
	// Call GetService via this repo
	_, _, staffID := setupTestData(t)
	// staff already exists, try GetService with random should error
	_, err := repo.GetService(context.Background(), uuid.New())
	assert.Error(t, err)
	_ = staffID
}
