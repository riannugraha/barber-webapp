package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"flowbook/api/internal/db"
	appmw "flowbook/api/internal/middleware"

	"flowbook/api/internal/config"
)

// Errors for handler mapping.
var (
	ErrDuplicateEmail      = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrUserNotFound        = errors.New("user not found")
)

// Service handles auth business logic: bcrypt, JWT 15m, refresh 7d via token_hash.
type Service struct {
	repo          Repository
	jwtSecret     string
	refreshSecret string // not used for signing, kept for future HMAC if needed
}

// NewService creates Service. jwtSecret must be at least 32 chars (openssl rand -hex 32).
func NewService(repo Repository, cfg config.Config) *Service {
	secret := cfg.JWTSecret
	refresh := cfg.RefreshSecret
	if refresh == "" {
		refresh = secret
	}
	return &Service{repo: repo, jwtSecret: secret, refreshSecret: refresh}
}

// NewServiceWithSecrets direct constructor (for tests).
func NewServiceWithSecrets(repo Repository, jwtSecret, refreshSecret string) *Service {
	if refreshSecret == "" {
		refreshSecret = jwtSecret
	}
	return &Service{repo: repo, jwtSecret: jwtSecret, refreshSecret: refreshSecret}
}

// RegisterRequest mirrors openapi RegisterRequest.
type RegisterRequest struct {
	Email          string  `json:"email" validate:"required,email"`
	Password       string  `json:"password" validate:"required,min=8"`
	Name           string  `json:"name" validate:"required,min=2"`
	Role           *string `json:"role" validate:"omitempty,oneof=OWNER STAFF CUSTOMER"`
	OrganizationID *string `json:"organizationId" validate:"omitempty,uuid"`
}

// LoginRequest mirrors openapi LoginRequest.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// UserResponse is the public user shape returned in AuthResponse.
type UserResponse struct {
	ID             string  `json:"id"`
	OrganizationID *string `json:"organizationId,omitempty"`
	Email          string  `json:"email"`
	Name           string  `json:"name"`
	Role           string  `json:"role"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AuthResponse mirrors openapi AuthResponse.
type AuthResponse struct {
	AccessToken string       `json:"accessToken"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	User        UserResponse `json:"user"`
}

// TokenPair holds generated tokens for handler to set cookies.
type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshTokenRaw  string
	RefreshExpiresAt time.Time
}

func toUserResponse(u db.User) UserResponse {
	var orgID *string
	if u.OrganizationID.Valid {
		uid, err := uuidFromBytes(u.OrganizationID.Bytes)
		if err == nil {
			s := uid.String()
			orgID = &s
		} else {
			// pgtype.UUID bytes directly
			// fallback: try parse via uuid.FromBytes? pgtype.UUID is [16]byte
			// Use hex formatting via uuid.UUID(u.OrganizationID.Bytes)
			// pgtype.UUID.Bytes is [16]byte
			uu := uuid.UUID(u.OrganizationID.Bytes)
			s2 := uu.String()
			if s2 != "00000000-0000-0000-0000-000000000000" {
				orgID = &s2
			}
		}
	}
	return UserResponse{
		ID:             u.ID.String(),
		OrganizationID: orgID,
		Email:          u.Email,
		Name:           u.Name,
		Role:           u.Role,
		CreatedAt:      u.CreatedAt,
	}
}

func uuidFromBytes(b [16]byte) (uuid.UUID, error) {
	return uuid.FromBytes(b[:])
}

// hashRefresh hashes raw refresh token with SHA256 hex (stored as token_hash).
// We use SHA256 rather than bcrypt for fast lookup by hash.
func hashRefresh(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// generateRefreshRaw creates a cryptographically random 32-byte hex token (64 chars).
func generateRefreshRaw() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generateAccessToken creates JWT access token 15m.
func (s *Service) generateAccessToken(u db.User) (string, time.Time, error) {
	if s.jwtSecret == "" {
		return "", time.Time{}, errors.New("JWT_SECRET not configured")
	}
	now := time.Now()
	exp := now.Add(15 * time.Minute)
	var orgID *string
	if u.OrganizationID.Valid {
		uu := uuid.UUID(u.OrganizationID.Bytes)
		if uu != uuid.Nil {
			str := uu.String()
			orgID = &str
		}
	}
	claims := appmw.Claims{
		UserID: u.ID.String(),
		Email:  u.Email,
		Role:   u.Role,
		OrgID:  orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
			Issuer:    "flowbook-api",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// Register creates a new user with bcrypt, then issues token pair.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, *TokenPair, error) {
	// Normalize email
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return nil, nil, errors.New("email required")
	}
	// Default role CUSTOMER if not provided
	role := "CUSTOMER"
	if req.Role != nil && *req.Role != "" {
		role = strings.ToUpper(strings.TrimSpace(*req.Role))
	}
	if role != "OWNER" && role != "STAFF" && role != "CUSTOMER" {
		role = "CUSTOMER"
	}

	// Check duplicate via repo (let DB unique handle too)
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, nil, ErrDuplicateEmail
	}

	// Hash password with bcrypt (cost 10 default)
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	var orgID pgtype.UUID
	if req.OrganizationID != nil && *req.OrganizationID != "" {
		uid, err := uuid.Parse(*req.OrganizationID)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid organizationId: %w", err)
		}
		orgID = pgtype.UUID{Bytes: [16]byte(uid), Valid: true}
	} else {
		orgID = pgtype.UUID{Valid: false}
	}

	user, err := s.repo.CreateUser(ctx, db.CreateUserParams{
		OrganizationID: orgID,
		Email:          email,
		PasswordHash:   string(hash),
		Name:           strings.TrimSpace(req.Name),
		Role:           role,
	})
	if err != nil {
		if isDuplicateError(err) {
			return nil, nil, ErrDuplicateEmail
		}
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	access, exp, err := s.generateAccessToken(user)
	if err != nil {
		return nil, nil, err
	}
	rawRefresh, err := generateRefreshRaw()
	if err != nil {
		return nil, nil, err
	}
	hashRefreshStr := hashRefresh(rawRefresh)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := s.repo.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashRefreshStr,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, nil, fmt.Errorf("store refresh token: %w", err)
	}

	resp := &AuthResponse{
		AccessToken: access,
		ExpiresAt:   exp,
		User:        toUserResponse(user),
	}
	pair := &TokenPair{
		AccessToken:      access,
		AccessExpiresAt:  exp,
		RefreshTokenRaw:  rawRefresh,
		RefreshExpiresAt: expiresAt,
	}
	return resp, pair, nil
}

// Login verifies bcrypt and issues token pair.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, *TokenPair, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	access, exp, err := s.generateAccessToken(user)
	if err != nil {
		return nil, nil, err
	}
	rawRefresh, err := generateRefreshRaw()
	if err != nil {
		return nil, nil, err
	}
	hashStr := hashRefresh(rawRefresh)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := s.repo.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hashStr,
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, nil, fmt.Errorf("store refresh token: %w", err)
	}

	resp := &AuthResponse{
		AccessToken: access,
		ExpiresAt:   exp,
		User:        toUserResponse(user),
	}
	pair := &TokenPair{
		AccessToken:      access,
		AccessExpiresAt:  exp,
		RefreshTokenRaw:  rawRefresh,
		RefreshExpiresAt: expiresAt,
	}
	return resp, pair, nil
}

// Refresh validates refresh cookie and issues new access token (no rotation for simplicity, but validates not revoked/expired).
func (s *Service) Refresh(ctx context.Context, rawRefresh string) (string, time.Time, error) {
	if strings.TrimSpace(rawRefresh) == "" {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	h := hashRefresh(rawRefresh)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, h)
	if err != nil {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	if rt.Revoked {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	if time.Now().After(rt.ExpiresAt) {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	access, exp, err := s.generateAccessToken(user)
	if err != nil {
		return "", time.Time{}, err
	}
	return access, exp, nil
}

// Logout revokes the refresh token hash.
func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	if strings.TrimSpace(rawRefresh) == "" {
		return ErrInvalidRefreshToken
	}
	h := hashRefresh(rawRefresh)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, h)
	if err != nil {
		// idempotent: if not found, treat as already revoked (for httptest)
		return nil
	}
	if rt.Revoked {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, h)
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") || strings.Contains(msg, "23505")
}
