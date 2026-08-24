package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers postgres with pgxpool.
type PostgresContainer struct {
	Container testcontainers.Container
	Pool      *pgxpool.Pool
	ConnStr   string // without sslmode? with sslmode disable
}

// CreatePostgresContainer spins up postgres:16-alpine via testcontainers-go 0.33+.
// It uses 1 container per TestMain reuse pattern — caller should Terminate after m.Run().
// Image is postgres:16-alpine as required by T13.
// Connection uses sslmode=disable, DB flowbook_test, user test / pass test.
func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	// Allow long timeout for podman cold start
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("flowbook_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.Run: %w", err)
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("connection string: %w", err)
	}

	// pgxpool with small pool for tests
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	// Ensure pooler-like behavior not needed for local, but keep
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		_ = ctr.Terminate(ctx)
		return nil, fmt.Errorf("ping: %w", err)
	}

	// Run migrations via golang-migrate 4.18+
	if err := runMigrations(connStr); err != nil {
		// Fallback to direct SQL exec if migrate fails (e.g., file path not found)
		if fbErr := runMigrationsDirect(ctx, pool); fbErr != nil {
			pool.Close()
			_ = ctr.Terminate(ctx)
			return nil, fmt.Errorf("migrate up failed: %v; fallback also failed: %v", err, fbErr)
		}
	}

	return &PostgresContainer{
		Container: ctr,
		Pool:      pool,
		ConnStr:   connStr,
	}, nil
}

// Terminate closes pool and terminates container.
func (p *PostgresContainer) Terminate(ctx context.Context) error {
	if p.Pool != nil {
		p.Pool.Close()
	}
	if p.Container != nil {
		// Use background with timeout to ensure termination even if ctx cancelled
		tCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return p.Container.Terminate(tCtx)
	}
	return nil
}

// runMigrations uses golang-migrate file source to apply up.
// It resolves migrations directory via runtime caller and file discovery.
func runMigrations(connStr string) error {
	migrationsPath, err := findMigrationsDir()
	if err != nil {
		return err
	}
	// golang-migrate expects postgres URL without pgbouncer, sslmode disable already
	// Ensure scheme postgres:// is present (testcontainers returns postgres://...)
	sourceURL := "file://" + migrationsPath
	m, err := migrate.New(sourceURL, connStr)
	if err != nil {
		return fmt.Errorf("migrate.New source=%s db=%s: %w", sourceURL, maskConnStr(connStr), err)
	}
	defer m.Close()
	// Up
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate.Up: %w", err)
	}
	return nil
}

func maskConnStr(s string) string {
	// hide password
	if idx := strings.Index(s, "@"); idx != -1 {
		if at := strings.LastIndex(s[:idx], ":"); at != -1 {
			return s[:at+1] + "****" + s[idx:]
		}
	}
	return s
}

// findMigrationsDir locates apps/api/migrations relative to this file or cwd.
func findMigrationsDir() (string, error) {
	// Try from runtime caller file location: internal/testhelpers/postgres.go -> ../../migrations
	_, file, _, ok := runtime.Caller(0)
	if ok {
		candidate := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			abs, _ := filepath.Abs(candidate)
			return abs, nil
		}
	}
	// Try cwd walking upwards
	cwd, err := os.Getwd()
	if err == nil {
		dir := cwd
		for i := 0; i < 6; i++ {
			candidate := filepath.Join(dir, "migrations")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				abs, _ := filepath.Abs(candidate)
				return abs, nil
			}
			candidate = filepath.Join(dir, "apps", "api", "migrations")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				abs, _ := filepath.Abs(candidate)
				return abs, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "", fmt.Errorf("migrations directory not found from %s (caller=%s)", cwd, file)
}

// runMigrationsDirect is fallback that reads 0001_init.up.sql and executes via pgxpool.
// Ensures test runs even if golang-migrate file resolution fails.
func runMigrationsDirect(ctx context.Context, pool *pgxpool.Pool) error {
	migrationsPath, err := findMigrationsDir()
	if err != nil {
		return err
	}
	upPath := filepath.Join(migrationsPath, "0001_init.up.sql")
	data, err := os.ReadFile(upPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", upPath, err)
	}
	// Exec whole file — pgx allows multiple statements via simple query
	_, err = pool.Exec(ctx, string(data))
	if err != nil {
		return fmt.Errorf("exec migration SQL: %w", err)
	}
	return nil
}

// MustCreatePostgresContainer is helper for tests that want to fail fast.
func MustCreatePostgresContainer(ctx context.Context) *PostgresContainer {
	pc, err := CreatePostgresContainer(ctx)
	if err != nil {
		panic(fmt.Sprintf("CreatePostgresContainer failed: %v", err))
	}
	return pc
}

// TruncateAll truncates bookings, payments, customers for isolation (Opsi A helper for integration tests).
func TruncateAll(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`)
	return err
}
