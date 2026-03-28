package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *entity.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, phone, email, full_name, role, is_verified, is_banned, is_flagged,
		                   listing_count_today, listing_count_date, upi_id, created_at, updated_at)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		u.ID, u.Phone, u.Email, u.FullName, u.Role, u.IsVerified, u.IsBanned, u.IsFlagged,
		u.ListingCountToday, u.ListingCountDate, u.UPIID, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return apperror.ErrConflict
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	u := &entity.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(phone,''), COALESCE(email,''), full_name, role, is_verified, is_banned, is_flagged,
		       listing_count_today, listing_count_date, upi_id, created_at, updated_at
		FROM users WHERE id = $1`, id).Scan(
		&u.ID, &u.Phone, &u.Email, &u.FullName, &u.Role, &u.IsVerified, &u.IsBanned, &u.IsFlagged,
		&u.ListingCountToday, &u.ListingCountDate, &u.UPIID, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*entity.User, error) {
	u := &entity.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(phone,''), COALESCE(email,''), full_name, role, is_verified, is_banned, is_flagged,
		       listing_count_today, listing_count_date, upi_id, created_at, updated_at
		FROM users WHERE phone = $1`, phone).Scan(
		&u.ID, &u.Phone, &u.Email, &u.FullName, &u.Role, &u.IsVerified, &u.IsBanned, &u.IsFlagged,
		&u.ListingCountToday, &u.ListingCountDate, &u.UPIID, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by phone: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	u := &entity.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, COALESCE(phone,''), COALESCE(email,''), full_name, role, is_verified, is_banned, is_flagged,
		       listing_count_today, listing_count_date, upi_id, created_at, updated_at
		FROM users WHERE email = $1`, email).Scan(
		&u.ID, &u.Phone, &u.Email, &u.FullName, &u.Role, &u.IsVerified, &u.IsBanned, &u.IsFlagged,
		&u.ListingCountToday, &u.ListingCountDate, &u.UPIID, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *entity.User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE users SET phone=NULLIF($1,''), email=NULLIF($2,''), full_name=$3, role=$4, is_verified=$5,
		is_banned=$6, is_flagged=$7, upi_id=$8, updated_at=$9
		WHERE id=$10`,
		u.Phone, u.Email, u.FullName, u.Role, u.IsVerified, u.IsBanned, u.IsFlagged,
		u.UPIID, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// IncrListingCount increments the daily listing counter, resetting it if the date has changed.
func (r *UserRepository) IncrListingCount(ctx context.Context, id uuid.UUID) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET listing_count_today = CASE
			WHEN listing_count_date < $1 THEN 1
			ELSE listing_count_today + 1
		END,
		listing_count_date = $1,
		updated_at = now()
		WHERE id = $2`, today, id)
	if err != nil {
		return fmt.Errorf("incr listing count: %w", err)
	}
	return nil
}

func (r *UserRepository) List(ctx context.Context, p pagination.Params) ([]*entity.User, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(phone,''), COALESCE(email,''), full_name, role, is_verified, is_banned, is_flagged,
		       listing_count_today, listing_count_date, upi_id, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit, p.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		u := &entity.User{}
		if err := rows.Scan(
			&u.ID, &u.Phone, &u.Email, &u.FullName, &u.Role, &u.IsVerified, &u.IsBanned, &u.IsFlagged,
			&u.ListingCountToday, &u.ListingCountDate, &u.UPIID, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *UserRepository) Ban(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET is_banned=true, updated_at=now() WHERE id=$1`, id)
	return err
}
