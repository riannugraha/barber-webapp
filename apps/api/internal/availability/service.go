package availability

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Slot represents a 15m grid slot returned by GetSlots.
// StartAt/EndAt are stored as UTC (timestamptz) — rendered in org timezone / requested tz.
type Slot struct {
	StartAt   time.Time  `json:"startAt"`
	EndAt     time.Time  `json:"endAt"`
	Available bool       `json:"available"`
	StaffID   *uuid.UUID `json:"staffId,omitempty"`
	StaffName string     `json:"staffName,omitempty"`
	Reason    *string    `json:"reason,omitempty"` // "taken" | "buffer" | nil
}

// Service is the calendar engine — implements TECH §5:
// 1. Load availability mingguan + availability_overrides untuk tanggal target
// 2. Generate grid 15m dalam org timezone
// 3. Query WHERE tstzrange(start_at,end_at) && $range untuk clash (PENDING|CONFIRMED only)
// 4. Filter slot muat duration+buffer tanpa overlap
// 5. Cache 30s memory
type Service struct {
	repo  Repository
	cache *slotCache
}

type slotCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	slots  []Slot
	expiry time.Time
	tz     string
}

func newSlotCache() *slotCache {
	return &slotCache{entries: make(map[string]cacheEntry)}
}

func (c *slotCache) get(key string) (cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok {
		return cacheEntry{}, false
	}
	if time.Now().After(e.expiry) {
		return cacheEntry{}, false
	}
	return e, true
}

func (c *slotCache) set(key string, slots []Slot, tz string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// copy slots to avoid mutation
	cp := make([]Slot, len(slots))
	copy(cp, slots)
	c.entries[key] = cacheEntry{slots: cp, expiry: time.Now().Add(ttl), tz: tz}
	// opportunistic cleanup of expired entries (cheap)
	for k, v := range c.entries {
		if time.Now().After(v.expiry) {
			delete(c.entries, k)
		}
	}
}

// NewService creates the calendar engine.
func NewService(repo Repository) *Service {
	return &Service{repo: repo, cache: newSlotCache()}
}

// Errors for handler mapping.
var (
	ErrInvalidServiceID = errors.New("invalid serviceId")
	ErrInvalidStaffID   = errors.New("invalid staffId")
	ErrInvalidDate      = errors.New("invalid date")
	ErrServiceNotFound  = errors.New("service not found")
)

// window is an availability interval in location time.
type window struct {
	start time.Time
	end   time.Time
}

// GetSlots implements the full engine. tz is IANA timezone (e.g. Asia/Jakarta). If empty defaults to org timezone or Asia/Jakarta.
// Returns slots, resolved tz, error.
func (s *Service) GetSlots(ctx context.Context, serviceIDStr, staffIDStr, dateStr, tzStr string) ([]Slot, string, error) {
	// Basic validation — keep errors distinct for 422 mapping.
	if strings.TrimSpace(serviceIDStr) == "" {
		return nil, "", fmt.Errorf("%w: serviceId required", ErrInvalidServiceID)
	}
	svcID, err := uuid.Parse(serviceIDStr)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrInvalidServiceID, err)
	}
	if strings.TrimSpace(dateStr) == "" {
		return nil, "", fmt.Errorf("%w: date required", ErrInvalidDate)
	}
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return nil, "", fmt.Errorf("%w: expected YYYY-MM-DD got %s", ErrInvalidDate, dateStr)
	}
	if staffIDStr != "" {
		if _, err := uuid.Parse(staffIDStr); err != nil {
			return nil, "", fmt.Errorf("%w: %v", ErrInvalidStaffID, err)
		}
	}

	// Fetch service first — need org timezone and duration/buffer.
	svc, err := s.repo.GetService(ctx, svcID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", fmt.Errorf("%w: %s", ErrServiceNotFound, serviceIDStr)
		}
		return nil, "", fmt.Errorf("get service: %w", err)
	}

	// Resolve timezone: requested tz wins, else org timezone, else Asia/Jakarta.
	resolvedTZ := tzStr
	if resolvedTZ == "" {
		// try org timezone
		if org, err2 := s.repo.GetOrganizationByID(ctx, svc.OrganizationID); err2 == nil && org.Timezone != "" {
			resolvedTZ = org.Timezone
		} else {
			resolvedTZ = "Asia/Jakarta"
		}
	}
	loc, err := time.LoadLocation(resolvedTZ)
	if err != nil {
		// fallback to org then default
		fallback := "Asia/Jakarta"
		if org, err2 := s.repo.GetOrganizationByID(ctx, svc.OrganizationID); err2 == nil && org.Timezone != "" {
			if l2, e2 := time.LoadLocation(org.Timezone); e2 == nil {
				loc = l2
				resolvedTZ = org.Timezone
			} else {
				loc, _ = time.LoadLocation(fallback)
				resolvedTZ = fallback
			}
		} else {
			loc, _ = time.LoadLocation(fallback)
			resolvedTZ = fallback
		}
	} else {
		// keep resolvedTZ as requested
	}

	// Cache key includes resolved tz to avoid off-by-hour DST misuse.
	cacheKey := fmt.Sprintf("%s|%s|%s|%s", serviceIDStr, staffIDStr, dateStr, resolvedTZ)
	if e, ok := s.cache.get(cacheKey); ok {
		return e.slots, e.tz, nil
	}

	// Parse day start in resolved location — this is calendar date in that tz.
	// dateStr already validated as YYYY-MM-DD, so parse must succeed.
	dayStartInLoc, _ := time.ParseInLocation("2006-01-02", dateStr, loc)

	duration := time.Duration(svc.DurationMinutes) * time.Minute
	buffer := time.Duration(svc.BufferMinutes) * time.Minute
	occupied := duration + buffer

	// Determine eligible staff list.
	var eligible []db.Staff
	if staffIDStr != "" {
		stID, _ := uuid.Parse(staffIDStr)
		st, err := s.repo.GetStaff(ctx, stID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// return empty, not error — staff not found => 0 slots (skill miss)
				slots := []Slot{}
				s.cache.set(cacheKey, slots, resolvedTZ, 30*time.Second)
				return slots, resolvedTZ, nil
			}
			return nil, "", fmt.Errorf("get staff: %w", err)
		}
		// Skill check: staff must be linked to service via staff_services.
		// ListStaffByService returns eligible staff for this service.
		eligibleForService, err := s.repo.ListStaffByService(ctx, svcID)
		if err != nil {
			return nil, "", fmt.Errorf("list staff by service: %w", err)
		}
		found := false
		for _, es := range eligibleForService {
			if es.ID == st.ID {
				found = true
				break
			}
		}
		if !found {
			// Skill mismatch — e.g., Hair Color hanya Bayu, request Andi => 0 slot
			slots := []Slot{}
			s.cache.set(cacheKey, slots, resolvedTZ, 30*time.Second)
			return slots, resolvedTZ, nil
		}
		eligible = []db.Staff{st}
	} else {
		// Any available — union of all eligible staff.
		list, err := s.repo.ListStaffByService(ctx, svcID)
		if err != nil {
			return nil, "", fmt.Errorf("list staff by service: %w", err)
		}
		eligible = list
		if len(eligible) == 0 {
			slots := []Slot{}
			s.cache.set(cacheKey, slots, resolvedTZ, 30*time.Second)
			return slots, resolvedTZ, nil
		}
	}

	// For each eligible staff generate slots.
	var all []Slot

	// Service cache for buffer/taken distinction per booking service.
	serviceCache := map[uuid.UUID]db.Service{svcID: svc}

	for _, st := range eligible {
		windows, err := s.getAvailabilityWindows(ctx, st.ID, dayStartInLoc, loc)
		if err != nil {
			return nil, "", err
		}
		if len(windows) == 0 {
			continue
		}
		// Overall range for bookings query.
		overallStart := windows[0].start
		overallEnd := windows[0].end
		for _, w := range windows[1:] {
			if w.start.Before(overallStart) {
				overallStart = w.start
			}
			if w.end.After(overallEnd) {
				overallEnd = w.end
			}
		}
		// Query bookings overlapping this staff's availability range.
		// Only PENDING|CONFIRMED block — ListOverlappingBookings already filters.
		bookings, err := s.repo.ListOverlappingBookings(ctx, st.ID, overallStart.UTC(), overallEnd.UTC())
		if err != nil {
			return nil, "", fmt.Errorf("list overlapping bookings: %w", err)
		}

		for _, w := range windows {
			// Generate 15m grid within this window.
			// Slot must fit duration+buffer inside window.
			for cur := w.start; !cur.Add(occupied).After(w.end); cur = cur.Add(15 * time.Minute) {
				// Respect context cancellation
				select {
				case <-ctx.Done():
					return nil, "", ctx.Err()
				default:
				}
				candidateStartUTC := cur.UTC()
				candidateEndUTC := cur.Add(duration).UTC()
				candidateOccupiedEndUTC := cur.Add(occupied).UTC()

				available := true
				var reason *string
				// Check overlap with bookings: occupied interval [candidateStart, candidateOccupiedEnd) overlaps booking [b.StartAt, b.EndAt)
				for _, b := range bookings {
					bStart := b.StartAt
					bEnd := b.EndAt
					// Convert bookings to UTC if not already (they are timestamptz UTC)
					// Overlap condition: candidateStart < bEnd && bStart < candidateOccupiedEnd
					if candidateStartUTC.Before(bEnd) && bStart.Before(candidateOccupiedEndUTC) {
						available = false
						// Determine buffer vs taken.
						// b.EndAt = start+duration+buffer for that booking's service.
						// We need to know booking's service duration to split core vs buffer.
						bSvc, ok := serviceCache[b.ServiceID]
						if !ok {
							if fetched, err2 := s.repo.GetService(ctx, b.ServiceID); err2 == nil {
								bSvc = fetched
								serviceCache[b.ServiceID] = fetched
							} else {
								// fallback to current service's duration
								bSvc = svc
							}
						}
						coreEnd := b.StartAt.Add(time.Duration(bSvc.DurationMinutes) * time.Minute)
						r := "taken"
						if !candidateStartUTC.Before(coreEnd) && candidateStartUTC.Before(bEnd) {
							r = "buffer"
						} else if candidateStartUTC.Before(coreEnd) {
							r = "taken"
						}
						reason = &r
						break
					}
				}

				sID := st.ID
				slot := Slot{
					StartAt:   candidateStartUTC,
					EndAt:     candidateEndUTC,
					Available: available,
					StaffID:   &sID,
					StaffName: st.Name,
					Reason:    reason,
				}
				all = append(all, slot)
			}
		}
	}

	// Handle Any available union deduplication.
	if staffIDStr == "" && len(eligible) > 1 {
		// Merge by start time: available if any staff available at that time.
		type grouped struct {
			start     time.Time
			end       time.Time
			available bool
			staffID   *uuid.UUID
			staffName string
			reason    *string
		}
		m := make(map[int64]grouped) // key unix nano
		for _, sl := range all {
			key := sl.StartAt.UnixNano()
			g, ok := m[key]
			if !ok {
				// first
				m[key] = grouped{
					start:     sl.StartAt,
					end:       sl.EndAt,
					available: sl.Available,
					staffID:   sl.StaffID,
					staffName: sl.StaffName,
					reason:    sl.Reason,
				}
			} else {
				if !g.available && sl.Available {
					// upgrade to available — union true
					g.available = true
					g.staffID = sl.StaffID
					g.staffName = sl.StaffName
					g.reason = nil
					m[key] = g
				}
				// if both unavailable, keep first reason
				// if both available, keep first staff
			}
		}
		var merged []Slot
		for _, g := range m {
			merged = append(merged, Slot{
				StartAt:   g.start,
				EndAt:     g.end,
				Available: g.available,
				StaffID:   g.staffID,
				StaffName: g.staffName,
				Reason:    g.reason,
			})
		}
		all = merged
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].StartAt.Before(all[j].StartAt)
	})

	// Ensure not nil for JSON
	if all == nil {
		all = []Slot{}
	}

	s.cache.set(cacheKey, all, resolvedTZ, 30*time.Second)
	return all, resolvedTZ, nil
}

// getAvailabilityWindows returns intervals in location time for a staff on a given calendar day.
// It respects overrides: is_closed => empty, else override times, else weekly template.
// Uses wall-time construction (time.Date with hour/min) to avoid DST off-by-hour when doing midnight.Add(duration).
func (s *Service) getAvailabilityWindows(ctx context.Context, staffID uuid.UUID, dayStartInLoc time.Time, loc *time.Location) ([]window, error) {
	dateStr := dayStartInLoc.Format("2006-01-02")
	// DATE comparison needs date only — use UTC midnight of that dateStr
	dateForQuery, _ := time.Parse("2006-01-02", dateStr)

	// Try override
	ov, err := s.repo.GetOverrideByStaffAndDate(ctx, staffID, dateForQuery)
	if err == nil {
		if ov.IsClosed {
			return nil, nil
		}
		if ov.StartTime.Valid && ov.EndTime.Valid {
			sDur := time.Duration(ov.StartTime.Microseconds) * time.Microsecond
			eDur := time.Duration(ov.EndTime.Microseconds) * time.Microsecond
			sh, sm, ss := durationToHMS(sDur)
			eh, em, es := durationToHMS(eDur)
			start := time.Date(dayStartInLoc.Year(), dayStartInLoc.Month(), dayStartInLoc.Day(), sh, sm, ss, 0, loc)
			end := time.Date(dayStartInLoc.Year(), dayStartInLoc.Month(), dayStartInLoc.Day(), eh, em, es, 0, loc)
			// overnight support: if end <= start, it spills to next day (21:00 -> 02:00)
			if !end.After(start) {
				end = end.Add(24 * time.Hour)
			}
			return []window{{start: start, end: end}}, nil
		}
		// if override exists but without times and not closed, treat as no availability
		return nil, nil
	}
	if err != nil && !isNoRows(err) {
		return nil, fmt.Errorf("get override: %w", err)
	}

	// Fallback to weekly availability
	list, err := s.repo.ListAvailabilityByStaff(ctx, staffID)
	if err != nil {
		return nil, fmt.Errorf("list availability: %w", err)
	}
	dow := int(dayStartInLoc.Weekday()) // 0 Sunday matches DB 0-6
	var windows []window
	for _, av := range list {
		if int(av.DayOfWeek) != dow {
			continue
		}
		if !av.StartTime.Valid || !av.EndTime.Valid {
			continue
		}
		sDur := time.Duration(av.StartTime.Microseconds) * time.Microsecond
		eDur := time.Duration(av.EndTime.Microseconds) * time.Microsecond
		sh, sm, ss := durationToHMS(sDur)
		eh, em, es := durationToHMS(eDur)
		start := time.Date(dayStartInLoc.Year(), dayStartInLoc.Month(), dayStartInLoc.Day(), sh, sm, ss, 0, loc)
		end := time.Date(dayStartInLoc.Year(), dayStartInLoc.Month(), dayStartInLoc.Day(), eh, em, es, 0, loc)
		if !end.After(start) {
			// overnight
			end = end.Add(24 * time.Hour)
		}
		windows = append(windows, window{start: start, end: end})
	}
	return windows, nil
}

func durationToHMS(d time.Duration) (int, int, int) {
	total := int(d.Seconds())
	h := total / 3600
	total %= 3600
	m := total / 60
	s := total % 60
	return h, m, s
}

// InvalidateCache clears the 30s memory cache (useful for tests or after booking mutations).
func (s *Service) InvalidateCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	s.cache.entries = make(map[string]cacheEntry)
}

// isNoRows checks if err is pgx.ErrNoRows (or wrapped).
func isNoRows(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows")
}

// Helper to convert pgtype.Time to duration (for testing exposed)
func pgTimeToDuration(t pgtype.Time) time.Duration {
	if !t.Valid {
		return 0
	}
	return time.Duration(t.Microseconds) * time.Microsecond
}

// Ensure Service handles context and pgx pooler correctly — no database/sql generic.
var _ = pgTimeToDuration

// Clear cache helper for WS hub broadcast hook (optional)
// Clears all entries that involve this staff, plus Any-available unions (they contain "" staff but share date). For simplicity clear whole cache (cheap, 30s).
func (s *Service) ClearCacheForStaff(staffID uuid.UUID) {
	s.InvalidateCache()
}

// ClearCacheForService clears cache for a service (e.g., after price/duration change).
func (s *Service) ClearCacheForService(serviceID uuid.UUID) {
	s.InvalidateCache()
}
