package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gewall/dnd-forge/api/internal/db"
	"github.com/gewall/dnd-forge/api/internal/db/sqlc"
	"github.com/gewall/dnd-forge/api/internal/identity"
)

// PostgresStore implements Store on top of the sqlc queries.
type PostgresStore struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

// NewPostgresStore returns a Store backed by pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool, q: sqlc.New(pool)}
}

func (s *PostgresStore) InTx(ctx context.Context, fn func(TokenStore) error) error {
	return db.InTx(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(&PostgresStore{pool: s.pool, q: s.q.WithTx(tx)})
	})
}

func (s *PostgresStore) CreateUser(ctx context.Context, u NewUser) (User, error) {
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        u.Email,
		PasswordHash: nonEmpty(u.PasswordHash),
		GoogleSub:    nonEmpty(u.GoogleSub),
		Name:         u.Name,
		Role:         sqlc.UserRole(u.Role),
	})
	if db.IsUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return toUser(row), nil
}

func (s *PostgresStore) UserByID(ctx context.Context, id uuid.UUID) (User, error) {
	return oneUser(s.q.UserByID(ctx, id))
}

func (s *PostgresStore) UserByEmail(ctx context.Context, email string) (User, error) {
	return oneUser(s.q.UserByEmail(ctx, email))
}

func (s *PostgresStore) UserByGoogleSub(ctx context.Context, subject string) (User, error) {
	return oneUser(s.q.UserByGoogleSub(ctx, &subject))
}

func (s *PostgresStore) LinkGoogle(ctx context.Context, id uuid.UUID, subject string) (User, error) {
	return oneUser(s.q.LinkGoogleSub(ctx, sqlc.LinkGoogleSubParams{ID: id, GoogleSub: &subject}))
}

func (s *PostgresStore) SetRole(ctx context.Context, id uuid.UUID, role identity.Role) (User, error) {
	return oneUser(s.q.SetUserRole(ctx, sqlc.SetUserRoleParams{ID: id, Role: sqlc.UserRole(role)}))
}

func (s *PostgresStore) SetPasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if err := s.q.SetUserPassword(ctx, sqlc.SetUserPasswordParams{ID: id, PasswordHash: &hash}); err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	return nil
}

func (s *PostgresStore) SaveRefreshToken(ctx context.Context, t RefreshToken) error {
	err := s.q.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    t.UserID,
		TokenHash: t.Hash,
		ExpiresAt: t.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (s *PostgresStore) RefreshTokenForUpdate(ctx context.Context, hash string) (RefreshToken, error) {
	row, err := s.q.RefreshTokenByHash(ctx, hash)
	if db.IsNoRows(err) {
		return RefreshToken{}, errRefreshTokenNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("refresh token: %w", err)
	}
	return RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		Hash:      row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt,
	}, nil
}

func (s *PostgresStore) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if err := s.q.RevokeRefreshToken(ctx, id); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (s *PostgresStore) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	if err := s.q.RevokeUserRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke user refresh tokens: %w", err)
	}
	return nil
}

func oneUser(row sqlc.User, err error) (User, error) {
	if db.IsNoRows(err) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("user: %w", err)
	}
	return toUser(row), nil
}

func toUser(r sqlc.User) User {
	u := User{
		ID:        r.ID,
		Email:     r.Email,
		Name:      r.Name,
		Role:      identity.Role(r.Role),
		IsBanned:  r.IsBanned,
		CreatedAt: r.CreatedAt,
	}
	if r.PasswordHash != nil {
		u.PasswordHash = *r.PasswordHash
	}
	if r.GoogleSub != nil {
		u.GoogleSub = *r.GoogleSub
	}
	if r.DailyQuota != nil {
		q := int(*r.DailyQuota)
		u.DailyQuota = &q
	}
	return u
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
