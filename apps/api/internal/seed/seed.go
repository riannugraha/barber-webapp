package seed

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// loc is the canonical org timezone — Asia/Jakarta (WIB, no DST, but engine handles DST).
var loc *time.Location

func init() {
	var err error
	loc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
}

// ServiceDef mirrors PRD §3 seed — 8 layanan.
type ServiceDef struct {
	Name            string
	Description     string
	DurationMinutes int32
	BufferMinutes   int32
	PriceCents      int32
	Color           string
}

var serviceDefs = []ServiceDef{
	{Name: "Classic Cut", Description: "Signature classic cut — rapi & cepat", DurationMinutes: 30, BufferMinutes: 10, PriceCents: 85000, Color: "#3b82f6"},
	{Name: "Premium Fade", Description: "Fade presisi dengan detailing", DurationMinutes: 45, BufferMinutes: 10, PriceCents: 120000, Color: "#7c3aed"},
	{Name: "Cut + Beard", Description: "Potong + grooming jenggot", DurationMinutes: 60, BufferMinutes: 15, PriceCents: 150000, Color: "#4f46e5"},
	{Name: "Beard Trim", Description: "Rapikan jenggot & kumis", DurationMinutes: 20, BufferMinutes: 10, PriceCents: 50000, Color: "#14b8a6"},
	{Name: "Hair Color", Description: "Coloring dengan konsultasi", DurationMinutes: 90, BufferMinutes: 15, PriceCents: 250000, Color: "#f59e0b"},
	{Name: "Father & Son", Description: "Paket ayah & anak", DurationMinutes: 60, BufferMinutes: 15, PriceCents: 180000, Color: "#10b981"},
	{Name: "Grooming Package", Description: "Paket lengkap grooming", DurationMinutes: 75, BufferMinutes: 15, PriceCents: 200000, Color: "#f43f5e"},
	{Name: "Konsultasi Style 15m", Description: "Konsultasi style gratis", DurationMinutes: 15, BufferMinutes: 5, PriceCents: 0, Color: "#64748b"},
}

// StaffDef — 3 staff, Sari join 20 Nov 2025.
type StaffDef struct {
	Name  string
	Email string
}

var staffDefs = []StaffDef{
	{Name: "Andi", Email: "andi@flowbook.test"},
	{Name: "Bayu", Email: "bayu@flowbook.test"},
	{Name: "Sari", Email: "sari@flowbook.test"},
}

// staffEligible maps service name -> staff names that can perform it (PRD §3 skill filter).
var staffEligible = map[string][]string{
	"Classic Cut":          {"Andi", "Bayu", "Sari"},
	"Premium Fade":         {"Andi", "Bayu", "Sari"},
	"Cut + Beard":          {"Andi"},
	"Beard Trim":           {"Andi", "Bayu", "Sari"},
	"Hair Color":           {"Bayu"},
	"Father & Son":         {"Andi"},
	"Grooming Package":     {"Andi", "Bayu"},
	"Konsultasi Style 15m": {"Bayu", "Sari"},
}

// customerNames — 60 Indonesian customers, deterministic.
var customerNames = []string{
	"Siti Rahayu", "Budi Santoso", "Ani Wijaya", "Rudi Hartono", "Dewi Lestari",
	"Agus Prasetyo", "Rina Marlina", "Eko Saputra", "Linda Kusuma", "Hendra Gunawan",
	"Maya Sari", "Joko Widodo", "Sri Mulyani", "Fajar Nugroho", "Nisa Amalia",
	"Doni Firmansyah", "Tari Wulandari", "Yudi Pratama", "Citra Dewi", "Arif Hidayat",
	"Lia Amelia", "Iwan Setiawan", "Novi Handayani", "Ahmad Fauzi", "Putri Ayu",
	"Slamet Riyadi", "Wulan Dari", "Bambang Pamungkas", "Yuni Astuti", "Dian Permata",
	"Rizky Ramadhan", "Fitri Handika", "Adit Nugraha", "Sinta Bella", "Hari Susanto",
	"Lestari Indah", "Gita Savitri", "Bayu Skak", "Andini Putri", "Rangga Azof",
	"Kiki Amalia", "Damar Wulan", "Eka Saputra", "Tono Wijaya", "Mira Lesmana",
	"Oka Antara", "Nia Ramadhani", "Reza Rahadian", "Acha Septriasa", "Vino Bastian",
	"Dian Sastro", "Lukman Sardi", "Cut Mini", "Tora Sudiro", "Winky Wiryawan",
	"Nirina Zubir", "Ringgo Agus", "Atiqah Hasiholan", "Marsha Timothy", "Chicco Jerikho",
	"Pevita Pearce", "Tatjana Saphira",
}

// EnsureOrganization creates or fetches the demo organization.
func EnsureOrganization(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	var id uuid.UUID
	// Try existing
	err := pool.QueryRow(ctx, `SELECT id FROM organizations WHERE slug='flowbarber-studio' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Insert
	id = uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO organizations (id, name, slug, timezone) VALUES ($1,$2,$3,$4) ON CONFLICT (slug) DO NOTHING`,
		id, "FlowBarber Studio", "flowbarber-studio", "Asia/Jakarta",
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert organization: %w", err)
	}
	// Re-select if conflict (another insert)
	if err := pool.QueryRow(ctx, `SELECT id FROM organizations WHERE slug='flowbarber-studio' LIMIT 1`).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("select organization after insert: %w", err)
	}
	return id, nil
}

// EnsureServices creates 8 services for org, returns map name->id.
func EnsureServices(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (map[string]uuid.UUID, error) {
	m := make(map[string]uuid.UUID, len(serviceDefs))
	for _, s := range serviceDefs {
		var id uuid.UUID
		err := pool.QueryRow(ctx,
			`SELECT id FROM services WHERE organization_id=$1 AND name=$2 LIMIT 1`, orgID, s.Name).Scan(&id)
		if err == nil {
			m[s.Name] = id
			continue
		}
		id = uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO services (id, organization_id, name, description, duration_minutes, buffer_minutes, price_cents, color, is_active) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,true) ON CONFLICT DO NOTHING`,
			id, orgID, s.Name, s.Description, s.DurationMinutes, s.BufferMinutes, s.PriceCents, s.Color,
		)
		if err != nil {
			return nil, fmt.Errorf("insert service %s: %w", s.Name, err)
		}
		// re-select id (in case conflict generated different id)
		_ = pool.QueryRow(ctx, `SELECT id FROM services WHERE organization_id=$1 AND name=$2 LIMIT 1`, orgID, s.Name).Scan(&id)
		m[s.Name] = id
	}
	return m, nil
}

// EnsureStaff creates 3 staff for org, returns map name->id.
func EnsureStaff(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (map[string]uuid.UUID, error) {
	m := make(map[string]uuid.UUID, len(staffDefs))
	for _, st := range staffDefs {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `SELECT id FROM staff WHERE organization_id=$1 AND name=$2 LIMIT 1`, orgID, st.Name).Scan(&id)
		if err == nil {
			m[st.Name] = id
			continue
		}
		id = uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO staff (id, organization_id, name, email, is_active) VALUES ($1,$2,$3,$4,true) ON CONFLICT DO NOTHING`,
			id, orgID, st.Name, st.Email,
		)
		if err != nil {
			return nil, fmt.Errorf("insert staff %s: %w", st.Name, err)
		}
		_ = pool.QueryRow(ctx, `SELECT id FROM staff WHERE organization_id=$1 AND name=$2 LIMIT 1`, orgID, st.Name).Scan(&id)
		m[st.Name] = id
	}
	return m, nil
}

// EnsureStaffServices creates skill mappings.
func EnsureStaffServices(ctx context.Context, pool *pgxpool.Pool, staffMap, serviceMap map[string]uuid.UUID) error {
	for svcName, eligible := range staffEligible {
		svcID, ok := serviceMap[svcName]
		if !ok {
			continue
		}
		for _, staffName := range eligible {
			stID, ok := staffMap[staffName]
			if !ok {
				continue
			}
			_, err := pool.Exec(ctx,
				`INSERT INTO staff_services (staff_id, service_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
				stID, svcID,
			)
			if err != nil {
				return fmt.Errorf("insert staff_service %s->%s: %w", staffName, svcName, err)
			}
		}
	}
	return nil
}

// EnsureAvailability creates weekly template 09:00-18:00 Mon-Sat, 10:00-16:00 Sun for each staff.
func EnsureAvailability(ctx context.Context, pool *pgxpool.Pool, staffMap map[string]uuid.UUID) error {
	for _, stID := range staffMap {
		// Check if already has availability
		var cnt int
		_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM availability WHERE staff_id=$1`, stID).Scan(&cnt)
		if cnt > 0 {
			continue
		}
		for dow := 0; dow <= 6; dow++ {
			startStr, endStr := "09:00", "18:00"
			if dow == 0 { // Sunday
				startStr = "10:00"
				endStr = "16:00"
			}
			id := uuid.New()
			_, err := pool.Exec(ctx,
				`INSERT INTO availability (id, staff_id, day_of_week, start_time, end_time) VALUES ($1,$2,$3,$4::time,$5::time) ON CONFLICT DO NOTHING`,
				id, stID, dow, startStr, endStr,
			)
			if err != nil {
				return fmt.Errorf("insert availability dow %d: %w", dow, err)
			}
		}
	}
	return nil
}

// EnsureUsers creates owner + staff users for E2E login (hashed bcrypt).
func EnsureUsers(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, staffMap map[string]uuid.UUID) error {
	users := []struct {
		Email    string
		Name     string
		Role     string
		Password string
		Staff    string // link to staff name if staff
	}{
		{Email: "owner@flowbook.test", Name: "Owner FlowBook", Role: "OWNER", Password: "ownerpass"},
		{Email: "andi@flowbook.test", Name: "Andi", Role: "STAFF", Password: "staffpass", Staff: "Andi"},
		{Email: "bayu@flowbook.test", Name: "Bayu", Role: "STAFF", Password: "staffpass", Staff: "Bayu"},
		{Email: "sari@flowbook.test", Name: "Sari", Role: "STAFF", Password: "staffpass", Staff: "Sari"},
	}
	for _, u := range users {
		var exists uuid.UUID
		err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email=$1 LIMIT 1`, u.Email).Scan(&exists)
		if err == nil {
			// Ensure staff link if needed
			if u.Staff != "" {
				if stID, ok := staffMap[u.Staff]; ok {
					_, _ = pool.Exec(ctx, `UPDATE staff SET user_id=$1 WHERE id=$2 AND user_id IS NULL`, exists, stID)
				}
			}
			continue
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password %s: %w", u.Email, err)
		}
		id := uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO users (id, organization_id, email, password_hash, name, role) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (email) DO NOTHING`,
			id, orgID, u.Email, string(hash), u.Name, u.Role,
		)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", u.Email, err)
		}
		// Re-fetch id if conflict
		_ = pool.QueryRow(ctx, `SELECT id FROM users WHERE email=$1 LIMIT 1`, u.Email).Scan(&id)
		if u.Staff != "" {
			if stID, ok := staffMap[u.Staff]; ok {
				_, _ = pool.Exec(ctx, `UPDATE staff SET user_id=$1 WHERE id=$2`, id, stID)
			}
		}
	}
	return nil
}

// EnsureCustomers creates n customers for org, returns map index->id and slice of structs.
type Customer struct {
	ID    uuid.UUID
	Name  string
	Email string
	Phone string
}

func EnsureCustomers(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, n int) ([]Customer, error) {
	// If already have >= n customers, reuse
	var existing int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers WHERE organization_id=$1`, orgID).Scan(&existing)
	if existing >= n {
		rows, err := pool.Query(ctx, `SELECT id, name, email, COALESCE(phone,'') FROM customers WHERE organization_id=$1 ORDER BY created_at LIMIT $2`, orgID, n)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Customer
		for rows.Next() {
			var c Customer
			var phone string
			if err := rows.Scan(&c.ID, &c.Name, &c.Email, &phone); err != nil {
				return nil, err
			}
			c.Phone = phone
			out = append(out, c)
		}
		return out, rows.Err()
	}
	// Insert missing deterministically from customerNames
	out := make([]Customer, 0, n)
	for i := 0; i < n; i++ {
		name := customerNames[i%len(customerNames)]
		// Make unique email by suffix if needed
		emailBase := fmt.Sprintf("%s%d@customer.test", sanitizeEmail(name), i)
		phone := fmt.Sprintf("0812%08d", 10000000+i)
		var id uuid.UUID
		err := pool.QueryRow(ctx, `SELECT id FROM customers WHERE organization_id=$1 AND email=$2 LIMIT 1`, orgID, emailBase).Scan(&id)
		if err == nil {
			out = append(out, Customer{ID: id, Name: name, Email: emailBase, Phone: phone})
			continue
		}
		id = uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO customers (id, organization_id, email, name, phone) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (organization_id, email) DO NOTHING`,
			id, orgID, emailBase, name, phone,
		)
		if err != nil {
			return nil, fmt.Errorf("insert customer %s: %w", emailBase, err)
		}
		_ = pool.QueryRow(ctx, `SELECT id FROM customers WHERE organization_id=$1 AND email=$2 LIMIT 1`, orgID, emailBase).Scan(&id)
		out = append(out, Customer{ID: id, Name: name, Email: emailBase, Phone: phone})
	}
	return out, nil
}

func sanitizeEmail(s string) string {
	// lower, replace space with dot, remove non-alnum
	out := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			out += string(r - 'A' + 'a')
		} else if r >= 'a' && r <= 'z' {
			out += string(r)
		} else if r == ' ' {
			out += "."
		} else if r >= '0' && r <= '9' {
			out += string(r)
		}
	}
	if out == "" {
		out = "customer"
	}
	return out
}

// bookings helpers

// serviceWeight for random pick — Classic Cut 35%.
var serviceWeight = []struct {
	Name   string
	Weight float64
}{
	{"Classic Cut", 0.35},
	{"Premium Fade", 0.15},
	{"Beard Trim", 0.12},
	{"Grooming Package", 0.10},
	{"Konsultasi Style 15m", 0.08},
	{"Cut + Beard", 0.08},
	{"Father & Son", 0.07},
	{"Hair Color", 0.05},
}

func pickService(r *rand.Rand) string {
	f := r.Float64()
	cum := 0.0
	for _, sw := range serviceWeight {
		cum += sw.Weight
		if f < cum {
			return sw.Name
		}
	}
	return "Classic Cut"
}

// pickStaffForService chooses eligible staff weighted by load.
// After Sari join date (2025-11-20) weights: Andi 0.45/Bayu 0.35/Sari 0.20; before: Andi 0.55/Bayu 0.45.
func pickStaffForService(r *rand.Rand, svcName string, date time.Time) string {
	eligible, ok := staffEligible[svcName]
	if !ok || len(eligible) == 0 {
		return "Andi"
	}
	if len(eligible) == 1 {
		return eligible[0]
	}
	// Build weight list for eligible
	type w struct {
		name string
		v    float64
	}
	var list []w
	sariJoin := time.Date(2025, 11, 20, 0, 0, 0, 0, loc)
	afterSari := !date.Before(sariJoin)
	for _, n := range eligible {
		var weight float64
		switch n {
		case "Andi":
			if afterSari {
				weight = 0.45
			} else {
				weight = 0.55
			}
		case "Bayu":
			if afterSari {
				weight = 0.35
			} else {
				weight = 0.45
			}
		case "Sari":
			if afterSari {
				weight = 0.20
			} else {
				weight = 0 // not eligible before join — skip
			}
		default:
			weight = 0.1
		}
		if weight == 0 {
			continue
		}
		// Only include if eligible for this service (already filtered)
		list = append(list, w{name: n, v: weight})
	}
	if len(list) == 0 {
		// fallback: pick first eligible (handles Sari before join but eligible includes Sari — but we filtered)
		for _, n := range eligible {
			if n != "Sari" {
				return n
			}
		}
		return eligible[0]
	}
	// Normalize and pick
	sum := 0.0
	for _, x := range list {
		sum += x.v
	}
	f := r.Float64() * sum
	cum := 0.0
	for _, x := range list {
		cum += x.v
		if f < cum {
			return x.name
		}
	}
	return list[0].name
}

// SeedMinimal creates ~10 bookings quickly for E2E beforeEach — 150ms target.
// It assumes organization/services/staff already exist or will create them.
// It does NOT truncate — caller should TRUNCATE bookings,payments,customers before calling for isolation (Opsi A).
// For standalone `go run ./cmd/seed --minimal` it ensure all base data first.
func SeedMinimal(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	orgID, err := EnsureOrganization(ctx, pool)
	if err != nil {
		return 0, err
	}
	serviceMap, err := EnsureServices(ctx, pool, orgID)
	if err != nil {
		return 0, err
	}
	staffMap, err := EnsureStaff(ctx, pool, orgID)
	if err != nil {
		return 0, err
	}
	if err := EnsureStaffServices(ctx, pool, staffMap, serviceMap); err != nil {
		return 0, err
	}
	if err := EnsureAvailability(ctx, pool, staffMap); err != nil {
		return 0, err
	}
	if err := EnsureUsers(ctx, pool, orgID, staffMap); err != nil {
		// non-fatal for seed
		slog.Warn("ensure users failed (minimal)", "error", err)
	}
	customers, err := EnsureCustomers(ctx, pool, orgID, 10)
	if err != nil {
		return 0, err
	}
	// Build reverse maps for quick lookup
	serviceByName := make(map[string]ServiceDef)
	for _, s := range serviceDefs {
		serviceByName[s.Name] = s
	}
	// For minimal, create 10 bookings on a near date (use 2026-08-20) spaced to avoid EXCLUDE.
	r := rand.New(rand.NewSource(42))
	baseDate := time.Date(2026, 8, 20, 0, 0, 0, 0, loc)
	// Per staff next availability tracking to avoid double-book
	nextAvailable := map[string]time.Time{
		"Andi": time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 9, 0, 0, 0, loc),
		"Bayu": time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 9, 0, 0, 0, loc),
		"Sari": time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 9, 0, 0, 0, loc),
	}
	inserted := 0
	for i := 0; i < 10; i++ {
		svcName := pickService(r)
		svcDef := serviceByName[svcName]
		staffName := pickStaffForService(r, svcName, baseDate)
		// Sari not available before join but baseDate is after, so ok
		cust := customers[r.Intn(len(customers))]
		stID := staffMap[staffName]
		svcID := serviceMap[svcName]
		occupied := time.Duration(svcDef.DurationMinutes+svcDef.BufferMinutes) * time.Minute
		startLocal := nextAvailable[staffName]
		// Add small random jitter 0-10 min but ensure within 09:00-17:00 window
		jitter := time.Duration(r.Intn(11)) * time.Minute
		startLocal = startLocal.Add(jitter)
		endLocal := startLocal.Add(occupied)
		// If beyond 18:00, wrap to next day 09:00 (still same baseDate for minimal — skip if exceeds)
		dayEnd := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 18, 0, 0, 0, loc)
		if endLocal.After(dayEnd) {
			// move to next day
			baseDate = baseDate.AddDate(0, 0, 1)
			for k := range nextAvailable {
				nextAvailable[k] = time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 9, 0, 0, 0, loc)
			}
			startLocal = nextAvailable[staffName].Add(jitter)
			endLocal = startLocal.Add(occupied)
		}
		nextAvailable[staffName] = endLocal.Add(time.Duration(r.Intn(5)) * time.Minute)

		startUTC := startLocal.UTC()
		endUTC := endLocal.UTC()
		// Status: minimal mostly CONFIRMED
		status := "CONFIRMED"
		if r.Float64() < 0.08 {
			status = "CANCELLED"
		}
		paymentStatus := "PAID"
		if status == "CANCELLED" {
			paymentStatus = "UNPAID"
		}
		if svcDef.PriceCents == 0 {
			paymentStatus = "PAID"
		}
		id := uuid.New()
		_, err = pool.Exec(ctx,
			`INSERT INTO bookings (id, organization_id, service_id, staff_id, customer_id, customer_name, customer_email, customer_phone, notes, start_at, end_at, status, payment_status)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			id, orgID, svcID, stID, customers[i%len(customers)].ID, cust.Name, cust.Email, cust.Phone, fmt.Sprintf("Seed minimal %d", i+1), startUTC, endUTC, status, paymentStatus,
		)
		if err != nil {
			// If EXCLUDE violation (23P01), skip and retry with next slot
			if isExclusionError(err) {
				slog.Warn("seed minimal exclusion skip", "staff", staffName, "start", startUTC)
				continue
			}
			return inserted, fmt.Errorf("insert minimal booking %d: %w", i, err)
		}
		inserted++
	}
	slog.Info("seed minimal done", "bookings", inserted, "org", orgID)
	return inserted, nil
}

// SeedFull creates ~1.500 bookings loop 2025-11-01 → 2026-08-24 with weekend/seasonal weighting.
func SeedFull(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	orgID, err := EnsureOrganization(ctx, pool)
	if err != nil {
		return 0, err
	}
	serviceMap, err := EnsureServices(ctx, pool, orgID)
	if err != nil {
		return 0, err
	}
	staffMap, err := EnsureStaff(ctx, pool, orgID)
	if err != nil {
		return 0, err
	}
	if err := EnsureStaffServices(ctx, pool, staffMap, serviceMap); err != nil {
		return 0, err
	}
	if err := EnsureAvailability(ctx, pool, staffMap); err != nil {
		return 0, err
	}
	if err := EnsureUsers(ctx, pool, orgID, staffMap); err != nil {
		slog.Warn("ensure users failed (full)", "error", err)
	}
	customers, err := EnsureCustomers(ctx, pool, orgID, 60)
	if err != nil {
		return 0, err
	}
	serviceByName := make(map[string]ServiceDef)
	for _, s := range serviceDefs {
		serviceByName[s.Name] = s
	}
	// For idempotency: delete existing bookings/payments for this org before reseeding
	// Keep customers/services/staff — only bookings/payments are regenerated
	if _, err := pool.Exec(ctx, `DELETE FROM payments WHERE organization_id=$1`, orgID); err != nil {
		slog.Warn("delete payments before full seed", "error", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM bookings WHERE organization_id=$1`, orgID); err != nil {
		return 0, fmt.Errorf("delete bookings before seed: %w", err)
	}
	r := rand.New(rand.NewSource(42))
	startDate := time.Date(2025, 11, 1, 0, 0, 0, 0, loc)
	endDate := time.Date(2026, 8, 24, 0, 0, 0, 0, loc)

	// Per day tracking for per-staff intervals to avoid overlap
	totalInserted := 0
	sariJoin := time.Date(2025, 11, 20, 0, 0, 0, 0, loc)

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		// Weight: base weekday 4, weekend 8 + seasonal multiplier
		base := 4
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			base = 8
		}
		mult := 1.0
		switch d.Month() {
		case time.December:
			mult = 1.6
		case time.January:
			mult = 1.2
		case time.February, time.March:
			mult = 1.0
		case time.April, time.May:
			mult = 0.9
		case time.June, time.July:
			mult = 1.15
		case time.August:
			mult = 0.85
		case time.November:
			if d.Day() < 20 {
				mult = 0.9
			} else {
				mult = 1.0
			}
		}
		countFloat := float64(base) * mult
		jitter := r.Intn(3) - 1 // -1..1
		count := int(countFloat) + jitter
		if count < 1 {
			count = 1
		}
		if count > 10 {
			count = 10
		}
		// For determinism, we also cap via weekday variation: if Dec weekend, ensure at least 6
		// Maintain intervals per staff for this day
		// Map staffName -> []interval
		type interval struct{ start, end time.Time }
		occupiedIntervals := map[string][]interval{
			"Andi": {},
			"Bayu": {},
			"Sari": {},
		}
		// Try to generate count bookings
		attempts := 0
		insertedForDay := 0
		for insertedForDay < count && attempts < count*5 {
			attempts++
			svcName := pickService(r)
			svcDef := serviceByName[svcName]
			staffName := pickStaffForService(r, svcName, d)
			// Sari not available before join
			if staffName == "Sari" && d.Before(sariJoin) {
				continue
			}
			stID, ok := staffMap[staffName]
			if !ok {
				continue
			}
			svcID, ok := serviceMap[svcName]
			if !ok {
				continue
			}
			cust := customers[r.Intn(len(customers))]
			occupied := time.Duration(svcDef.DurationMinutes+svcDef.BufferMinutes) * time.Minute
			// Random start time within 09:00-18:00 window that fits occupied
			dayStart := time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, loc)
			dayEnd := time.Date(d.Year(), d.Month(), d.Day(), 18, 0, 0, 0, loc)
			if d.Weekday() == time.Sunday {
				dayStart = time.Date(d.Year(), d.Month(), d.Day(), 10, 0, 0, 0, loc)
				dayEnd = time.Date(d.Year(), d.Month(), d.Day(), 16, 0, 0, 0, loc)
			}
			windowMinutes := int(dayEnd.Sub(dayStart).Minutes()) - int(occupied.Minutes())
			if windowMinutes < 0 {
				continue
			}
			offset := r.Intn(windowMinutes + 1)
			startLocal := dayStart.Add(time.Duration(offset) * time.Minute)
			// Align to 5 min grid for realism (optional, but keep 15m grid engine compatibility: engine uses 15m grid)
			// Snap to 5m
			min := startLocal.Minute()
			snapped := (min / 5) * 5
			startLocal = time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), startLocal.Hour(), snapped, 0, 0, loc)
			endLocal := startLocal.Add(occupied)
			if endLocal.After(dayEnd) {
				continue
			}
			// Check overlap with existing intervals for this staff this day
			overlap := false
			for _, iv := range occupiedIntervals[staffName] {
				if startLocal.Before(iv.end) && iv.start.Before(endLocal) {
					overlap = true
					break
				}
			}
			if overlap {
				continue
			}
			// Prepare UTC and status
			startUTC := startLocal.UTC()
			endUTC := endLocal.UTC()
			status := "CONFIRMED"
			// 7.2% cancelled, 8% pending for future sensitivity
			p := r.Float64()
			if p < 0.072 {
				status = "CANCELLED"
			} else if d.After(time.Date(2026, 8, 20, 0, 0, 0, 0, loc)) && r.Float64() < 0.15 {
				status = "PENDING"
			} else if r.Float64() < 0.02 {
				status = "PENDING"
			}
			paymentStatus := "PAID"
			if status == "CANCELLED" {
				paymentStatus = "UNPAID"
			}
			if status == "PENDING" {
				paymentStatus = "UNPAID"
			}
			if svcDef.PriceCents == 0 {
				// free always confirmed unless cancelled
				if status != "CANCELLED" {
					status = "CONFIRMED"
					paymentStatus = "PAID"
				}
			}
			id := uuid.New()
			// Choose payment_status logic for dashboard: CONFIRMED/COMPLETED counted
			_, err = pool.Exec(ctx,
				`INSERT INTO bookings (id, organization_id, service_id, staff_id, customer_id, customer_name, customer_email, customer_phone, notes, start_at, end_at, status, payment_status)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
				id, orgID, svcID, stID, cust.ID, cust.Name, cust.Email, cust.Phone, fmt.Sprintf("Seed %s %s", svcName, d.Format("2006-01-02")), startUTC, endUTC, status, paymentStatus,
			)
			if err != nil {
				if isExclusionError(err) {
					// Rare due to client check — skip
					continue
				}
				return totalInserted, fmt.Errorf("insert booking %s %s: %w", d.Format("2006-01-02"), svcName, err)
			}
			occupiedIntervals[staffName] = append(occupiedIntervals[staffName], interval{start: startLocal, end: endLocal})
			insertedForDay++
			totalInserted++
		}
		// Optional progress log every 30 days
		if totalInserted%300 == 0 && insertedForDay > 0 {
			slog.Info("seed progress", "date", d.Format("2006-01-02"), "total", totalInserted)
		}
	}
	slog.Info("seed full done", "total_bookings", totalInserted, "from", startDate.Format("2006-01-02"), "to", endDate.Format("2006-01-02"))
	return totalInserted, nil
}

func isExclusionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// pgx returns "ERROR: conflicting key value violates exclusion constraint \"no_overlap\" (SQLSTATE 23P01)"
	return contains(msg, "23P01") || contains(msg, "exclusion") || contains(msg, "no_overlap")
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// SeedMinimalTx is a helper for testhelpers — truncates and seeds minimal in one transaction context.
func SeedMinimalTx(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	// Caller should have already truncated; we just seed minimal data
	return SeedMinimal(ctx, pool)
}
