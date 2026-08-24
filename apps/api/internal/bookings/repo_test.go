package bookings

import (
	"context"
	"os"
	"testing"
	"time"

	"flowbook/api/internal/db"
	"flowbook/api/internal/testhelpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPool    *pgxpool.Pool
	testQueries *db.Queries
	testCtr     *testhelpers.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := testhelpers.CreatePostgresContainer(ctx)
	if err != nil {
		// If Docker not available, skip integration tests gracefully
		// But for T13 we expect it to pass; we exit with error to surface
		panic("failed to create postgres container: " + err.Error())
	}
	testCtr = ctr
	testPool = ctr.Pool
	testQueries = db.New(testPool)
	code := m.Run()
	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

func setupTestData(t *testing.T) (orgID, serviceID, staffID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	// Clean previous
	_, err := testPool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `TRUNCATE staff_services, availability, availability_overrides, staff, services, organizations RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	orgID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO organizations (id, name, slug, timezone) VALUES ($1, $2, $3, $4)`, orgID, "Test Org", "test-org-"+orgID.String()[:8], "Asia/Jakarta")
	require.NoError(t, err)

	serviceID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO services (id, organization_id, name, duration_minutes, buffer_minutes, price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, serviceID, orgID, "Classic Cut", 30, 10, 85000)
	require.NoError(t, err)

	staffID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO staff (id, organization_id, name, email) VALUES ($1,$2,$3,$4)`, staffID, orgID, "Andi", "andi@test.example")
	require.NoError(t, err)

	_, err = testPool.Exec(ctx, `INSERT INTO staff_services (staff_id, service_id) VALUES ($1,$2)`, staffID, serviceID)
	require.NoError(t, err)

	// Minimal availability for completeness (not needed for repo test but for GetSlots if used)
	_, err = testPool.Exec(ctx, `INSERT INTO availability (id, staff_id, day_of_week, start_time, end_time) VALUES ($1,$2,$3,$4::time,$5::time)`, uuid.New(), staffID, 1, "09:00", "18:00")
	require.NoError(t, err)

	return orgID, serviceID, staffID
}

func TestCreateBooking_ExcludeOverlap(t *testing.T) {
	require.NotNil(t, testPool, "test pool not initialized - container failed")
	ctx := context.Background()
	orgID, serviceID, staffID := setupTestData(t)
	// Use queries directly to test EXCLUDE constraint 23P01
	start1 := time.Date(2025, 11, 10, 10, 0, 0, 0, time.UTC)
	end1 := start1.Add(40 * time.Minute) // 30+10

	booking1, err := testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "Alice",
		CustomerEmail: "alice@test.com",
		StartAt: start1,
		EndAt: end1,
		Status: "CONFIRMED",
		PaymentStatus: "PAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", booking1.Status)

	// Overlapping 10:15 same staff -> should violate EXCLUDE 23P01
	start2 := time.Date(2025, 11, 10, 10, 15, 0, 0, time.UTC)
	end2 := start2.Add(40 * time.Minute)
	_, err = testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "Bob",
		CustomerEmail: "bob@test.com",
		StartAt: start2,
		EndAt: end2,
		Status: "PENDING",
		PaymentStatus: "UNPAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.Error(t, err, "overlapping booking should violate EXCLUDE")
	assert.Contains(t, err.Error(), "23P01", "should be SQLSTATE 23P01 exclusion violation")
	// Also contains exclusion or no_overlap
	assert.True(t, containsString(err.Error(), "exclusion") || containsString(err.Error(), "23P01") || containsString(err.Error(), "no_overlap"))

	// Non-overlapping 11:00 should succeed
	start3 := time.Date(2025, 11, 10, 11, 0, 0, 0, time.UTC)
	end3 := start3.Add(40 * time.Minute)
	booking3, err := testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "Carol",
		CustomerEmail: "carol@test.com",
		StartAt: start3,
		EndAt: end3,
		Status: "CONFIRMED",
		PaymentStatus: "PAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", booking3.Status)

	// CANCELLED should NOT block — insert overlapping but CANCELLED should succeed
	start4 := time.Date(2025, 11, 10, 10, 15, 0, 0, time.UTC)
	end4 := start4.Add(40 * time.Minute)
	booking4, err := testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "Dave",
		CustomerEmail: "dave@test.com",
		StartAt: start4,
		EndAt: end4,
		Status: "CANCELLED",
		PaymentStatus: "UNPAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err, "CANCELLED should not trigger EXCLUDE")
	assert.Equal(t, "CANCELLED", booking4.Status)

	// Verify via repo layer that RescheduleTx also respects EXCLUDE
	repo := NewRepository(testPool)
	// Try reschedule booking3 to overlap booking1 => should fail 23P01
	_, err = repo.RescheduleTx(ctx, booking3.ID, staffID, start1, end1)
	require.Error(t, err)
	assert.True(t, containsString(err.Error(), "23P01") || containsString(err.Error(), "exclusion"))
}

func TestTransactionRollback(t *testing.T) {
	require.NotNil(t, testPool)
	ctx := context.Background()
	orgID, serviceID, staffID := setupTestData(t)

	// Count before
	var countBefore int64
	err := testPool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings WHERE organization_id=$1`, orgID).Scan(&countBefore)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countBefore)

	// Begin tx, insert, then rollback
	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	qtx := db.New(tx)
	start := time.Date(2025, 11, 11, 9, 0, 0, 0, time.UTC)
	end := start.Add(40 * time.Minute)
	_, err = qtx.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "RollbackTest",
		CustomerEmail: "rollback@test.com",
		StartAt: start,
		EndAt: end,
		Status: "CONFIRMED",
		PaymentStatus: "PAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	// Rollback
	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// Verify not persisted
	var countAfter int64
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings WHERE organization_id=$1`, orgID).Scan(&countAfter)
	require.NoError(t, err)
	assert.Equal(t, int64(0), countAfter, "rollback should leave 0 bookings")

	// Now commit path: insert via repo and ensure it persists via pool
	repo := NewRepository(testPool)
	b, err := repo.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID,
		ServiceID: serviceID,
		StaffID: staffID,
		CustomerName: "Persist",
		CustomerEmail: "persist@test.com",
		StartAt: start,
		EndAt: end,
		Status: "CONFIRMED",
		PaymentStatus: "PAID",
		CustomerID: pgtype.UUID{Valid: false},
		CustomerPhone: pgtype.Text{Valid: false},
		Notes: pgtype.Text{Valid: false},
		StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, b.ID)
	var countPersist int64
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM bookings WHERE id=$1`, b.ID).Scan(&countPersist)
	require.NoError(t, err)
	assert.Equal(t, int64(1), countPersist)
}

func TestListOverlappingBookings_RealTstzrange(t *testing.T) {
	require.NotNil(t, testPool)
	ctx := context.Background()
	orgID, serviceID, staffID := setupTestData(t)
	// Insert two bookings with gap
	b1Start := time.Date(2025, 11, 12, 9, 0, 0, 0, time.UTC)
	b1End := b1Start.Add(40 * time.Minute)
	_, err := testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID, ServiceID: serviceID, StaffID: staffID,
		CustomerName: "A", CustomerEmail: "a@test.com", StartAt: b1Start, EndAt: b1End, Status: "CONFIRMED", PaymentStatus: "PAID",
		CustomerID: pgtype.UUID{Valid: false}, CustomerPhone: pgtype.Text{Valid: false}, Notes: pgtype.Text{Valid: false}, StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)
	b2Start := time.Date(2025, 11, 12, 11, 0, 0, 0, time.UTC)
	b2End := b2Start.Add(40 * time.Minute)
	_, err = testQueries.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID: orgID, ServiceID: serviceID, StaffID: staffID,
		CustomerName: "B", CustomerEmail: "b@test.com", StartAt: b2Start, EndAt: b2End, Status: "PENDING", PaymentStatus: "UNPAID",
		CustomerID: pgtype.UUID{Valid: false}, CustomerPhone: pgtype.Text{Valid: false}, Notes: pgtype.Text{Valid: false}, StripeSessionID: pgtype.Text{Valid: false},
	})
	require.NoError(t, err)

	// Query overlapping 09:15-09:30 should return only first
	overlaps, err := testQueries.ListOverlappingBookings(ctx, db.ListOverlappingBookingsParams{
		StaffID: staffID,
		Column2: time.Date(2025, 11, 12, 9, 15, 0, 0, time.UTC),
		Column3: time.Date(2025, 11, 12, 9, 30, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, overlaps, 1)
	assert.True(t, overlaps[0].StartAt.Equal(b1Start), "start should equal %v got %v", b1Start, overlaps[0].StartAt)

	// Query 10:00-11:30 should return both? Actually gap 09:40-11:00 free, so 10:30-10:45 not overlapping either
	overlaps2, err := testQueries.ListOverlappingBookings(ctx, db.ListOverlappingBookingsParams{
		StaffID: staffID,
		Column2: time.Date(2025, 11, 12, 10, 30, 0, 0, time.UTC),
		Column3: time.Date(2025, 11, 12, 10, 45, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, overlaps2, 0)

	// Full day query should return 2
	overlaps3, err := testQueries.ListOverlappingBookings(ctx, db.ListOverlappingBookingsParams{
		StaffID: staffID,
		Column2: time.Date(2025, 11, 12, 0, 0, 0, 0, time.UTC),
		Column3: time.Date(2025, 11, 12, 23, 59, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, overlaps3, 2)
}

func containsString(s, sub string) bool {
	if len(sub) == 0 { return true }
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
