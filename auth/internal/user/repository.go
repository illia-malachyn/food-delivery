package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("user not found")
	ErrEmailUsed = errors.New("email already used")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
}

type Repository interface {
	Create(ctx context.Context, email, passwordHash string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, userID int64) (User, error)
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) Create(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, password_hash`,
		email,
		passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailUsed
		}
		return User{}, err
	}

	return u, nil
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `SELECT id, email, password_hash FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return u, nil
}

func (r *postgresRepository) FindByID(ctx context.Context, userID int64) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `SELECT id, email, password_hash FROM users WHERE id = $1`, userID).
		Scan(&u.ID, &u.Email, &u.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
