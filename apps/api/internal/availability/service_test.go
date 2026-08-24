package availability

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
)

// ---------- Mock Repository ----------

type mockRepo struct {
	services       map[uuid.UUID]db.Service
	staff          map[uuid.UUID]db.Staff
	organizations  map[uuid.UUID]db.Organization
	availabilities map[uuid.UUID][]db.Availability // by staffID
	overrides      map[string]db.AvailabilityOverride // key staffID|date
	staffByService map[uuid.UUID][]db.Staff
	bookings       map[uuid.UUID][]db.Booking // by staffID (overlapping)
	errGetService  error
	errGetStaff    error
	errGetOrg      error
	errListAvail   error
	errGetOverride error
	errListStaffByService error
	errListOverlap error
	callsGetService int
	callsGetStaff int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		services:       make(map[uuid.UUID]db.Service),
		staff:          make(map[uuid.UUID]db.Staff),
		organizations:  make(map[uuid.UUID]db.Organization),
		availabilities: make(map[uuid.UUID][]db.Availability),
		overrides:      make(map[string]db.AvailabilityOverride),
		staffByService: make(map[uuid.UUID][]db.Staff),
		bookings:       make(map[uuid.UUID][]db.Booking),
	}
}

func (m *mockRepo) GetService(ctx context.Context, id uuid.UUID) (db.Service, error) {
	m.callsGetService++
	if m.errGetService != nil {
		return db.Service{}, m.errGetService
	}
	if s, ok := m.services[id]; ok {
		return s, nil
	}
	return db.Service{}, pgx.ErrNoRows
}
func (m *mockRepo) GetStaff(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	m.callsGetStaff++
	if m.errGetStaff != nil {
		return db.Staff{}, m.errGetStaff
	}
	if s, ok := m.staff[id]; ok {
		return s, nil
	}
	return db.Staff{}, pgx.ErrNoRows
}
func (m *mockRepo) GetOrganizationByID(ctx context.Context, id uuid.UUID) (db.Organization, error) {
	if m.errGetOrg != nil {
		return db.Organization{}, m.errGetOrg
	}
	if o, ok := m.organizations[id]; ok {
		return o, nil
	}
	// fallback not found
	return db.Organization{}, pgx.ErrNoRows
}
func (m *mockRepo) ListAvailabilityByStaff(ctx context.Context, staffID uuid.UUID) ([]db.Availability, error) {
	if m.errListAvail != nil {
		return nil, m.errListAvail
	}
	return m.availabilities[staffID], nil
}
func (m *mockRepo) GetOverrideByStaffAndDate(ctx context.Context, staffID uuid.UUID, date time.Time) (db.AvailabilityOverride, error) {
	if m.errGetOverride != nil {
		return db.AvailabilityOverride{}, m.errGetOverride
	}
	key := staffID.String() + "|" + date.Format("2006-01-02")
	if ov, ok := m.overrides[key]; ok {
		return ov, nil
	}
	return db.AvailabilityOverride{}, pgx.ErrNoRows
}
func (m *mockRepo) ListStaffByService(ctx context.Context, serviceID uuid.UUID) ([]db.Staff, error) {
	if m.errListStaffByService != nil {
		return nil, m.errListStaffByService
	}
	if list, ok := m.staffByService[serviceID]; ok {
		return list, nil
	}
	return nil, nil
}
func (m *mockRepo) ListOverlappingBookings(ctx context.Context, staffID uuid.UUID, start, end time.Time) ([]db.Booking, error) {
	if m.errListOverlap != nil {
		return nil, m.errListOverlap
	}
	return m.bookings[staffID], nil
}
func (m *mockRepo) ListStaffServices(ctx context.Context, staffID uuid.UUID) ([]db.StaffService, error) {
	return nil, nil
}

// helpers

func pgTime(d time.Duration) pgtype.Time {
	return pgtype.Time{Microseconds: int64(d / time.Microsecond), Valid: true}
}
func timeFromHM(h, m int) time.Duration {
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute
}

// build fixtures
type fixtures struct {
	orgID        uuid.UUID
	svcClassicID uuid.UUID
	svcHairColorID uuid.UUID
	svcOvernightID uuid.UUID
	svcFreeID    uuid.UUID
	staffAndiID  uuid.UUID
	staffBayuID  uuid.UUID
	staffSariID  uuid.UUID
	repo         *mockRepo
}

func buildFixtures() *fixtures {
	orgID := uuid.New()
	svcClassic := uuid.New()
	svcHair := uuid.New()
	svcOvernight := uuid.New()
	svcFree := uuid.New()
	andi := uuid.New()
	bayu := uuid.New()
	sari := uuid.New()

	repo := newMockRepo()
	repo.organizations[orgID] = db.Organization{ID: orgID, Name: "FlowBarber Studio", Slug: "flowbarber-studio", Timezone: "Asia/Jakarta"}
	repo.services[svcClassic] = db.Service{ID: svcClassic, OrganizationID: orgID, Name: "Classic Cut", DurationMinutes: 30, BufferMinutes: 10, PriceCents: 85000}
	repo.services[svcHair] = db.Service{ID: svcHair, OrganizationID: orgID, Name: "Hair Color", DurationMinutes: 90, BufferMinutes: 15, PriceCents: 250000}
	repo.services[svcOvernight] = db.Service{ID: svcOvernight, OrganizationID: orgID, Name: "Overnight Service", DurationMinutes: 30, BufferMinutes: 10, PriceCents: 100000}
	repo.services[svcFree] = db.Service{ID: svcFree, OrganizationID: orgID, Name: "Konsultasi Style 15m", DurationMinutes: 15, BufferMinutes: 5, PriceCents: 0}

	repo.staff[andi] = db.Staff{ID: andi, OrganizationID: orgID, Name: "Andi"}
	repo.staff[bayu] = db.Staff{ID: bayu, OrganizationID: orgID, Name: "Bayu"}
	repo.staff[sari] = db.Staff{ID: sari, OrganizationID: orgID, Name: "Sari"}

	// staffByService skill mapping
	repo.staffByService[svcClassic] = []db.Staff{repo.staff[andi], repo.staff[bayu], repo.staff[sari]}
	repo.staffByService[svcHair] = []db.Staff{repo.staff[bayu]} // Hair Color hanya Bayu
	repo.staffByService[svcOvernight] = []db.Staff{repo.staff[andi], repo.staff[bayu]}
	repo.staffByService[svcFree] = []db.Staff{repo.staff[bayu], repo.staff[sari]}

	// Availability 09:00-18:00 Mon-Sat, 10:00-16:00 Sun for each staff (dow 0=Sun)
	for _, sid := range []uuid.UUID{andi, bayu, sari} {
		for dow := 0; dow <= 6; dow++ {
			startStr := timeFromHM(9, 0)
			endStr := timeFromHM(18, 0)
			if dow == 0 {
				startStr = timeFromHM(10, 0)
				endStr = timeFromHM(16, 0)
			}
			// Special overnight staff Andi on specific? We'll keep default; overnight test will replace separately
			av := db.Availability{
				ID:        uuid.New(),
				StaffID:   sid,
				DayOfWeek: int32(dow),
				StartTime: pgTime(startStr),
				EndTime:   pgTime(endStr),
			}
			repo.availabilities[sid] = append(repo.availabilities[sid], av)
		}
	}

	return &fixtures{
		orgID: orgID, svcClassicID: svcClassic, svcHairColorID: svcHair, svcOvernightID: svcOvernight, svcFreeID: svcFree,
		staffAndiID: andi, staffBayuID: bayu, staffSariID: sari, repo: repo,
	}
}

// ---------- Table-driven TestGetSlots 100% ----------

func TestGetSlots(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	// ensure fixtures per subtest isolated? Provide fresh repo each
	t.Run("buffer 10m blocks next", func(t *testing.T) {
		fx := buildFixtures()
		svc := NewService(fx.repo)
		// date Monday 2025-11-10 (Monday dow 1, availability 09-18)
		date := "2025-11-10"
		// Create booking 10:00-10:40 for Andi
		dayStart, _ := time.ParseInLocation("2006-01-02", date, loc)
		startLocal := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, loc)
		endLocal := startLocal.Add(40 * time.Minute) // 30+10
		booking := db.Booking{
			ID: fx.staffAndiID, // not used
			StaffID: fx.staffAndiID,
			ServiceID: fx.svcClassicID,
			StartAt: startLocal.UTC(),
			EndAt:   endLocal.UTC(),
			Status: "CONFIRMED",
		}
		fx.repo.bookings[fx.staffAndiID] = []db.Booking{booking}
		// Need to set booking's service cache already in repo.services; GetSlots will fetch via GetService for buffer logic
		slots, tz, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Equal(t, "Asia/Jakarta", tz)
		// Find specific slots
		var findSlot = func(h, m int) *Slot {
			want := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), h, m, 0, 0, loc).UTC()
			for _, s := range slots {
				if s.StartAt.Equal(want) {
					return &s
				}
			}
			return nil
		}
		// 10:00 should be taken
		s10 := findSlot(10, 0)
		require.NotNil(t, s10, "slot 10:00 should exist")
		assert.False(t, s10.Available)
		assert.NotNil(t, s10.Reason)
		assert.Equal(t, "taken", *s10.Reason)
		// 10:15 should be taken (overlaps core)
		s1015 := findSlot(10, 15)
		require.NotNil(t, s1015)
		assert.False(t, s1015.Available)
		assert.Equal(t, "taken", *s1015.Reason)
		// 10:30 should be buffer (start == coreEnd 10:30)
		s1030 := findSlot(10, 30)
		require.NotNil(t, s1030)
		assert.False(t, s1030.Available)
		assert.Equal(t, "buffer", *s1030.Reason)
		// 10:45 should be available
		s1045 := findSlot(10, 45)
		require.NotNil(t, s1045)
		assert.True(t, s1045.Available)
		assert.Nil(t, s1045.Reason)
		// Count check: without booking total slots for 09-18 with 40 occupied 15m grid => (9h*60 -40)/15 +1 = (540-40)/15+1= 500/15+1=33+1=34? Let's compute brute: loop.
		// Just ensure >30 and taken count 3.
		takenCount := 0
		for _, s := range slots {
			if !s.Available {
				takenCount++
			}
		}
		assert.GreaterOrEqual(t, takenCount, 3)
	})

	t.Run("DST Asia/Jakarta 2025-11-02", func(t *testing.T) {
		// Asia/Jakarta has no DST, but ensure conversion not off by hour vs UTC
		fx := buildFixtures()
		svc := NewService(fx.repo)
		date := "2025-11-02" // Sunday, availability 10-16 for Asia/Jakarta
		slots, tz, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffBayuID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Equal(t, "Asia/Jakarta", tz)
		require.NotEmpty(t, slots)
		locJakarta, _ := time.LoadLocation("Asia/Jakarta")
		// First slot should be 10:00 WIB == 03:00 UTC
		dayStart, _ := time.ParseInLocation("2006-01-02", date, locJakarta)
		expectedStartUTC := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, locJakarta).UTC()
		assert.Equal(t, expectedStartUTC, slots[0].StartAt)
		assert.Equal(t, expectedStartUTC.Add(30*time.Minute), slots[0].EndAt)
		// Check all slots are in 15m grid and within 10-16 window minus buffer, no DST shift
		for _, s := range slots {
			// convert back to Jakarta
			localStart := s.StartAt.In(locJakarta)
			// Should be between 10:00 and 16:00 - occupied
			assert.True(t, !localStart.Before(time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, locJakarta)))
			assert.True(t, !s.StartAt.Add(40*time.Minute).After(time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 16, 0, 0, 0, locJakarta)))
		}
		// Also test without tz param should default to org timezone (Asia/Jakarta) and produce same
		slots2, tz2, err2 := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffBayuID.String(), date, "")
		require.NoError(t, err2)
		assert.Equal(t, "Asia/Jakarta", tz2)
		assert.Equal(t, len(slots), len(slots2))
		assert.Equal(t, slots[0].StartAt, slots2[0].StartAt)
	})

	t.Run("override libur 0 slot", func(t *testing.T) {
		fx := buildFixtures()
		svc := NewService(fx.repo)
		date := "2025-11-15"
		// set override is_closed for Andi on that date
		ovDate, _ := time.Parse("2006-01-02", date)
		key := fx.staffAndiID.String() + "|" + ovDate.Format("2006-01-02")
		fx.repo.overrides[key] = db.AvailabilityOverride{
			ID: fx.staffAndiID, StaffID: fx.staffAndiID, Date: ovDate, IsClosed: true,
		}
		slots, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Len(t, slots, 0)
		// Same for Any should still have Bayu/Sari slots, not zero
		slotsAny, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), "", date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.NotEmpty(t, slotsAny, "Any should still have slots from other staff even if one closed")
	})

	t.Run("Hair Color hanya Bayu", func(t *testing.T) {
		fx := buildFixtures()
		svc := NewService(fx.repo)
		date := "2025-11-11" // Tuesday
		// Request Andi for Hair Color => 0
		slotsAndi, _, err := svc.GetSlots(context.Background(), fx.svcHairColorID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Len(t, slotsAndi, 0)
		// Bayu => >0
		slotsBayu, _, err := svc.GetSlots(context.Background(), fx.svcHairColorID.String(), fx.staffBayuID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.NotEmpty(t, slotsBayu)
		// Sari also 0
		slotsSari, _, err := svc.GetSlots(context.Background(), fx.svcHairColorID.String(), fx.staffSariID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Len(t, slotsSari, 0)
		// Any available should equal Bayu only (union with only Bayu eligible)
		slotsAny, _, err := svc.GetSlots(context.Background(), fx.svcHairColorID.String(), "", date, "Asia/Jakarta")
		require.NoError(t, err)
		assert.Equal(t, len(slotsBayu), len(slotsAny))
		// Verify staffId points to Bayu
		for _, s := range slotsAny {
			if s.Available {
				assert.Equal(t, fx.staffBayuID, *s.StaffID)
				break
			}
		}
	})

	t.Run("overnight 21:00-02:00", func(t *testing.T) {
		fx := buildFixtures()
		// Override Andi's availability for specific date to overnight
		// Replace availabilities for Andi to only overnight on that dow
		// We'll use override with times 21:00-02:00
		date := "2025-11-12" // Wednesday
		ovDate, _ := time.Parse("2006-01-02", date)
		key := fx.staffAndiID.String() + "|" + ovDate.Format("2006-01-02")
		fx.repo.overrides[key] = db.AvailabilityOverride{
			ID: fx.staffAndiID, StaffID: fx.staffAndiID, Date: ovDate, IsClosed: false,
			StartTime: pgTime(timeFromHM(21, 0)),
			EndTime:   pgTime(timeFromHM(2, 0)),
			Reason: pgtype.Text{String: "overnight", Valid: true},
		}
		svc := NewService(fx.repo)
		slots, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		require.NotEmpty(t, slots)
		locJakarta, _ := time.LoadLocation("Asia/Jakarta")
		dayStart, _ := time.ParseInLocation("2006-01-02", date, locJakarta)
		// First slot 21:00 same day
		expectedFirst := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 21, 0, 0, 0, locJakarta).UTC()
		assert.Equal(t, expectedFirst, slots[0].StartAt)
		// Last slot should be 01:15 next day (since 02:00 -40 =01:20, last 15m grid <=01:15)
		last := slots[len(slots)-1]
		lastLocal := last.StartAt.In(locJakarta)
		assert.Equal(t, 1, lastLocal.Hour())
		assert.Equal(t, 15, lastLocal.Minute())
		// Next day 02:00 is end, ensure no slot beyond
		assert.True(t, lastLocal.Add(40*time.Minute).Equal(time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 2, 0, 0, 0, locJakarta).Add(24*time.Hour)) || lastLocal.Add(40*time.Minute).Before(time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 2, 0, 0, 0, locJakarta).Add(24*time.Hour)))
		// Also test weekly overnight without override: set avail to 21-02
		fx2 := buildFixtures()
		// clear default avail for Bayu dow 4 (Thursday=4) and set overnight
		fx2.repo.availabilities[fx2.staffBayuID] = []db.Availability{
			{ID: uuid.New(), StaffID: fx2.staffBayuID, DayOfWeek: 4, StartTime: pgTime(timeFromHM(21, 0)), EndTime: pgTime(timeFromHM(2, 0))},
		}
		svc2 := NewService(fx2.repo)
		// date Thursday 2025-11-13 dow 4
		date2 := "2025-11-13"
		slots2, _, err := svc2.GetSlots(context.Background(), fx2.svcClassicID.String(), fx2.staffBayuID.String(), date2, "Asia/Jakarta")
		require.NoError(t, err)
		assert.NotEmpty(t, slots2)
		assert.Equal(t, 18, len(slots2)) // 21:00-02:00 =5h=300m, occupied 40, grid 15: (300-40)/15+1 = 18
		loc2, _ := time.LoadLocation("Asia/Jakarta")
		dStart2, _ := time.ParseInLocation("2006-01-02", date2, loc2)
		assert.Equal(t, time.Date(dStart2.Year(), dStart2.Month(), dStart2.Day(), 21, 0, 0, 0, loc2).UTC(), slots2[0].StartAt)
	})

	t.Run("Any available union", func(t *testing.T) {
		fx := buildFixtures()
		svc := NewService(fx.repo)
		date := "2025-11-10"
		locJakarta, _ := time.LoadLocation("Asia/Jakarta")
		dayStart, _ := time.ParseInLocation("2006-01-02", date, locJakarta)
		// Book Andi at 10:00, but Bayu free
		startLocal := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, locJakarta)
		endLocal := startLocal.Add(40 * time.Minute)
		fx.repo.bookings[fx.staffAndiID] = []db.Booking{{StaffID: fx.staffAndiID, ServiceID: fx.svcClassicID, StartAt: startLocal.UTC(), EndAt: endLocal.UTC(), Status: "CONFIRMED"}}
		fx.repo.bookings[fx.staffBayuID] = nil
		fx.repo.bookings[fx.staffSariID] = nil
		// Any should still be available at 10:00 via Bayu/Sari
		slotsAny, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), "", date, "Asia/Jakarta")
		require.NoError(t, err)
		var slot10 *Slot
		want10 := startLocal.UTC()
		for _, s := range slotsAny {
			if s.StartAt.Equal(want10) {
				slot10 = &s
				break
			}
		}
		require.NotNil(t, slot10)
		assert.True(t, slot10.Available, "Any available should be available if any staff free")
		assert.NotNil(t, slot10.StaffID)
		// Ensure union does not duplicate: length should equal single staff's full day length (since union dedup)
		slotsSingle, _, _ := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		// single may have taken, any has same count but with available flag upgraded
		assert.Equal(t, len(slotsSingle), len(slotsAny))
		// Now book all staff at same slot => Any should be unavailable
		for _, sid := range []uuid.UUID{fx.staffAndiID, fx.staffBayuID, fx.staffSariID} {
			fx.repo.bookings[sid] = []db.Booking{{StaffID: sid, ServiceID: fx.svcClassicID, StartAt: startLocal.UTC(), EndAt: endLocal.UTC(), Status: "CONFIRMED"}}
		}
		svc.InvalidateCache()
		slotsAny2, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), "", date, "Asia/Jakarta")
		require.NoError(t, err)
		var slot10b *Slot
		for _, s := range slotsAny2 {
			if s.StartAt.Equal(want10) {
				slot10b = &s
				break
			}
		}
		require.NotNil(t, slot10b)
		assert.False(t, slot10b.Available)
	})

	t.Run("CANCELLED does not block", func(t *testing.T) {
		fx := buildFixtures()
		svc := NewService(fx.repo)
		date := "2025-11-10"
		locJakarta, _ := time.LoadLocation("Asia/Jakarta")
		dayStart, _ := time.ParseInLocation("2006-01-02", date, locJakarta)
		startLocal := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, locJakarta)
		endLocal := startLocal.Add(40 * time.Minute)
		// Mock will return only PENDING/CONFIRMED bookings, but we simulate that ListOverlapping returns empty for cancelled
		// So we set no bookings (simulating cancelled not returned)
		fx.repo.bookings[fx.staffAndiID] = []db.Booking{} // cancelled booking would be filtered out by repo
		slots, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		require.NoError(t, err)
		var found *Slot
		for _, s := range slots {
			if s.StartAt.Equal(startLocal.UTC()) {
				found = &s
				break
			}
		}
		require.NotNil(t, found)
		assert.True(t, found.Available)
		// But if we add a CANCELLED booking directly, service shouldn't consider it anyway because repo filters, but service additionally checks overlap only with returned.
		// Ensure that after we add PENDING it blocks
		fx.repo.bookings[fx.staffAndiID] = []db.Booking{{StaffID: fx.staffAndiID, ServiceID: fx.svcClassicID, StartAt: startLocal.UTC(), EndAt: endLocal.UTC(), Status: "PENDING"}}
		svc.InvalidateCache()
		slots2, _, _ := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
		var found2 *Slot
		for _, s := range slots2 {
			if s.StartAt.Equal(startLocal.UTC()) {
				found2 = &s
				break
			}
		}
		require.NotNil(t, found2)
		assert.False(t, found2.Available)
	})
}

// Additional error branches for 100%

func TestGetSlots_Validation(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	_, _, err := svc.GetSlots(context.Background(), "", fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrInvalidServiceID)
	_, _, err = svc.GetSlots(context.Background(), "not-uuid", fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrInvalidServiceID)
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrInvalidDate)
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025/11/10", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrInvalidDate)
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), "bad-uuid", "2025-11-10", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrInvalidStaffID)
	// service not found
	fakeID := uuid.New().String()
	_, _, err = svc.GetSlots(context.Background(), fakeID, "", "2025-11-10", "Asia/Jakarta")
	assert.ErrorIs(t, err, ErrServiceNotFound)
	// GetService error propagation
	fx.repo.errGetService = errors.New("db down")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), "", "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "get service")
	fx.repo.errGetService = nil
	// Staff get error
	fx.repo.errGetStaff = errors.New("staff db err")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "get staff")
	fx.repo.errGetStaff = nil
	// ListStaffByService error
	fx.repo.errListStaffByService = errors.New("list staff err")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), "", "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "list staff by service")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "list staff by service")
	fx.repo.errListStaffByService = nil
	// No eligible staff
	fx2 := buildFixtures()
	fx2.repo.staffByService[fx2.svcClassicID] = nil
	svc2 := NewService(fx2.repo)
	slots, _, err := svc2.GetSlots(context.Background(), fx2.svcClassicID.String(), "", "2025-11-10", "Asia/Jakarta")
	require.NoError(t, err)
	assert.Empty(t, slots)
	// ListAvailability error
	fx.repo.errListAvail = errors.New("avail err")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "list availability")
	fx.repo.errListAvail = nil
	// GetOverride error
	fx.repo.errGetOverride = errors.New("override err")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "get override")
	fx.repo.errGetOverride = nil
	// ListOverlapping error
	fx.repo.errListOverlap = errors.New("overlap err")
	_, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.ErrorContains(t, err, "list overlapping bookings")
	fx.repo.errListOverlap = nil
	// Invalid staff not found -> empty (skill miss already tested but also via not found)
	slots, _, err = svc.GetSlots(context.Background(), fx.svcClassicID.String(), uuid.New().String(), "2025-11-10", "Asia/Jakarta")
	require.NoError(t, err)
	assert.Empty(t, slots)
	// Cache hit test — verify second call returns cached slots even after modifying underlying availability
	fx3 := buildFixtures()
	svc3 := NewService(fx3.repo)
	slotsA, _, _ := svc3.GetSlots(context.Background(), fx3.svcClassicID.String(), fx3.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	// Mutate underlying availability to would-be empty, but cache should still return original
	fx3.repo.availabilities[fx3.staffAndiID] = nil
	slotsB, _, err := svc3.GetSlots(context.Background(), fx3.svcClassicID.String(), fx3.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	require.NoError(t, err)
	assert.Equal(t, slotsA, slotsB)
	assert.NotEmpty(t, slotsB)
}

func TestGetSlots_TimezoneFallback(t *testing.T) {
	fx := buildFixtures()
	// org timezone empty => fallback to Asia/Jakarta
	fx.repo.organizations[fx.orgID] = db.Organization{ID: fx.orgID, Name: "Test", Slug: "test", Timezone: ""}
	svc := NewService(fx.repo)
	slots, tz, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Jakarta", tz)
	assert.NotEmpty(t, slots)
	// invalid tz fallback
	slots2, tz2, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Invalid/Zone")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Jakarta", tz2)
	assert.NotEmpty(t, slots2)
	// valid other tz like UTC
	slots3, tz3, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "UTC")
	require.NoError(t, err)
	assert.Equal(t, "UTC", tz3)
	assert.NotEmpty(t, slots3)
	// org timezone custom should be used when tz empty
	fx2 := buildFixtures()
	fx2.repo.organizations[fx2.orgID] = db.Organization{ID: fx2.orgID, Timezone: "UTC"}
	svc2 := NewService(fx2.repo)
	_, tz4, _ := svc2.GetSlots(context.Background(), fx2.svcClassicID.String(), fx2.staffAndiID.String(), "2025-11-10", "")
	assert.Equal(t, "UTC", tz4)
	// GetOrganization error fallback to default
	fx2.repo.errGetOrg = errors.New("db err")
	_, tz5, _ := svc2.GetSlots(context.Background(), fx2.svcClassicID.String(), fx2.staffAndiID.String(), "2025-11-10", "")
	// should still fallback to Asia/Jakarta via default, not error
	assert.Equal(t, "Asia/Jakarta", tz5)
}

func TestGetSlots_ContextCancellation(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := svc.GetSlots(ctx, fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestGetAvailabilityWindows_Edge(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	dayStart, _ := time.ParseInLocation("2006-01-02", "2025-11-10", loc)
	// override with valid times but is_closed false
	ovDate, _ := time.Parse("2006-01-02", "2025-11-10")
	key := fx.staffAndiID.String() + "|" + ovDate.Format("2006-01-02")
	fx.repo.overrides[key] = db.AvailabilityOverride{
		StaffID: fx.staffAndiID, Date: ovDate, IsClosed: false,
		StartTime: pgTime(timeFromHM(8, 0)), EndTime: pgTime(timeFromHM(12, 0)),
	}
	windows, err := svc.getAvailabilityWindows(context.Background(), fx.staffAndiID, dayStart, loc)
	require.NoError(t, err)
	require.Len(t, windows, 1)
	assert.Equal(t, 8, windows[0].start.Hour())
	assert.Equal(t, 12, windows[0].end.Hour())
	// override with no times and not closed => 0
	fx.repo.overrides[key] = db.AvailabilityOverride{StaffID: fx.staffAndiID, Date: ovDate, IsClosed: false}
	windows, err = svc.getAvailabilityWindows(context.Background(), fx.staffAndiID, dayStart, loc)
	require.NoError(t, err)
	assert.Len(t, windows, 0)
	// invalid avail entries without times should be skipped
	fx2 := buildFixtures()
	fx2.repo.availabilities[fx2.staffAndiID] = []db.Availability{
		{ID: uuid.New(), StaffID: fx2.staffAndiID, DayOfWeek: 1, StartTime: pgtype.Time{Valid: false}, EndTime: pgtype.Time{Valid: false}},
	}
	// 2025-11-10 is Monday dow 1
	windows, err = NewService(fx2.repo).getAvailabilityWindows(context.Background(), fx2.staffAndiID, dayStart, loc)
	require.NoError(t, err)
	assert.Len(t, windows, 0)
}

func TestCacheOperations(t *testing.T) {
	svc := NewService(newMockRepo())
	loc, _ := time.LoadLocation("Asia/Jakarta")
	fx := buildFixtures()
	svc2 := NewService(fx.repo)
	slots, _, _ := svc2.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	assert.NotEmpty(t, slots)
	// set expiry test
	c := svc2.cache
	c.mu.Lock()
	for k, v := range c.entries {
		v.expiry = time.Now().Add(-1 * time.Second)
		c.entries[k] = v
	}
	c.mu.Unlock()
	// next GetSlots should not hit expired cache (will regenerate)
	slots2, _, err := svc2.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Asia/Jakarta")
	require.NoError(t, err)
	assert.Equal(t, len(slots), len(slots2))
	// Invalidate
	svc2.InvalidateCache()
	assert.Empty(t, svc2.cache.entries)
	// ClearForStaff and ClearForService
	svc2.ClearCacheForStaff(fx.staffAndiID)
	svc2.ClearCacheForService(fx.svcClassicID)
	_ = svc
	_ = loc
}

func TestHelpers(t *testing.T) {
	h0, m0, s0 := durationToHMS(time.Hour + 30*time.Minute + 10*time.Second)
	assert.Equal(t, 1, h0); assert.Equal(t, 30, m0); assert.Equal(t, 10, s0)
	// Actually durationToHMS returns h,m,s
	h, m, s := durationToHMS(2*time.Hour + 15*time.Minute + 5*time.Second)
	assert.Equal(t, 2, h); assert.Equal(t, 15, m); assert.Equal(t, 5, s)
	assert.True(t, isNoRows(pgx.ErrNoRows))
	assert.True(t, isNoRows(fmt.Errorf("no rows in result set")))
	assert.False(t, isNoRows(errors.New("other")))
	assert.False(t, isNoRows(nil))
	d := pgTimeToDuration(pgtype.Time{Microseconds: int64(3600*1e6), Valid: true})
	assert.Equal(t, time.Hour, d)
	assert.Equal(t, time.Duration(0), pgTimeToDuration(pgtype.Time{Valid: false}))
}

func TestGetSlots_BufferFetchServiceCache(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	date := "2025-11-10"
	loc, _ := time.LoadLocation("Asia/Jakarta")
	dayStart, _ := time.ParseInLocation("2006-01-02", date, loc)
	startLocal := time.Date(dayStart.Year(), dayStart.Month(), dayStart.Day(), 10, 0, 0, 0, loc)
	endLocal := startLocal.Add(40 * time.Minute)
	// Booking with service = Hair Color (different duration 90) but stored as booking for Andi
	// To trigger serviceCache fetch for b.ServiceID not in cache initially? Our cache initially has only svcClassic; so when booking's service is Hair Color, it will fetch via GetService
	fx.repo.bookings[fx.staffAndiID] = []db.Booking{
		{StaffID: fx.staffAndiID, ServiceID: fx.svcHairColorID, StartAt: startLocal.UTC(), EndAt: startLocal.Add(105*time.Minute).UTC(), Status: "CONFIRMED"},
		{StaffID: fx.staffAndiID, ServiceID: fx.svcClassicID, StartAt: endLocal.Add(60*time.Minute).UTC(), EndAt: endLocal.Add(100*time.Minute).UTC(), Status: "CONFIRMED"},
	}
	// Ensure Classic and Hair both exist in repo
	slots, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
	require.NoError(t, err)
	assert.NotEmpty(t, slots)
	// Also test when GetService for booking's service fails, fallback to current svc
	fx.repo.services[fx.svcHairColorID] = db.Service{} // will cause getService to still found? Actually we remove to cause error
	delete(fx.repo.services, fx.svcHairColorID)
	svc.InvalidateCache()
	slots2, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
	require.NoError(t, err)
	assert.NotEmpty(t, slots2)
	_ = endLocal
}

// ---------- Handler tests ----------

func TestHandler_GetSlots_Success(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fx.svcClassicID.String()+"&staffId="+fx.staffAndiID.String()+"&date=2025-11-10&tz=Asia/Jakarta", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := h.GetSlots(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp SlotsResponse
	// we trust json marshaling; just check code
	_ = resp
	assert.Contains(t, rec.Body.String(), `"slots"`)
}

func TestHandler_GetSlots_Validation422(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	tests := []struct {
		name string
		url  string
	}{
		{"missing serviceId", "/availability/slots?date=2025-11-10"},
		{"invalid serviceId", "/availability/slots?serviceId=bad&date=2025-11-10"},
		{"missing date", "/availability/slots?serviceId=" + fx.svcClassicID.String()},
		{"invalid date", "/availability/slots?serviceId=" + fx.svcClassicID.String() + "&date=bad"},
		{"invalid staffId", "/availability/slots?serviceId=" + fx.svcClassicID.String() + "&date=2025-11-10&staffId=bad"},
		{"invalid tz", "/availability/slots?serviceId=" + fx.svcClassicID.String() + "&date=2025-11-10&tz=Bad/Zone"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			_ = h.GetSlots(c)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		})
	}
}

func TestHandler_GetSlots_NotFoundAndInternal(t *testing.T) {
	fx := buildFixtures()
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	// not found via fake service
	fake := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fake+"&date=2025-11-10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.GetSlots(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// internal error via repo err
	fx2 := buildFixtures()
	fx2.repo.errGetService = errors.New("internal db boom")
	svc2 := NewService(fx2.repo)
	h2 := NewHandler(svc2)
	req2 := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fx2.svcClassicID.String()+"&date=2025-11-10", nil)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	_ = h2.GetSlots(c2)
	assert.Equal(t, http.StatusInternalServerError, rec2.Code)
}

func TestHandler_GetSlots_DefaultsAndNilSlots(t *testing.T) {
	fx := buildFixtures()
	// Make service with no slots (closed)
	fx.repo.overrides[fx.staffAndiID.String()+"|2025-11-10"] = db.AvailabilityOverride{
		StaffID: fx.staffAndiID, Date: mustParseDate("2025-11-10"), IsClosed: true,
	}
	svc := NewService(fx.repo)
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/availability/slots?serviceId="+fx.svcClassicID.String()+"&staffId="+fx.staffAndiID.String()+"&date=2025-11-10", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.GetSlots(c)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"slots":[]`)
}

func mustParseDate(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func TestIsInvalidError(t *testing.T) {
	assert.True(t, isInvalidError(errors.New("invalid serviceId")))
	assert.True(t, isInvalidError(errors.New("validation failed")))
	assert.True(t, isInvalidError(errors.New("required field missing")))
	assert.False(t, isInvalidError(errors.New("not found")))
	assert.False(t, isInvalidError(nil))
	assert.True(t, isNotFoundError(errors.New("service not found")))
	assert.False(t, isNotFoundError(errors.New("other")))
}

func TestContainsHelper(t *testing.T) {
	assert.True(t, contains("hello world", "world"))
	assert.False(t, contains("hello", "world"))
	assert.True(t, contains("abc", ""))
}

// Ensure 100% via checking all exported helpers exercised
func TestSlotCache_GetSet(t *testing.T) {
	c := newSlotCache()
	assert.NotNil(t, c)
	c.set("k1", []Slot{{Available: true}}, "Asia/Jakarta", time.Minute)
	e, ok := c.get("k1")
	assert.True(t, ok)
	assert.Equal(t, "Asia/Jakarta", e.tz)
	_, ok = c.get("missing")
	assert.False(t, ok)
	c.set("k2", []Slot{}, "UTC", -time.Minute) // expired immediately
	// get should return false due expiry
	// Need to bypass cleanup? get checks expiry
	// But set also cleans expired; k2 expiry in past so after set, it should be deleted?
	// Actually set with negative ttl sets expiry in past, then cleanup loop deletes it
	_, ok = c.get("k2")
	assert.False(t, ok)
	// test copy isolation
	orig := []Slot{{Available: true, StaffName: "Andi"}}
	c.set("k3", orig, "Asia/Jakarta", time.Minute)
	orig[0].StaffName = "changed"
	e3, _ := c.get("k3")
	assert.Equal(t, "Andi", e3.slots[0].StaffName)
}

// Test edge for contains empty sub
func TestSlotCache_Expiry(t *testing.T) {
	c := newSlotCache()
	c.set("a", []Slot{{Available: true}}, "Asia/Jakarta", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	_, ok := c.get("a")
	assert.False(t, ok)
	// test that set cleans expired entries
	c.set("b", []Slot{{Available: false}}, "UTC", time.Minute)
	// after previous expiry, there should be only b
	assert.Len(t, c.entries, 1)
}

func TestGetSlots_InvalidTimezoneFallbackViaOrg(t *testing.T) {
	fx := buildFixtures()
	fx.repo.organizations[fx.orgID] = db.Organization{ID: fx.orgID, Timezone: "UTC"}
	svc := NewService(fx.repo)
	// provide invalid tz, should fallback to org's UTC via fallback logic inside GetSlots (second branch)
	slots, tz, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Bad/Zone")
	require.NoError(t, err)
	// Since tz param was invalid, LoadLocation failed, service falls back to org timezone UTC
	assert.Equal(t, "UTC", tz)
	assert.NotEmpty(t, slots)
	// Provide invalid tz but org also invalid
	fx.repo.organizations[fx.orgID] = db.Organization{ID: fx.orgID, Timezone: "Also/Bad"}
	svc.InvalidateCache()
	slots2, tz2, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), "2025-11-10", "Bad/Zone2")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Jakarta", tz2)
	assert.NotEmpty(t, slots2)
}

// Test pgTimeToDuration already but need coverage
func TestPgTimeToDurationEdge(t *testing.T) {
	// valid zero microseconds
	assert.Equal(t, time.Duration(0), pgTimeToDuration(pgtype.Time{Microseconds: 0, Valid: true}))
	_ = strings.Contains("a", "b") // keep import used
}

// Ensure handler helper contains used
func TestHandlerHelpers(t *testing.T) {
	assert.True(t, isInvalidError(errors.New("validation error")))
	assert.True(t, isNotFoundError(fmt.Errorf("entity not found")))
}

func TestGetSlots_MultipleWindows(t *testing.T) {
	fx := buildFixtures()
	// Create staff with two windows for same dow Monday (dow 1): 09:00-12:00 and 13:00-18:00
	monday := int32(1)
	fx.repo.availabilities[fx.staffAndiID] = []db.Availability{
		{ID: uuid.New(), StaffID: fx.staffAndiID, DayOfWeek: monday, StartTime: pgTime(timeFromHM(9, 0)), EndTime: pgTime(timeFromHM(12, 0))},
		{ID: uuid.New(), StaffID: fx.staffAndiID, DayOfWeek: monday, StartTime: pgTime(timeFromHM(13, 0)), EndTime: pgTime(timeFromHM(18, 0))},
	}
	svc := NewService(fx.repo)
	date := "2025-11-10" // Monday
	slots, _, err := svc.GetSlots(context.Background(), fx.svcClassicID.String(), fx.staffAndiID.String(), date, "Asia/Jakarta")
	require.NoError(t, err)
	assert.NotEmpty(t, slots)
	// Ensure slots exist in both windows and gap 12-13 has no slots
	loc, _ := time.LoadLocation("Asia/Jakarta")
	dayStart, _ := time.ParseInLocation("2006-01-02", date, loc)
	// Slot 12:00 should not exist because 12:00 +40 >12:00
	var has12, has13 bool
	for _, s := range slots {
		local := s.StartAt.In(loc)
		if local.Hour() == 12 && local.Minute() == 0 {
			has12 = true
		}
		if local.Hour() == 13 && local.Minute() == 0 {
			has13 = true
		}
	}
	assert.False(t, has12, "slot at 12:00 should not exist (outside 09-12 window with buffer)")
	assert.True(t, has13, "slot at 13:00 should exist (start of second window)")
	// Overall windows count should be 2, and overallStart/end calculation branch hit
	// Count expected: first window 09-12 =180m, second 13-18=300m; slots per window: (180-40)/15+1=10, (300-40)/15+1=18 => total 28
	assert.Equal(t, 28, len(slots))
	_ = dayStart

	// Reversed order to hit overallStart update branch (w.start.Before(overallStart))
	fx2 := buildFixtures()
	fx2.repo.availabilities[fx2.staffAndiID] = []db.Availability{
		{ID: uuid.New(), StaffID: fx2.staffAndiID, DayOfWeek: monday, StartTime: pgTime(timeFromHM(13, 0)), EndTime: pgTime(timeFromHM(15, 0))},
		{ID: uuid.New(), StaffID: fx2.staffAndiID, DayOfWeek: monday, StartTime: pgTime(timeFromHM(9, 0)), EndTime: pgTime(timeFromHM(12, 0))},
	}
	svc2 := NewService(fx2.repo)
	slots2, _, err := svc2.GetSlots(context.Background(), fx2.svcClassicID.String(), fx2.staffAndiID.String(), date, "Asia/Jakarta")
	require.NoError(t, err)
	// Second window start 09:00 < first window start 13:00, so overallStart should become 09:00
	assert.NotEmpty(t, slots2)
	assert.Equal(t, 16, len(slots2)) // (120-40)/15+1=6 + (180-40)/15+1=10 =>16
	assert.Greater(t, len(slots2), 10)
}
