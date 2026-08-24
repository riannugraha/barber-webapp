package bookings

import (
	"context"
	"errors"
	"testing"
	"time"

	"flowbook/api/internal/availability"
	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mocks

type mockRepoSvc struct {
	getServiceFn       func(ctx context.Context, id uuid.UUID) (db.Service, error)
	getStaffFn         func(ctx context.Context, id uuid.UUID) (db.Staff, error)
	getStaffByUserFn   func(ctx context.Context, id uuid.UUID) (db.Staff, error)
	getOrgFn           func(ctx context.Context, id uuid.UUID) (db.Organization, error)
	listStaffByServiceFn func(ctx context.Context, id uuid.UUID) ([]db.Staff, error)
	getBookingFn       func(ctx context.Context, id uuid.UUID) (db.Booking, error)
	getBookingByOrgFn  func(ctx context.Context, id, org uuid.UUID) (db.Booking, error)
	listBookingsFn     func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error)
	countFilteredFn    func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error)
	createBookingFn    func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error)
	cancelBookingFn    func(ctx context.Context, id uuid.UUID) (db.Booking, error)
	rescheduleFn       func(ctx context.Context, arg db.RescheduleBookingParams) (db.Booking, error)
	rescheduleTxFn     func(ctx context.Context, id, staffID uuid.UUID, start, end time.Time) (db.Booking, error)
	upsertCustomerFn   func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error)
	beginTxFn          func(ctx context.Context) (interface{}, error)
}

func (m *mockRepoSvc) GetService(ctx context.Context, id uuid.UUID) (db.Service, error) {
	if m.getServiceFn != nil { return m.getServiceFn(ctx, id) }
	return db.Service{}, errors.New("not implemented")
}
func (m *mockRepoSvc) GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	if m.getStaffFn != nil { return m.getStaffFn(ctx, id) }
	return db.Staff{}, errors.New("not implemented")
}
func (m *mockRepoSvc) GetStaffByUserID(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	if m.getStaffByUserFn != nil { return m.getStaffByUserFn(ctx, id) }
	return db.Staff{}, errors.New("not implemented")
}
func (m *mockRepoSvc) GetOrganization(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	if m.getOrgFn != nil { return m.getOrgFn(ctx, id) }
	return db.Organization{}, errors.New("not implemented")
}
func (m *mockRepoSvc) ListStaffByService(ctx context.Context, id uuid.UUID) ([]db.Staff, error) {
	if m.listStaffByServiceFn != nil { return m.listStaffByServiceFn(ctx, id) }
	return nil, nil
}
func (m *mockRepoSvc) GetBooking(ctx context.Context, id uuid.UUID) (db.Booking, error) {
	if m.getBookingFn != nil { return m.getBookingFn(ctx, id) }
	return db.Booking{}, errors.New("not implemented")
}
func (m *mockRepoSvc) GetBookingByIDAndOrg(ctx context.Context, id, org uuid.UUID) (db.Booking, error) {
	if m.getBookingByOrgFn != nil { return m.getBookingByOrgFn(ctx, id, org) }
	return db.Booking{}, errors.New("not implemented")
}
func (m *mockRepoSvc) ListBookings(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error) {
	if m.listBookingsFn != nil { return m.listBookingsFn(ctx, org, from, to, status, staffID, limit, offset) }
	return nil, nil
}
func (m *mockRepoSvc) CountFilteredBookings(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error) {
	if m.countFilteredFn != nil { return m.countFilteredFn(ctx, org, from, to, status, staffID) }
	return 0, nil
}
func (m *mockRepoSvc) CreateBooking(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
	if m.createBookingFn != nil { return m.createBookingFn(ctx, arg) }
	return db.Booking{}, nil
}
func (m *mockRepoSvc) CancelBooking(ctx context.Context, id uuid.UUID) (db.Booking, error) {
	if m.cancelBookingFn != nil { return m.cancelBookingFn(ctx, id) }
	return db.Booking{}, nil
}
func (m *mockRepoSvc) RescheduleBooking(ctx context.Context, arg db.RescheduleBookingParams) (db.Booking, error) {
	if m.rescheduleFn != nil { return m.rescheduleFn(ctx, arg) }
	return db.Booking{}, nil
}
func (m *mockRepoSvc) RescheduleTx(ctx context.Context, id, staffID uuid.UUID, start, end time.Time) (db.Booking, error) {
	if m.rescheduleTxFn != nil { return m.rescheduleTxFn(ctx, id, staffID, start, end) }
	return db.Booking{}, nil
}
func (m *mockRepoSvc) UpsertCustomer(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) {
	if m.upsertCustomerFn != nil { return m.upsertCustomerFn(ctx, arg) }
	return db.Customer{ID: uuid.New()}, nil
}
func (m *mockRepoSvc) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return nil, nil
}

type mockAvail struct {
	slots map[string][]availability.Slot
	err   error
	cleared int
	invalidated int
}
func (a *mockAvail) GetSlots(ctx context.Context, sid, stid, date, tz string) ([]availability.Slot, string, error) {
	if a.err != nil { return nil, "", a.err }
	key := sid+"|"+stid+"|"+date+"|"+tz
	if v, ok := a.slots[key]; ok { return v, tz, nil }
	// default: return one available slot for any request that matches exactly time? We'll generate via stored
	// For tests, we set slots explicitly per date; fallback empty
	return []availability.Slot{}, tz, nil
}
func (a *mockAvail) InvalidateCache() { a.invalidated++ }
func (a *mockAvail) ClearCacheForStaff(id uuid.UUID) { a.cleared++ }

type mockHub struct { broadcasts int; last interface{} }
func (h *mockHub) Broadcast(org uuid.UUID, payload interface{}) { h.broadcasts++; h.last = payload }
type mockEmail struct{}
func (e *mockEmail) SendBookingConfirmed(ctx context.Context, b db.Booking, s db.Service, st db.Staff, org db.Organization) error { return nil }
func (e *mockEmail) SendCancelled(ctx context.Context, b db.Booking, s db.Service, st db.Staff, org db.Organization) error { return nil }

func newTestService(repo *mockRepoSvc, avail *mockAvail) *Service {
	hub := &mockHub{}
	svc := NewService(repo, avail, hub)
	svc.SetEmailSender(&mockEmail{})
	return svc
}

func baseOrgAndService() (uuid.UUID, db.Service, db.Staff) {
	org := uuid.New()
	svc := db.Service{ID: uuid.New(), OrganizationID: org, Name: "Classic Cut", DurationMinutes: 30, BufferMinutes: 10, PriceCents: 85000}
	staff := db.Staff{ID: uuid.New(), OrganizationID: org, Name: "Andi"}
	return org, svc, staff
}

func TestCreate_ValidationErrors(t *testing.T) {
	_, svc, _ := baseOrgAndService()
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
	}
	avail := &mockAvail{slots: map[string][]availability.Slot{}}
	s := newTestService(repo, avail)
	_, err := s.Create(context.Background(), CreateRequest{ServiceID: uuid.Nil, StaffID: uuid.New(), StartAt: time.Now(), CustomerName: "A", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: uuid.Nil, StartAt: time.Now(), CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: uuid.New(), StartAt: time.Time{}, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: uuid.New(), StartAt: time.Now(), CustomerName: "A", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: uuid.New(), StartAt: time.Now(), CustomerName: "An", CustomerEmail: "bad"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
}

func TestCreate_Success_PendingAndFree(t *testing.T) {
	org, svcClassic, staff := baseOrgAndService()
	svcFree := db.Service{ID: uuid.New(), OrganizationID: org, Name: "Free", DurationMinutes: 15, BufferMinutes: 5, PriceCents: 0}
	now := time.Now().UTC().Truncate(time.Minute)
	// Avail returns slot matching now
	availSlots := func(svcID uuid.UUID, staffID uuid.UUID) *mockAvail {
		loc, _ := time.LoadLocation("Asia/Jakarta")
		date := now.In(loc).Format("2006-01-02")
		key := svcID.String()+"|"+staffID.String()+"|"+date+"|Asia/Jakarta"
		return &mockAvail{slots: map[string][]availability.Slot{
			key: {{StartAt: now, EndAt: now.Add(30*time.Minute), Available: true, StaffID: &staffID}},
		}}
	}
	// Pending case
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) {
			if id == svcClassic.ID { return svcClassic, nil }
			return svcFree, nil
		},
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{ID: uuid.New(), OrganizationID: arg.OrganizationID, ServiceID: arg.ServiceID, StaffID: arg.StaffID, StartAt: arg.StartAt, EndAt: arg.EndAt, Status: arg.Status, PaymentStatus: arg.PaymentStatus}, nil
		},
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil },
	}
	s := newTestService(repo, availSlots(svcClassic.ID, staff.ID))
	b, err := s.Create(context.Background(), CreateRequest{ServiceID: svcClassic.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "An@Example.COM", CustomerPhone: strPtr("0812")})
	require.NoError(t, err)
	assert.Equal(t, "PENDING", b.Status)
	assert.Equal(t, "UNPAID", b.PaymentStatus)

	// Free case -> CONFIRMED
	repo2 := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svcFree, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{ID: uuid.New(), OrganizationID: arg.OrganizationID, ServiceID: arg.ServiceID, StaffID: arg.StaffID, StartAt: arg.StartAt, EndAt: arg.EndAt, Status: arg.Status, PaymentStatus: arg.PaymentStatus}, nil
		},
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil },
	}
	s2 := newTestService(repo2, availSlots(svcFree.ID, staff.ID))
	b2, err := s2.Create(context.Background(), CreateRequest{ServiceID: svcFree.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", b2.Status)
	assert.Equal(t, "PAID", b2.PaymentStatus)
}

func TestCreate_SlotUnavailableAndConflict(t *testing.T) {
	org, svc, staff := baseOrgAndService()
	now := time.Now().UTC()
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := now.In(loc).Format("2006-01-02")
	// Slot not found
	availEmpty := &mockAvail{slots: map[string][]availability.Slot{
		svc.ID.String()+"|"+staff.ID.String()+"|"+date+"|Asia/Jakarta": {},
	}}
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
	}
	s := newTestService(repo, availEmpty)
	_, err := s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrSlotUnavailable)

	// Slot found but not available
	availTaken := &mockAvail{slots: map[string][]availability.Slot{
		svc.ID.String()+"|"+staff.ID.String()+"|"+date+"|Asia/Jakarta": {{StartAt: now.UTC(), Available: false}},
	}}
	s2 := newTestService(repo, availTaken)
	_, err = s2.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrSlotUnavailable)

	// Slot available but CreateBooking returns exclusion error 23P01
	availOk := &mockAvail{slots: map[string][]availability.Slot{
		svc.ID.String()+"|"+staff.ID.String()+"|"+date+"|Asia/Jakarta": {{StartAt: now.UTC(), Available: true}},
	}}
	repo3 := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{}, errors.New(`ERROR: conflicting key value violates exclusion constraint "no_overlap" (SQLSTATE 23P01)`)
		},
	}
	s3 := newTestService(repo3, availOk)
	_, err = s3.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrConflict)

	// Generic create error
	repo3.createBookingFn = func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) { return db.Booking{}, errors.New("db down") }
	_, err = s3.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorContains(t, err, "create booking")
}

func TestCreate_OrgMismatchAndStaffOrgMismatch(t *testing.T) {
	org, svc, staff := baseOrgAndService()
	otherOrg := uuid.New()
	staffOther := db.Staff{ID: staff.ID, OrganizationID: otherOrg, Name: "Andi"}
	now := time.Now().UTC()
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := now.In(loc).Format("2006-01-02")
	avail := &mockAvail{slots: map[string][]availability.Slot{
		svc.ID.String()+"|"+staff.ID.String()+"|"+date+"|Asia/Jakarta": {{StartAt: now.UTC(), Available: true}},
	}}
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staffOther, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
	}
	s := newTestService(repo, avail)
	_, err := s.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorContains(t, err, "same organization")
	// org mismatch via request
	repo.getStaffFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil }
	other := otherOrg
	_, err = s.Create(context.Background(), CreateRequest{OrganizationID: &other, ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	assert.ErrorContains(t, err, "organization mismatch")
}

func TestList_RBAC(t *testing.T) {
	org := uuid.New()
	staffID := uuid.New()
	userID := uuid.New()
	repo := &mockRepoSvc{
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) {
			if id == userID { return db.Staff{ID: staffID, OrganizationID: org}, nil }
			return db.Staff{}, errors.New("not found")
		},
		countFilteredFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error) { return 1, nil },
		listBookingsFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error) {
			return []db.Booking{{ID: uuid.New()}}, nil
		},
	}
	s := newTestService(repo, &mockAvail{})
	// CUSTOMER forbidden
	_, err := s.List(context.Background(), ListFilter{OrgID: org, Role: "CUSTOMER"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)
	// STAFF must provide own, forbidden to query other staff
	otherStaff := uuid.New()
	_, err = s.List(context.Background(), ListFilter{OrgID: org, Role: "STAFF", UserID: userID, StaffID: &otherStaff})
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)
	// STAFF success with own
	res, err := s.List(context.Background(), ListFilter{OrgID: org, Role: "STAFF", UserID: userID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Data))
	// OWNER full
	res, err = s.List(context.Background(), ListFilter{OrgID: org, Role: "OWNER", StaffID: &staffID})
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.Data))
	// validation org required
	_, err = s.List(context.Background(), ListFilter{OrgID: uuid.Nil, Role: "OWNER"})
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	// pagination defaults
	res, err = s.List(context.Background(), ListFilter{OrgID: org, Role: "OWNER", Page: 0, Limit: 200})
	require.NoError(t, err)
	assert.Equal(t, 1, res.Page)
	assert.Equal(t, 100, res.Limit) // capped
}

func TestGetByID_RBAC(t *testing.T) {
	org := uuid.New()
	bookingID := uuid.New()
	staffID := uuid.New()
	userStaff := uuid.New()
	userID := uuid.New()
	booking := db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, CustomerEmail: "a@b.com"}
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			if id == bookingID { return booking, nil }
			return db.Booking{}, errors.New("not found")
		},
		getBookingByOrgFn: func(ctx context.Context, id, o uuid.UUID) (db.Booking, error) {
			if id == bookingID && o == org { return booking, nil }
			return db.Booking{}, errors.New("not found")
		},
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) {
			if id == userID { return db.Staff{ID: userStaff}, nil }
			return db.Staff{}, errors.New("not found")
		},
	}
	s := newTestService(repo, &mockAvail{})
	// invalid id
	_, err := s.GetByID(context.Background(), uuid.Nil, &org, "OWNER", uuid.Nil)
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	// not found
	_, err = s.GetByID(context.Background(), uuid.New(), &org, "OWNER", uuid.Nil)
	assert.ErrorIs(t, errors.Unwrap(err), ErrNotFound)
	// STAFF forbidden when not own
	_, err = s.GetByID(context.Background(), bookingID, &org, "STAFF", userID)
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)
	// STAFF own
	repo.getStaffByUserFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffID}, nil }
	_, err = s.GetByID(context.Background(), bookingID, &org, "STAFF", uuid.New())
	require.NoError(t, err)
	// CUSTOMER email mismatch
	_, err = s.GetByIDWithEmail(context.Background(), bookingID, &org, "CUSTOMER", uuid.New(), "other@b.com")
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)
	// CUSTOMER own
	_, err = s.GetByIDWithEmail(context.Background(), bookingID, &org, "CUSTOMER", uuid.New(), "a@b.com")
	require.NoError(t, err)
	// CUSTOMER without email (public track)
	_, err = s.GetByIDWithEmail(context.Background(), bookingID, &org, "CUSTOMER", uuid.New(), "")
	require.NoError(t, err)
}

func TestCancel(t *testing.T) {
	org := uuid.New()
	bookingID := uuid.New()
	staffID := uuid.New()
	// existing CANCELLED idempotent
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, Status: "CANCELLED"}, nil
		},
	}
	s := newTestService(repo, &mockAvail{})
	b, err := s.Cancel(context.Background(), bookingID, "OWNER", uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", b.Status)

	// CUSTOMER forbidden
	repo.getBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
		return db.Booking{ID: bookingID, StaffID: staffID, Status: "CONFIRMED"}, nil
	}
	_, err = s.Cancel(context.Background(), bookingID, "CUSTOMER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)

	// STAFF not own
	repo.getStaffByUserFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: uuid.New()}, nil }
	_, err = s.Cancel(context.Background(), bookingID, "STAFF", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)

	// STAFF own and success
	repo.getStaffByUserFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffID}, nil }
	repo.cancelBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
		return db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, Status: "CANCELLED"}, nil
	}
	b, err = s.Cancel(context.Background(), bookingID, "STAFF", uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "CANCELLED", b.Status)
	repo.cancelBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) { return db.Booking{}, errors.New("db err") }
	_, err = s.Cancel(context.Background(), bookingID, "STAFF", uuid.New())
	assert.ErrorContains(t, err, "cancel booking")

	// also test Cancel with avail cache cleared
	repo.cancelBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
		return db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, Status: "CANCELLED"}, nil
	}
	// Ensure not found case
	repo.getBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) { return db.Booking{}, errors.New("not found") }
	_, err = s.Cancel(context.Background(), uuid.New(), "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrNotFound)
}

func TestReschedule(t *testing.T) {
	org := uuid.New()
	svcID := uuid.New()
	staffOld := uuid.New()
	staffNew := uuid.New()
	bookingID := uuid.New()
	now := time.Now().UTC()
	later := now.Add(2 * time.Hour)
	svc := db.Service{ID: svcID, OrganizationID: org, DurationMinutes: 30, BufferMinutes: 10}
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, ServiceID: svcID, StaffID: staffOld, StartAt: now, EndAt: now.Add(40 * time.Minute), Status: "CONFIRMED"}, nil
		},
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffOld}, nil },
		rescheduleTxFn: func(ctx context.Context, id, sid uuid.UUID, start, end time.Time) (db.Booking, error) {
			return db.Booking{ID: bookingID, StaffID: sid, StartAt: start, EndAt: end}, nil
		},
	}
	// avail returns available for later slot
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := later.In(loc).Format("2006-01-02")
	avail := &mockAvail{slots: map[string][]availability.Slot{
		svcID.String()+"|"+staffNew.String()+"|"+date+"|Asia/Jakarta": {{StartAt: later.UTC(), Available: true}},
		svcID.String()+"|"+staffOld.String()+"|"+date+"|Asia/Jakarta": {{StartAt: later.UTC(), Available: true}},
	}}
	s := newTestService(repo, avail)

	// validation invalid id
	_, err := s.Reschedule(context.Background(), uuid.Nil, RescheduleRequest{StaffID: staffNew, StartAt: later}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: uuid.Nil, StartAt: later}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	_, err = s.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffNew, StartAt: time.Time{}}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)

	// STAFF cannot reschedule to other staff
	_, err = s.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffNew, StartAt: later}, "STAFF", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)

	// CUSTOMER forbidden
	_, err = s.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: later}, "CUSTOMER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrForbidden)

	// cancelled booking
	repo.getBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
		return db.Booking{ID: bookingID, Status: "CANCELLED"}, nil
	}
	_, err = s.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: later}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrValidation)
	repo.getBookingFn = func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
		return db.Booking{ID: bookingID, OrganizationID: org, ServiceID: svcID, StaffID: staffOld, StartAt: now, EndAt: now.Add(40 * time.Minute), Status: "CONFIRMED"}, nil
	}

	// slot not on grid
	availEmpty := &mockAvail{slots: map[string][]availability.Slot{
		svcID.String()+"|"+staffOld.String()+"|"+date+"|Asia/Jakarta": {},
	}}
	s2 := newTestService(repo, availEmpty)
	_, err = s2.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: later}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrSlotUnavailable)

	// slot taken but overlaps old (allow)
	availTaken := &mockAvail{slots: map[string][]availability.Slot{
		svcID.String()+"|"+staffOld.String()+"|"+date+"|Asia/Jakarta": {{StartAt: now.UTC(), Available: false}},
	}}
	// request to move to same old slot (should be allowed via overlapsOld)
	// Use now as later (same as existing)
	availOverlap := &mockAvail{slots: map[string][]availability.Slot{
		svcID.String()+"|"+staffOld.String()+"|"+now.In(loc).Format("2006-01-02")+"|Asia/Jakarta": {{StartAt: now.UTC(), Available: false}},
	}}
	s3 := newTestService(repo, availOverlap)
	_, err = s3.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: now}, "OWNER", uuid.New())
	// Should succeed because overlapsOld true, then tries RescheduleTx which we mock success
	require.NoError(t, err)

	_ = availTaken
	_ = later

	// success reschedule to new staff (OWNER)
	repo.getStaffByUserFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffOld}, nil }
	availOk := &mockAvail{slots: map[string][]availability.Slot{
		svcID.String() + "|" + staffOld.String() + "|" + date + "|Asia/Jakarta": {{StartAt: later.UTC(), Available: true}},
	}}
	s4 := newTestService(repo, availOk)
	_, err = s4.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: later}, "OWNER", uuid.New())
	require.NoError(t, err)

	// RescheduleTx conflict 23P01
	repo.rescheduleTxFn = func(ctx context.Context, id, sid uuid.UUID, start, end time.Time) (db.Booking, error) {
		return db.Booking{}, errors.New(`23P01 exclusion`)
	}
	_, err = s4.Reschedule(context.Background(), bookingID, RescheduleRequest{StaffID: staffOld, StartAt: later}, "OWNER", uuid.New())
	assert.ErrorIs(t, errors.Unwrap(err), ErrConflict)
}

func TestHelpersBookings(t *testing.T) {
	assert.False(t, isExclusionError(nil))
	assert.True(t, isExclusionError(errors.New("23P01")))
	assert.True(t, isExclusionError(errors.New("exclusion")))
	assert.False(t, isExclusionError(errors.New("other")))
	assert.NotNil(t, toPgText(nil))
	s := "hi"
	assert.True(t, toPgText(&s).Valid)
	empty := " "
	assert.False(t, toPgText(&empty).Valid)
	assert.True(t, safeBroadcast(nil, uuid.New(), nil) == nil)
	hub := &mockHub{}
	assert.NoError(t, safeBroadcast(hub, uuid.New(), "test"))
	// avail nil case
	org, svc, staff := baseOrgAndService()
	now := time.Now().UTC()
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{ID: uuid.New(), Status: arg.Status, PaymentStatus: arg.PaymentStatus}, nil
		},
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) {
			return db.Customer{}, errors.New("upsert fail")
		},
	}
	sNilAvail := NewService(repo, nil, nil)
	b, err := sNilAvail.Create(context.Background(), CreateRequest{ServiceID: svc.ID, StaffID: staff.ID, StartAt: now, CustomerName: "An", CustomerEmail: "a@b.com"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, b.ID)
}

func strPtr(s string) *string { return &s }
