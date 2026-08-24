package bookings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"flowbook/api/internal/availability"
	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Sentinel errors for handler mapping.
var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrSlotUnavailable = errors.New("slot unavailable")
	ErrForbidden       = errors.New("forbidden")
	ErrValidation      = errors.New("validation error")
)

// Hub broadcasts realtime events — gorilla/websocket native (Koyeb), no Pusher.
// Implemented by internal/ws.Hub; for bookings we keep interface small.
type Hub interface {
	Broadcast(orgID uuid.UUID, payload interface{})
}

// AvailabilityValidator is the calendar engine interface for validation via GetSlots.
// Using concrete *availability.Service keeps it simple; interface for testability.
type AvailabilityValidator interface {
	GetSlots(ctx context.Context, serviceIDStr, staffIDStr, dateStr, tzStr string) ([]availability.Slot, string, error)
	InvalidateCache()
	ClearCacheForStaff(staffID uuid.UUID)
}

// Service is bookings domain logic — owns validation via availability, EXCLUDE handling, RBAC scoping.
type Service struct {
	repo  Repository
	avail AvailabilityValidator
	hub   Hub
}

// noopHub is fallback when WS hub not wired (tests).
type noopHub struct{}

func (n *noopHub) Broadcast(_ uuid.UUID, _ interface{}) {}

// NewService creates Service. hub may be nil -> noop, avail may be nil -> skip slot validation (not recommended for prod).
func NewService(repo Repository, avail AvailabilityValidator, hub Hub) *Service {
	if hub == nil {
		hub = &noopHub{}
	}
	return &Service{repo: repo, avail: avail, hub: hub}
}

// CreateRequest mirrors openapi CreateBookingRequest.
type CreateRequest struct {
	OrganizationID *uuid.UUID `json:"organizationId"`
	ServiceID      uuid.UUID  `json:"serviceId"`
	StaffID        uuid.UUID  `json:"staffId"`
	StartAt        time.Time  `json:"startAt"`
	CustomerName   string     `json:"customerName"`
	CustomerEmail  string     `json:"customerEmail"`
	CustomerPhone  *string    `json:"customerPhone"`
	Notes          *string    `json:"notes"`
}

// ListFilter holds GET /bookings filter params.
type ListFilter struct {
	OrgID   uuid.UUID
	From    *time.Time
	To      *time.Time
	Status  *string
	StaffID *uuid.UUID
	Page    int
	Limit   int
	// RBAC context
	Role   string
	UserID uuid.UUID
}

// PaginatedResult holds bookings + pagination meta.
type PaginatedResult struct {
	Data       []db.Booking `json:"data"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	Total      int64        `json:"total"`
	TotalPages int          `json:"totalPages"`
}

// RescheduleRequest for POST /bookings/:id/reschedule
type RescheduleRequest struct {
	StaffID uuid.UUID `json:"staffId"`
	StartAt time.Time `json:"startAt"`
}

// Create validates via availability.Service.GetSlots, inserts tstzrange, maps 23P01 -> 409.
func (s *Service) Create(ctx context.Context, req CreateRequest) (db.Booking, error) {
	if req.ServiceID == uuid.Nil {
		return db.Booking{}, fmt.Errorf("%w: serviceId required", ErrValidation)
	}
	if req.StaffID == uuid.Nil {
		return db.Booking{}, fmt.Errorf("%w: staffId required", ErrValidation)
	}
	if req.StartAt.IsZero() {
		return db.Booking{}, fmt.Errorf("%w: startAt required", ErrValidation)
	}
	if strings.TrimSpace(req.CustomerName) == "" || len(strings.TrimSpace(req.CustomerName)) < 2 {
		return db.Booking{}, fmt.Errorf("%w: customerName min 2", ErrValidation)
	}
	if strings.TrimSpace(req.CustomerEmail) == "" || !strings.Contains(req.CustomerEmail, "@") {
		return db.Booking{}, fmt.Errorf("%w: customerEmail invalid", ErrValidation)
	}
	// Fetch service to get org, duration, buffer, price.
	svc, err := s.repo.GetService(ctx, req.ServiceID)
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: service not found", ErrNotFound)
	}
	orgID := svc.OrganizationID
	// If request specifies org and mismatches service org, reject
	if req.OrganizationID != nil && *req.OrganizationID != orgID {
		return db.Booking{}, fmt.Errorf("%w: organization mismatch", ErrValidation)
	}
	// Fetch staff existence
	st, err := s.repo.GetStaff(ctx, req.StaffID)
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: staff not found", ErrNotFound)
	}
	if st.OrganizationID != orgID {
		return db.Booking{}, fmt.Errorf("%w: staff not in same organization", ErrValidation)
	}
	// Compute endAt as start + duration + buffer (occupied interval for EXCLUDE)
	occupied := time.Duration(svc.DurationMinutes+svc.BufferMinutes) * time.Minute
	if occupied <= 0 {
		occupied = time.Duration(svc.DurationMinutes) * time.Minute
	}
	endAt := req.StartAt.Add(occupied)
	if !endAt.After(req.StartAt) {
		return db.Booking{}, fmt.Errorf("%w: invalid interval", ErrValidation)
	}
	// Resolve org timezone for slot validation date/tz.
	orgTZ := "Asia/Jakarta"
	if org, err2 := s.repo.GetOrganization(ctx, orgID); err2 == nil && org.Timezone != "" {
		orgTZ = org.Timezone
	}
	loc, err := time.LoadLocation(orgTZ)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Jakarta")
		orgTZ = "Asia/Jakarta"
	}
	// Derive calendar date in org location from startAt
	startInLoc := req.StartAt.In(loc)
	dateStr := startInLoc.Format("2006-01-02")

	// Validate via availability engine if available
	if s.avail != nil {
		slots, _, err := s.avail.GetSlots(ctx, req.ServiceID.String(), req.StaffID.String(), dateStr, orgTZ)
		if err != nil {
			// Map not found to 404, invalid to 422
			if errors.Is(err, availability.ErrServiceNotFound) {
				return db.Booking{}, fmt.Errorf("%w: service not found", ErrNotFound)
			}
			// Skill mismatch or other validation — GetSlots returns empty slice, not error; but invalid date returns error
			return db.Booking{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		// Find matching slot by start time UTC
		found := false
		available := false
		for _, sl := range slots {
			if sl.StartAt.Equal(req.StartAt.UTC()) {
				found = true
				available = sl.Available
				break
			}
		}
		if !found {
			// Slot not on grid or outside availability
			return db.Booking{}, fmt.Errorf("%w: slot not on available grid for date %s", ErrSlotUnavailable, dateStr)
		}
		if !available {
			return db.Booking{}, fmt.Errorf("%w: slot already taken or buffer-blocked", ErrSlotUnavailable)
		}
	}

	// Upsert customer (idempotent per org+email)
	var custID pgtype.UUID
	if strings.TrimSpace(req.CustomerEmail) != "" {
		cust, err := s.repo.UpsertCustomer(ctx, db.UpsertCustomerParams{
			OrganizationID: orgID,
			Email:          strings.TrimSpace(strings.ToLower(req.CustomerEmail)),
			Name:           strings.TrimSpace(req.CustomerName),
			Phone:          toPgText(req.CustomerPhone),
		})
			if err == nil {
			// cust.ID is uuid.UUID ([16]byte), pgtype.UUID expects [16]byte
			custID = pgtype.UUID{Bytes: [16]byte(cust.ID), Valid: true}
		} else {
			// Non-fatal: continue without customer_id link (still store denormalized name/email)
			custID = pgtype.UUID{Valid: false}
		}
	}

	// Determine status: free service skip Stripe -> CONFIRMED, else PENDING
	status := "PENDING"
	paymentStatus := "UNPAID"
	if svc.PriceCents == 0 {
		status = "CONFIRMED"
		paymentStatus = "PAID"
	}

	// Insert
	b, err := s.repo.CreateBooking(ctx, db.CreateBookingParams{
		OrganizationID:  orgID,
		ServiceID:       req.ServiceID,
		StaffID:         req.StaffID,
		CustomerID:      custID,
		CustomerName:    strings.TrimSpace(req.CustomerName),
		CustomerEmail:   strings.TrimSpace(strings.ToLower(req.CustomerEmail)),
		CustomerPhone:   toPgText(req.CustomerPhone),
		Notes:           toPgText(req.Notes),
		StartAt:         req.StartAt.UTC(),
		EndAt:           endAt.UTC(),
		Status:          status,
		PaymentStatus:   paymentStatus,
		StripeSessionID: pgtype.Text{Valid: false},
	})
	if err != nil {
		if isExclusionError(err) {
			return db.Booking{}, fmt.Errorf("%w: slot already taken for this staff", ErrConflict)
		}
		return db.Booking{}, fmt.Errorf("create booking: %w", err)
	}
	// Invalidate cache for this staff (and union)
	if s.avail != nil {
		s.avail.ClearCacheForStaff(req.StaffID)
	}
	// Broadcast realtime slot_taken
	_ = safeBroadcast(s.hub, orgID, map[string]interface{}{
		"type":      "slot_taken",
		"staffId":   req.StaffID.String(),
		"startAt":   req.StartAt.UTC().Format(time.RFC3339),
		"endAt":     endAt.UTC().Format(time.RFC3339),
		"bookingId": b.ID.String(),
	})
	return b, nil
}

// List validates RBAC and returns paginated bookings.
func (s *Service) List(ctx context.Context, f ListFilter) (PaginatedResult, error) {
	if f.OrgID == uuid.Nil {
		return PaginatedResult{}, fmt.Errorf("%w: organization required", ErrValidation)
	}
	// RBAC: CUSTOMER cannot list
	if strings.EqualFold(f.Role, "CUSTOMER") {
		return PaginatedResult{}, fmt.Errorf("%w: customer cannot list bookings", ErrForbidden)
	}
	// STAFF only miliknya
	if strings.EqualFold(f.Role, "STAFF") {
		if f.UserID == uuid.Nil {
			return PaginatedResult{}, fmt.Errorf("%w: staff user id required", ErrForbidden)
		}
		st, err := s.repo.GetStaffByUserID(ctx, f.UserID)
		if err != nil {
			// Try fallback: see if userId directly matches staff id? But spec maps staff via user_id
			return PaginatedResult{}, fmt.Errorf("%w: staff record not found for user", ErrForbidden)
		}
		// Enforce filter to own staff
		if f.StaffID != nil && *f.StaffID != st.ID {
			return PaginatedResult{}, fmt.Errorf("%w: staff can only view own bookings", ErrForbidden)
		}
		// Override to own
		id := st.ID
		f.StaffID = &id
	}
	// Defaults for pagination
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	offset := int32((f.Page - 1) * f.Limit)
	limit := int32(f.Limit)

	total, err := s.repo.CountFilteredBookings(ctx, f.OrgID, f.From, f.To, f.Status, f.StaffID)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("count bookings: %w", err)
	}
	items, err := s.repo.ListBookings(ctx, f.OrgID, f.From, f.To, f.Status, f.StaffID, limit, offset)
	if err != nil {
		return PaginatedResult{}, fmt.Errorf("list bookings: %w", err)
	}
	if items == nil {
		items = []db.Booking{}
	}
	totalPages := 0
	if f.Limit > 0 {
		totalPages = int((total + int64(f.Limit) - 1) / int64(f.Limit))
	}
	return PaginatedResult{
		Data:       items,
		Page:       f.Page,
		Limit:      f.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// GetByID returns booking by id with RBAC (STAFF only own, CUSTOMER only own email).
func (s *Service) GetByID(ctx context.Context, id uuid.UUID, orgID *uuid.UUID, role string, userID uuid.UUID) (db.Booking, error) {
	return s.GetByIDWithEmail(ctx, id, orgID, role, userID, "")
}

// GetByIDWithEmail extends GetByID with customer email check for CUSTOMER track.
func (s *Service) GetByIDWithEmail(ctx context.Context, id uuid.UUID, orgID *uuid.UUID, role string, userID uuid.UUID, userEmail string) (db.Booking, error) {
	if id == uuid.Nil {
		return db.Booking{}, fmt.Errorf("%w: invalid id", ErrValidation)
	}
	var b db.Booking
	var err error
	if orgID != nil && *orgID != uuid.Nil {
		b, err = s.repo.GetBookingByIDAndOrg(ctx, id, *orgID)
	} else {
		b, err = s.repo.GetBooking(ctx, id)
	}
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: booking not found", ErrNotFound)
	}
	// STAFF scoping
	if strings.EqualFold(role, "STAFF") {
		if userID == uuid.Nil {
			return db.Booking{}, fmt.Errorf("%w: staff id required", ErrForbidden)
		}
		st, err := s.repo.GetStaffByUserID(ctx, userID)
		if err != nil {
			return db.Booking{}, fmt.Errorf("%w: staff not found", ErrForbidden)
		}
		if b.StaffID != st.ID {
			return db.Booking{}, fmt.Errorf("%w: staff can only view own bookings", ErrForbidden)
		}
	}
	// CUSTOMER can only view own booking by email (track)
	if strings.EqualFold(role, "CUSTOMER") {
		if userEmail != "" && !strings.EqualFold(b.CustomerEmail, userEmail) {
			return db.Booking{}, fmt.Errorf("%w: customer can only view own bookings", ErrForbidden)
		}
		// If email not provided (public track without auth), allow — frontend will have id only
		// But if authenticated CUSTOMER, we already checked email match above when provided
	}
	// OWNER full, CUSTOMER/public allowed for track (no check when no email)
	return b, nil
}

// Cancel marks booking CANCELLED, broadcasts slot_taken.
func (s *Service) Cancel(ctx context.Context, id uuid.UUID, role string, userID uuid.UUID) (db.Booking, error) {
	b, err := s.repo.GetBooking(ctx, id)
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: booking not found", ErrNotFound)
	}
	// RBAC
	if strings.EqualFold(role, "CUSTOMER") {
		return db.Booking{}, fmt.Errorf("%w: customer cannot cancel via this endpoint", ErrForbidden)
	}
	if strings.EqualFold(role, "STAFF") {
		if userID == uuid.Nil {
			return db.Booking{}, fmt.Errorf("%w: staff id required", ErrForbidden)
		}
		st, err := s.repo.GetStaffByUserID(ctx, userID)
		if err != nil {
			return db.Booking{}, fmt.Errorf("%w: staff not found", ErrForbidden)
		}
		if b.StaffID != st.ID {
			return db.Booking{}, fmt.Errorf("%w: staff can only cancel own bookings", ErrForbidden)
		}
	}
	if b.Status == "CANCELLED" {
		return b, nil // idempotent
	}
	updated, err := s.repo.CancelBooking(ctx, id)
	if err != nil {
		return db.Booking{}, fmt.Errorf("cancel booking: %w", err)
	}
	if s.avail != nil {
		s.avail.ClearCacheForStaff(b.StaffID)
	}
	_ = safeBroadcast(s.hub, b.OrganizationID, map[string]interface{}{
		"type":      "slot_taken",
		"action":    "cancel",
		"staffId":   b.StaffID.String(),
		"startAt":   b.StartAt.Format(time.RFC3339),
		"endAt":     b.EndAt.Format(time.RFC3339),
		"bookingId": b.ID.String(),
	})
	return updated, nil
}

// Reschedule validates new slot via GetSlots and executes cancel+create tx semantics (UPDATE in tx).
func (s *Service) Reschedule(ctx context.Context, id uuid.UUID, req RescheduleRequest, role string, userID uuid.UUID) (db.Booking, error) {
	if id == uuid.Nil {
		return db.Booking{}, fmt.Errorf("%w: invalid id", ErrValidation)
	}
	if req.StaffID == uuid.Nil {
		return db.Booking{}, fmt.Errorf("%w: staffId required", ErrValidation)
	}
	if req.StartAt.IsZero() {
		return db.Booking{}, fmt.Errorf("%w: startAt required", ErrValidation)
	}
	existing, err := s.repo.GetBooking(ctx, id)
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: booking not found", ErrNotFound)
	}
	// RBAC similar to cancel
	if strings.EqualFold(role, "CUSTOMER") {
		return db.Booking{}, fmt.Errorf("%w: customer cannot reschedule via this endpoint", ErrForbidden)
	}
	if strings.EqualFold(role, "STAFF") {
		if userID == uuid.Nil {
			return db.Booking{}, fmt.Errorf("%w: staff id required", ErrForbidden)
		}
		st, err := s.repo.GetStaffByUserID(ctx, userID)
		if err != nil {
			return db.Booking{}, fmt.Errorf("%w: staff not found", ErrForbidden)
		}
		if existing.StaffID != st.ID {
			return db.Booking{}, fmt.Errorf("%w: staff can only reschedule own bookings", ErrForbidden)
		}
		// Also if moving to another staff, STAFF cannot reschedule to other staff? For strict RBAC, STAFF reschedule must keep own staffId.
		// But spec says STAFF only miliknya, so new staff must be same as own.
		if req.StaffID != st.ID {
			return db.Booking{}, fmt.Errorf("%w: staff cannot reschedule to another staff", ErrForbidden)
		}
	}
	if existing.Status == "CANCELLED" {
		return db.Booking{}, fmt.Errorf("%w: cannot reschedule cancelled booking", ErrValidation)
	}
	// Need service to compute occupied
	svc, err := s.repo.GetService(ctx, existing.ServiceID)
	if err != nil {
		return db.Booking{}, fmt.Errorf("%w: service not found", ErrNotFound)
	}
	occupied := time.Duration(svc.DurationMinutes+svc.BufferMinutes) * time.Minute
	newEnd := req.StartAt.UTC().Add(occupied)

	// Validate new slot via GetSlots
	orgTZ := "Asia/Jakarta"
	if org, err2 := s.repo.GetOrganization(ctx, existing.OrganizationID); err2 == nil && org.Timezone != "" {
		orgTZ = org.Timezone
	}
	loc, err := time.LoadLocation(orgTZ)
	if err != nil {
		loc, _ = time.LoadLocation("Asia/Jakarta")
		orgTZ = "Asia/Jakarta"
	}
	dateStr := req.StartAt.In(loc).Format("2006-01-02")
	if s.avail != nil {
		slots, _, err := s.avail.GetSlots(ctx, svc.ID.String(), req.StaffID.String(), dateStr, orgTZ)
		if err != nil {
			return db.Booking{}, fmt.Errorf("%w: slot validation failed: %v", ErrValidation, err)
		}
		found := false
		available := false
		for _, sl := range slots {
			if sl.StartAt.Equal(req.StartAt.UTC()) {
				found = true
				available = sl.Available
				// Allow if this is the existing booking's own slot (GetSlots would mark it taken due to its own presence)
				if sl.StartAt.Equal(existing.StartAt) && req.StaffID == existing.StaffID {
					available = true
				}
				break
			}
		}
		if !found {
			return db.Booking{}, fmt.Errorf("%w: slot not on grid for %s", ErrSlotUnavailable, dateStr)
		}
		if !available {
			// Special handling for reschedule: if new interval overlaps old booking's interval on same staff,
			// GetSlots would incorrectly report unavailable because old booking still blocks. But after move, old slot will be free.
			// Allow reschedule if new interval overlaps old interval (since old will be freed); DB EXCLUDE will still catch conflicts with other bookings.
			overlapsOld := req.StaffID == existing.StaffID && req.StartAt.UTC().Before(existing.EndAt) && existing.StartAt.Before(newEnd.UTC())
			if !overlapsOld {
				return db.Booking{}, fmt.Errorf("%w: new slot already taken or buffer-blocked", ErrSlotUnavailable)
			}
			// otherwise treat as available for reschedule attempt — let tx handle true conflicts
		}
	}

	// Execute tx — handles EXCLUDE 23P01 detection
	updated, err := s.repo.RescheduleTx(ctx, id, req.StaffID, req.StartAt.UTC(), newEnd.UTC())
	if err != nil {
		if isExclusionError(err) {
			return db.Booking{}, fmt.Errorf("%w: slot already taken for this staff", ErrConflict)
		}
		if strings.Contains(err.Error(), "already cancelled") || strings.Contains(err.Error(), "cannot reschedule cancelled") {
			return db.Booking{}, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		return db.Booking{}, fmt.Errorf("reschedule: %w", err)
	}
	if s.avail != nil {
		s.avail.ClearCacheForStaff(req.StaffID)
		if req.StaffID != existing.StaffID {
			s.avail.ClearCacheForStaff(existing.StaffID)
		}
	}
	_ = safeBroadcast(s.hub, existing.OrganizationID, map[string]interface{}{
		"type":      "slot_taken",
		"action":    "reschedule",
		"staffId":   req.StaffID.String(),
		"startAt":   req.StartAt.UTC().Format(time.RFC3339),
		"endAt":     newEnd.UTC().Format(time.RFC3339),
		"bookingId": updated.ID.String(),
	})
	return updated, nil
}

func toPgText(s *string) pgtype.Text {
	if s == nil || strings.TrimSpace(*s) == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func isExclusionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23p01") || strings.Contains(msg, "exclusion") || strings.Contains(msg, "no_overlap") || strings.Contains(msg, "conflicting key value violates exclusion")
}

func safeBroadcast(h Hub, orgID uuid.UUID, payload interface{}) error {
	if h == nil {
		return nil
	}
	// recover panic if hub misbehaves
	defer func() { _ = recover() }()
	h.Broadcast(orgID, payload)
	return nil
}
