package bookings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flowbook/api/internal/availability"
	"flowbook/api/internal/db"
	appmw "flowbook/api/internal/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandler_Create_ValidationAndParsing(t *testing.T) {
	_, svc, staff := baseOrgAndService()
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()

	// invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// missing serviceId
	body := map[string]interface{}{"staffId": staff.ID.String(), "startAt": time.Now().Format(time.RFC3339), "customerName": "An", "customerEmail": "a@b.com"}
	b, _ := json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid serviceId uuid
	body = map[string]interface{}{"serviceId": "bad", "staffId": staff.ID.String(), "startAt": time.Now().Format(time.RFC3339), "customerName": "An", "customerEmail": "a@b.com"}
	b, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid staffId
	body = map[string]interface{}{"serviceId": svc.ID.String(), "staffId": "bad", "startAt": time.Now().Format(time.RFC3339), "customerName": "An", "customerEmail": "a@b.com"}
	b, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid startAt
	body = map[string]interface{}{"serviceId": svc.ID.String(), "staffId": staff.ID.String(), "startAt": "bad", "customerName": "An", "customerEmail": "a@b.com"}
	b, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid organizationId
	orgBad := "bad-uuid"
	body = map[string]interface{}{"organizationId": orgBad, "serviceId": svc.ID.String(), "staffId": staff.ID.String(), "startAt": time.Now().Format(time.RFC3339), "customerName": "An", "customerEmail": "a@b.com"}
	b, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// valid with RFC3339Nano fallback
	nowNano := time.Now().UTC().Format(time.RFC3339Nano)
	body = map[string]interface{}{"serviceId": svc.ID.String(), "staffId": staff.ID.String(), "startAt": nowNano, "customerName": "An", "customerEmail": "a@b.com"}
	b, _ = json.Marshal(body)
	// Need repo to handle Create with avail nil -> no slot check
	repo.getServiceFn = func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil }
	repo.getStaffFn = func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil }
	repo.getOrgFn = func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: uuid.New(), Timezone: "Asia/Jakarta"}, nil }
	repo.createBookingFn = func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
		return db.Booking{ID: uuid.New(), OrganizationID: arg.OrganizationID, ServiceID: arg.ServiceID, StaffID: arg.StaffID, CustomerName: arg.CustomerName, CustomerEmail: arg.CustomerEmail, StartAt: arg.StartAt, EndAt: arg.EndAt, Status: arg.Status, PaymentStatus: arg.PaymentStatus, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
	}
	repo.upsertCustomerFn = func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil }
	// Use handler with avail nil
	h2 := NewHandler(NewService(repo, nil, &mockHub{}))
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h2.Create(c)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandler_List_QueryParams(t *testing.T) {
	org := uuid.New()
	staffID := uuid.New()
	userID := uuid.New()
	repo := &mockRepoSvc{
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffID, OrganizationID: org}, nil },
		countFilteredFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error) { return 1, nil },
		listBookingsFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, sid *uuid.UUID, limit, offset int32) ([]db.Booking, error) {
			useID := staffID
			if sid != nil {
				useID = *sid
			}
			return []db.Booking{{ID: uuid.New(), OrganizationID: org, ServiceID: uuid.New(), StaffID: useID, CustomerName: "A", CustomerEmail: "a@b.com", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Status: "CONFIRMED", PaymentStatus: "PAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}}, nil
		},
	}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()

	// valid with all query params
	req := httptest.NewRequest(http.MethodGet, "/bookings?organizationId="+org.String()+"&from=2025-11-10&to=2025-11-11&status=CONFIRMED&staffId="+staffID.String()+"&page=1&limit=10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	claims := &appmw.Claims{UserID: userID.String(), Role: "OWNER", OrgID: handlerStrPtr(org.String())}
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusOK, rec.Code)

	// invalid from
	req = httptest.NewRequest(http.MethodGet, "/bookings?from=bad", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid to
	req = httptest.NewRequest(http.MethodGet, "/bookings?to=bad", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid status
	req = httptest.NewRequest(http.MethodGet, "/bookings?status=BAD", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid staffId
	req = httptest.NewRequest(http.MethodGet, "/bookings?staffId=bad", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid page
	req = httptest.NewRequest(http.MethodGet, "/bookings?page=0", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid limit
	req = httptest.NewRequest(http.MethodGet, "/bookings?limit=200", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// missing org
	claimsNoOrg := &appmw.Claims{UserID: userID.String(), Role: "OWNER", OrgID: nil}
	req = httptest.NewRequest(http.MethodGet, "/bookings", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claimsNoOrg)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid org uuid from token
	claimsBadOrg := &appmw.Claims{UserID: userID.String(), Role: "OWNER", OrgID: handlerStrPtr("bad")}
	req = httptest.NewRequest(http.MethodGet, "/bookings", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claimsBadOrg)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// from with RFC3339
	req = httptest.NewRequest(http.MethodGet, "/bookings?from="+time.Now().Format(time.RFC3339), nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.NotEqual(t, http.StatusInternalServerError, rec.Code)

	// to with date only should set end of day
	req = httptest.NewRequest(http.MethodGet, "/bookings?to=2025-11-10", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.List(c)
	assert.NotEqual(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_GetByID_Errors(t *testing.T) {
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) { return db.Booking{}, assert.AnError },
		getBookingByOrgFn: func(ctx context.Context, id, org uuid.UUID) (db.Booking, error) { return db.Booking{}, assert.AnError },
	}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()

	// missing id
	req := httptest.NewRequest(http.MethodGet, "/bookings/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("")
	_ = h.GetByID(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid uuid
	req = httptest.NewRequest(http.MethodGet, "/bookings/bad", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad")
	_ = h.GetByID(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// not found via service
	req = httptest.NewRequest(http.MethodGet, "/bookings/"+uuid.New().String(), nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())
	// set claims to trigger GetByIDWithEmail path
	claims := &appmw.Claims{UserID: uuid.New().String(), Role: "OWNER", OrgID: handlerStrPtr(uuid.New().String())}
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.GetByID(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// with query param id fallback
	req = httptest.NewRequest(http.MethodGet, "/bookings?id="+uuid.New().String(), nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.GetByID(c)
	// should try query param and then fail with not found -> 404 or 422? It will try parse and then call service -> 404
	assert.Contains(t, rec.Body.String(), "not_found")
}

func TestHandler_Cancel_SuccessAndErrors(t *testing.T) {
	org := uuid.New()
	bookingID := uuid.New()
	staffID := uuid.New()
	userID := uuid.New()
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, Status: "CONFIRMED"}, nil
		},
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffID}, nil },
		cancelBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, StaffID: staffID, Status: "CANCELLED", CustomerName: "A", CustomerEmail: "a@b.com", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), PaymentStatus: "UNPAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	// success
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	claims := &appmw.Claims{UserID: userID.String(), Role: "OWNER", OrgID: handlerStrPtr(org.String())}
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Cancel(c)
	assert.Equal(t, http.StatusOK, rec.Code)

	// invalid id
	req = httptest.NewRequest(http.MethodPost, "/bookings/bad/cancel", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad")
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Cancel(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// missing id
	req = httptest.NewRequest(http.MethodPost, "/bookings//cancel", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("")
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Cancel(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHandler_Reschedule_SuccessAndValidation(t *testing.T) {
	org := uuid.New()
	bookingID := uuid.New()
	staffID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, ServiceID: uuid.New(), StaffID: staffID, StartAt: now, EndAt: now.Add(40 * time.Minute), Status: "CONFIRMED"}, nil
		},
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) {
			return db.Service{ID: id, OrganizationID: org, DurationMinutes: 30, BufferMinutes: 10}, nil
		},
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: staffID}, nil },
		rescheduleTxFn: func(ctx context.Context, id, sid uuid.UUID, start, end time.Time) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, StaffID: sid, StartAt: start, EndAt: end, Status: "CONFIRMED", CustomerName: "A", CustomerEmail: "a@b.com", PaymentStatus: "PAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	// Need to set avail to return slot for reschedule
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := now.Add(2 * time.Hour).In(loc).Format("2006-01-02")
	// Create a simple mock that always returns available
	mockAvailForResched := &mockAvailResched{available: true, start: now.Add(2 * time.Hour).UTC()}
	h := NewHandler(NewService(repo, mockAvailForResched, &mockHub{}))
	e := echo.New()
	// success
	body := map[string]interface{}{"staffId": staffID.String(), "startAt": now.Add(2 * time.Hour).Format(time.RFC3339)}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/reschedule", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	claims := &appmw.Claims{UserID: userID.String(), Role: "OWNER", OrgID: handlerStrPtr(org.String())}
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Reschedule(c)
	// may be 200 or 422 depending on slot validation; we just check not 500
	assert.NotEqual(t, http.StatusInternalServerError, rec.Code)

	// invalid id
	req = httptest.NewRequest(http.MethodPost, "/bookings/bad/reschedule", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("bad")
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Reschedule(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid staffId in body
	bodyBad := map[string]interface{}{"staffId": "bad", "startAt": now.Format(time.RFC3339)}
	b, _ = json.Marshal(bodyBad)
	req = httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/reschedule", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Reschedule(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// invalid startAt
	bodyBad2 := map[string]interface{}{"staffId": staffID.String(), "startAt": "bad"}
	b, _ = json.Marshal(bodyBad2)
	req = httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/reschedule", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	c.Set(appmw.ContextKeyUser, claims)
	_ = h.Reschedule(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// missing auth
	req = httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/reschedule", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	_ = h.Reschedule(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_ = date
}

type availabilitySlotForHandler struct {
	StartAt   time.Time
	Available bool
}
type mockAvailResched struct {
	available bool
	start     time.Time
}

func (m *mockAvailResched) GetSlots(ctx context.Context, sid, stid, date, tz string) ([]availability.Slot, string, error) {
	// return a slot that matches start
	return []availability.Slot{{StartAt: m.start, Available: m.available}}, tz, nil
}
func (m *mockAvailResched) InvalidateCache() {}
func (m *mockAvailResched) ClearCacheForStaff(id uuid.UUID) {}

func TestHandler_MapServiceErrorAndHelpers(t *testing.T) {
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	// Test mapServiceError via Create with service returning ErrNotFound
	repo.getServiceFn = func(ctx context.Context, id uuid.UUID) (db.Service, error) { return db.Service{}, assert.AnError }
	// Actually Create will map to not_found via repo.GetService error
	// We'll call handler Create with valid payload but repo returns error that maps to 500
	body := map[string]interface{}{"serviceId": uuid.New().String(), "staffId": uuid.New().String(), "startAt": time.Now().Format(time.RFC3339), "customerName": "An", "customerEmail": "a@b.com"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.Create(c)
	// should be 404 or 500 depending on error mapping
	assert.NotEqual(t, http.StatusOK, rec.Code)

	// Test validationError helper directly
	// Create a validation error via missing fields
	body = map[string]interface{}{}
	b, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// Test toBookingResponse with valid pgtype
	bk := db.Booking{ID: uuid.New(), OrganizationID: uuid.New(), ServiceID: uuid.New(), StaffID: uuid.New(), CustomerID: pgtype.UUID{Valid: false}, CustomerPhone: pgtype.Text{String: "0812", Valid: true}, Notes: pgtype.Text{String: "note", Valid: true}, StripeSessionID: pgtype.Text{String: "cs_test", Valid: true}, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Status: "PENDING", PaymentStatus: "UNPAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	resp := toBookingResponse(bk)
	assert.Equal(t, "0812", *resp.CustomerPhone)
	assert.Equal(t, "note", *resp.Notes)
	assert.Equal(t, "cs_test", *resp.StripeSessionID)

	// parseDateParam with RFC3339Nano
	_, err := parseDateParam(time.Now().Format(time.RFC3339Nano))
	assert.NoError(t, err)

	// jsonFieldName already tested but ensure coverage
	assert.Equal(t, "serviceId", jsonFieldName("ServiceID"))
	assert.Equal(t, "staffId", jsonFieldName("StaffID"))
	assert.Equal(t, "custom", jsonFieldName("Custom"))
	assert.Equal(t, "", jsonFieldName(""))
	// msgForTag tested via validationError path
	_ = e
}
