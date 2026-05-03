package pgrepo

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joaquing/clone-supremacy/internal/domain/userdom"
)

// Users is the Postgres-backed implementation of the
// [authsvc.UsersRepository] interface.
type Users struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users { return &Users{pool: pool} }

func (r *Users) Create(ctx context.Context, email, hash, display, color string) (userdom.User, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO users (email, password_hash, display_name, color)
        VALUES ($1, $2, $3, $4)
        RETURNING id, email, password_hash, display_name, color, is_bot, created_at, last_login_at`,
		email, hash, display, color)
	u, err := scanUser(row)
	if err != nil && strings.Contains(err.Error(), "users_email_key") {
		return userdom.User{}, userdom.ErrExists
	}
	return u, err
}

func (r *Users) ByEmail(ctx context.Context, email string) (userdom.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, email, password_hash, display_name, color, is_bot, created_at, last_login_at
        FROM users WHERE email = $1`, email)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userdom.User{}, userdom.ErrNotFound
	}
	return u, err
}

func (r *Users) ByID(ctx context.Context, id uuid.UUID) (userdom.User, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, email, password_hash, display_name, color, is_bot, created_at, last_login_at
        FROM users WHERE id = $1`, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userdom.User{}, userdom.ErrNotFound
	}
	return u, err
}

func (r *Users) TouchLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

// EnsureBot upserts a service-account user with is_bot=true. Returns the
// existing record if one already matches the email, otherwise it creates
// a fresh row. Used by the ai-bot mode at startup to make sure there is
// always a pool of bot identities to lend to unattended slots.
func (r *Users) EnsureBot(ctx context.Context, email, display, color string) (userdom.User, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO users (email, password_hash, display_name, color, is_bot)
        VALUES ($1, '', $2, $3, TRUE)
        ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
        RETURNING id, email, password_hash, display_name, color, is_bot, created_at, last_login_at`,
		email, display, color)
	return scanUser(row)
}

// ListBots returns up to limit bot accounts, ordered by id for stable
// round-robin assignment.
func (r *Users) ListBots(ctx context.Context, limit int) ([]userdom.User, error) {
	if limit <= 0 {
		limit = 16
	}
	rows, err := r.pool.Query(ctx, `
        SELECT id, email, password_hash, display_name, color, is_bot, created_at, last_login_at
        FROM users WHERE is_bot = TRUE
        ORDER BY id
        LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []userdom.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (userdom.User, error) {
	var u userdom.User
	if err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Color,
		&u.IsBot, &u.CreatedAt, &u.LastLoginAt,
	); err != nil {
		return userdom.User{}, err
	}
	return u, nil
}
