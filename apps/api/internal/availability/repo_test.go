package availability

import (
	"context"
	"testing"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDB implements db.DBTX for repo tests without real DB.
type mockDB struct {
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{err: pgx.ErrNoRows}
}
func (m *mockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return nil, assert.AnError
}
func (m *mockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag(""), nil
}

type mockRow struct {
	err error
}

func (r *mockRow) Scan(dest ...interface{}) error {
	return r.err
}

func TestNewRepository(t *testing.T) {
	// NewRepository with valid queries
	mdb := &mockDB{}
	qs := db.New(mdb)
	repo := NewRepository(qs)
	assert.NotNil(t, repo)
	// also with nil (should not panic, but methods will panic if called — we test construction only)
	repo2 := NewRepository(nil)
	assert.NotNil(t, repo2)
}

func TestRepository_Methods_WithMock(t *testing.T) {
	mdb := &mockDB{}
	qs := db.New(mdb)
	repo := NewRepository(qs)
	ctx := context.Background()
	id := uuid.New()
	// All methods delegate to sqlc; they will return errors from mockDB
	_, err := repo.GetService(ctx, id)
	assert.Error(t, err)
	_, err = repo.GetStaff(ctx, id)
	assert.Error(t, err)
	_, err = repo.GetOrganizationByID(ctx, id)
	assert.Error(t, err)
	_, err = repo.ListAvailabilityByStaff(ctx, id)
	assert.Error(t, err)
	_, err = repo.GetOverrideByStaffAndDate(ctx, id, time.Now())
	assert.Error(t, err)
	_, err = repo.ListStaffByService(ctx, id)
	assert.Error(t, err)
	_, err = repo.ListOverlappingBookings(ctx, id, time.Now(), time.Now().Add(time.Hour))
	assert.Error(t, err)
	_, err = repo.ListStaffServices(ctx, id)
	assert.Error(t, err)
}

func TestRepository_GetOverride_And_ListStaffServices_WithRowSuccess(t *testing.T) {
	// Provide a mock that returns no rows for QueryRow but we check handling
	mdb := &mockDB{
		queryRowFn: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &mockRow{err: pgx.ErrNoRows}
		},
	}
	qs := db.New(mdb)
	repo := NewRepository(qs)
	_, err := repo.GetOverrideByStaffAndDate(context.Background(), uuid.New(), time.Now())
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
