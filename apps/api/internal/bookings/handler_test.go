package bookings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flowbook/api/internal/db"
	appmw "flowbook/api/internal/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHandlerWithMock(repo *mockRepoSvc, avail *mockAvail) *Handler {
	svc := NewService(repo, avail, &mockHub{})
	h := NewHandler(svc)
	return h
}

func TestHandler_Create_201(t *testing.T) {
	org, svc, staff := baseOrgAndService()
	now := time.Now().UTC().Truncate(time.Second)
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{ID: uuid.New(), OrganizationID: org, ServiceID: svc.ID, StaffID: staff.ID, CustomerName: arg.CustomerName, CustomerEmail: arg.CustomerEmail, StartAt: arg.StartAt, EndAt: arg.EndAt, Status: arg.Status, PaymentStatus: arg.PaymentStatus, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil },
	}
	svcNilAvail := NewService(repo, nil, &mockHub{})
	h := NewHandler(svcNilAvail)
	e := echo.New()
	body := map[string]interface{}{
		"serviceId": svc.ID.String(),
		"staffId": staff.ID.String(),
		"startAt": now.Format(time.RFC3339),
		"customerName": "Anastasia",
		"customerEmail": "ana@test.com",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp bookingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, svc.ID.String(), resp.ServiceID)
}

func TestHandler_Create_409_Conflict(t *testing.T) {
	org, svc, staff := baseOrgAndService()
	now := time.Now().UTC()
	repo := &mockRepoSvc{
		getServiceFn: func(ctx context.Context, id uuid.UUID) (db.Service, error) { return svc, nil },
		getStaffFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return staff, nil },
		getOrgFn: func(ctx context.Context, id uuid.UUID) (db.Organization, error) { return db.Organization{ID: org, Timezone: "Asia/Jakarta"}, nil },
		upsertCustomerFn: func(ctx context.Context, arg db.UpsertCustomerParams) (db.Customer, error) { return db.Customer{ID: uuid.New()}, nil },
		createBookingFn: func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
			return db.Booking{}, assert.AnError
		},
	}
	// Make Create return ErrConflict via isExclusionError: createBooking returns 23P01
	repo.createBookingFn = func(ctx context.Context, arg db.CreateBookingParams) (db.Booking, error) {
		return db.Booking{}, errors.New(`conflicting key value violates exclusion constraint "no_overlap" (SQLSTATE 23P01)`)
	}
	svcNoAvail := NewService(repo, nil, &mockHub{})
	h := NewHandler(svcNoAvail)
	e := echo.New()
	body := map[string]interface{}{
		"serviceId": svc.ID.String(),
		"staffId": staff.ID.String(),
		"startAt": now.Format(time.RFC3339),
		"customerName": "An",
		"customerEmail": "a@b.com",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "Slot already taken")
}

func TestHandler_Create_422_Validation(t *testing.T) {
	_, svc, staff := baseOrgAndService()
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	// missing fields
	body := map[string]interface{}{
		"serviceId": svc.ID.String(),
		"staffId": staff.ID.String(),
		"startAt": time.Now().Format(time.RFC3339),
		"customerName": "A", // too short
		"customerEmail": "bad",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewReader(b))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.Create(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHandler_List_403_Forbidden(t *testing.T) {
	org := uuid.New()
	repo := &mockRepoSvc{
		getStaffByUserFn: func(ctx context.Context, id uuid.UUID) (db.Staff, error) { return db.Staff{ID: uuid.New(), OrganizationID: org}, nil },
		countFilteredFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID) (int64, error) { return 0, nil },
		listBookingsFn: func(ctx context.Context, org uuid.UUID, from, to *time.Time, status *string, staffID *uuid.UUID, limit, offset int32) ([]db.Booking, error) { return nil, nil },
	}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bookings?organizationId="+org.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Set CUSTOMER claims
	claims := &appmw.Claims{UserID: uuid.New().String(), Role: "CUSTOMER", OrgID: handlerStrPtr(org.String())}
	c.Set(appmw.ContextKeyUser, claims)
	err := h.List(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_List_401_Unauthorized(t *testing.T) {
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bookings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.List(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandler_GetByID_200(t *testing.T) {
	org := uuid.New()
	bookingID := uuid.New()
	repo := &mockRepoSvc{
		getBookingFn: func(ctx context.Context, id uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, ServiceID: uuid.New(), StaffID: uuid.New(), CustomerName: "An", CustomerEmail: "a@b.com", StartAt: time.Now(), EndAt: time.Now().Add(30*time.Minute), Status: "CONFIRMED", PaymentStatus: "PAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		getBookingByOrgFn: func(ctx context.Context, id, o uuid.UUID) (db.Booking, error) {
			return db.Booking{ID: bookingID, OrganizationID: org, ServiceID: uuid.New(), StaffID: uuid.New(), CustomerName: "An", CustomerEmail: "a@b.com", StartAt: time.Now(), EndAt: time.Now().Add(30*time.Minute), Status: "CONFIRMED", PaymentStatus: "PAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bookings/"+bookingID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	_ = h.GetByID(c)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_Cancel_And_Reschedule_RequireAuth(t *testing.T) {
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	// Cancel without auth -> 401
	bookingID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bookingID.String())
	_ = h.Cancel(c)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	// Reschedule without auth -> 401
	req2 := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/reschedule", bytes.NewReader([]byte(`{}`)))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetParamNames("id")
	c2.SetParamValues(bookingID.String())
	_ = h.Reschedule(c2)
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)
}

func TestHandler_RegisterRoutes(t *testing.T) {
	repo := &mockRepoSvc{}
	h := newHandlerWithMock(repo, &mockAvail{})
	e := echo.New()
	g := e.Group("/api/v1")
	h.RegisterRoutes(g, "")
	// check routes
	found := false
	for _, r := range e.Routes() {
		if r.Method == http.MethodPost && r.Path == "/api/v1/bookings" {
			found = true
		}
	}
	assert.True(t, found)
	// with jwt secret
	e2 := echo.New()
	g2 := e2.Group("/api/v1")
	h.RegisterRoutes(g2, "secret")
	found2 := false
	for _, r := range e2.Routes() {
		if r.Method == http.MethodGet && r.Path == "/api/v1/bookings" {
			found2 = true
		}
	}
	assert.True(t, found2)
}

func TestHandler_ValidationHelpers(t *testing.T) {
	h := &Handler{}
	// jsonFieldName
	assert.Equal(t, "serviceId", jsonFieldName("ServiceID"))
	assert.Equal(t, "custom", jsonFieldName("Custom"))
	assert.Equal(t, "", jsonFieldName(""))
	// msgForTag
	// need validator.FieldError mock? skip
	// toBookingResponse with pgtype
	b := db.Booking{ID: uuid.New(), OrganizationID: uuid.New(), ServiceID: uuid.New(), StaffID: uuid.New(), CustomerID: pgtype.UUID{Valid: false}, CustomerPhone: pgtype.Text{Valid: false}, Notes: pgtype.Text{Valid: false}, StripeSessionID: pgtype.Text{Valid: false}, StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), Status: "PENDING", PaymentStatus: "UNPAID", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	resp := toBookingResponse(b)
	assert.Equal(t, b.ID.String(), resp.ID)
	// with valid customerID
	custID := uuid.New()
	b.CustomerID = pgtype.UUID{Bytes: custID, Valid: true}
	resp = toBookingResponse(b)
	assert.Equal(t, custID.String(), *resp.CustomerID)
	// parseDateParam
	_, err := parseDateParam("2025-11-10")
	assert.NoError(t, err)
	_, err = parseDateParam(time.Now().Format(time.RFC3339))
	assert.NoError(t, err)
	_, err = parseDateParam("bad")
	assert.Error(t, err)
	_ = h
}

func handlerStrPtr(s string) *string { return &s }
