package auth

import (
	"context"
	"fmt"

	"flowbook/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository abstracts DB access for auth — allows httptest without real pool.
type Repository interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}

// PGRepository is the production implementation backed by pgxpool + sqlc.
type PGRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// NewPGRepository creates a PGRepository. pool may be nil only for tests that mock Repository.
func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	var q *db.Queries
	if pool != nil {
		q = db.New(pool)
	}
	return &PGRepository{pool: pool, q: q}
}

// NewRepositoryWithQueries is for testing with custom DBTX (e.g., pgxmock or testcontainers Tx).
func NewRepositoryWithQueries(q *db.Queries) *PGRepository {
	return &PGRepository{q: q}
}

func (r *PGRepository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if r.q == nil {
		return db.User{}, fmt.Errorf("db not initialized")
	}
	return r.q.CreateUser(ctx, arg)
}

func (r *PGRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if r.q == nil {
		return db.User{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetUserByEmail(ctx, email)
}

func (r *PGRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	if r.q == nil {
		return db.User{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetUserByID(ctx, id)
}

func (r *PGRepository) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	if r.q == nil {
		return db.RefreshToken{}, fmt.Errorf("db not initialized")
	}
	return r.q.CreateRefreshToken(ctx, arg)
}

func (r *PGRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	if r.q == nil {
		return db.RefreshToken{}, fmt.Errorf("db not initialized")
	}
	return r.q.GetRefreshTokenByHash(ctx, tokenHash)
}

func (r *PGRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if r.q == nil {
		return fmt.Errorf("db not initialized")
	}
	return r.q.RevokeRefreshToken(ctx, tokenHash)
}

// InMemoryRepository is a test double with no external DB — used for httptest unit tests.
// It stores users and refresh tokens in maps.
type InMemoryRepository struct {
	usersByEmail  map[string]db.User
	usersByID     map[uuid.UUID]db.User
	refreshByHash map[string]db.RefreshToken
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		usersByEmail:  make(map[string]db.User),
		usersByID:     make(map[uuid.UUID]db.User),
		refreshByHash: make(map[string]db.RefreshToken),
	}
}

func (m *InMemoryRepository) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if _, exists := m.usersByEmail[arg.Email]; exists {
		return db.User{}, fmt.Errorf("duplicate key value violates unique constraint \"users_email_key\"")
	}
	id := uuid.New()
	// derive org uuid if present
	u := db.User{
		ID:             id,
		OrganizationID: arg.OrganizationID,
		Email:          arg.Email,
		PasswordHash:   arg.PasswordHash,
		Name:           arg.Name,
		Role:           arg.Role,
	}
	m.usersByEmail[arg.Email] = u
	m.usersByID[id] = u
	return u, nil
}

func (m *InMemoryRepository) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return db.User{}, fmt.Errorf("no rows in result set")
	}
	return u, nil
}

func (m *InMemoryRepository) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	u, ok := m.usersByID[id]
	if !ok {
		return db.User{}, fmt.Errorf("no rows in result set")
	}
	return u, nil
}

func (m *InMemoryRepository) CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	rt := db.RefreshToken{
		ID:        uuid.New(),
		UserID:    arg.UserID,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
		Revoked:   false,
	}
	m.refreshByHash[arg.TokenHash] = rt
	return rt, nil
}

func (m *InMemoryRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error) {
	rt, ok := m.refreshByHash[tokenHash]
	if !ok {
		return db.RefreshToken{}, fmt.Errorf("no rows in result set")
	}
	return rt, nil
}

func (m *InMemoryRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	rt, ok := m.refreshByHash[tokenHash]
	if !ok {
		return fmt.Errorf("no rows in result set")
	}
	rt.Revoked = true
	m.refreshByHash[tokenHash] = rt
	return nil
}
