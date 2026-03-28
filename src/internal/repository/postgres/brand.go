package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
)

type BrandRepository struct {
	db *pgxpool.Pool
}

func NewBrandRepository(db *pgxpool.Pool) *BrandRepository {
	return &BrandRepository{db: db}
}

func (r *BrandRepository) Create(ctx context.Context, b *entity.Brand) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO brands (id, name, slug, logo_url, verification_source, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, b.Name, b.Slug, b.LogoURL, b.VerificationSource, b.Status, b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create brand: %w", err)
	}
	return nil
}

func (r *BrandRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Brand, error) {
	b := &entity.Brand{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, logo_url, verification_source, status, created_at, updated_at
		FROM brands WHERE id = $1`, id).Scan(
		&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.VerificationSource, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find brand by id: %w", err)
	}
	return b, nil
}

func (r *BrandRepository) FindBySlug(ctx context.Context, slug string) (*entity.Brand, error) {
	b := &entity.Brand{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, slug, logo_url, verification_source, status, created_at, updated_at
		FROM brands WHERE slug = $1`, slug).Scan(
		&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.VerificationSource, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find brand by slug: %w", err)
	}
	return b, nil
}

func (r *BrandRepository) ListActive(ctx context.Context) ([]*entity.Brand, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, logo_url, verification_source, status, created_at, updated_at
		FROM brands WHERE status = $1 ORDER BY name`, entity.BrandStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list active brands: %w", err)
	}
	defer rows.Close()

	var brands []*entity.Brand
	for rows.Next() {
		b := &entity.Brand{}
		if err := rows.Scan(
			&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.VerificationSource, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		brands = append(brands, b)
	}
	return brands, rows.Err()
}

func (r *BrandRepository) ListWithCount(ctx context.Context) ([]*entity.Brand, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.name, b.slug, b.logo_url, b.color, b.verification_source, b.status, b.created_at, b.updated_at,
		       COUNT(l.id) FILTER (WHERE l.status = 'LIVE') AS listing_count
		FROM brands b
		LEFT JOIN listings l ON l.brand_id = b.id
		WHERE b.status = $1
		GROUP BY b.id
		ORDER BY b.name`, entity.BrandStatusActive)
	if err != nil {
		return nil, fmt.Errorf("list brands with count: %w", err)
	}
	defer rows.Close()

	var brands []*entity.Brand
	for rows.Next() {
		b := &entity.Brand{}
		if err := rows.Scan(
			&b.ID, &b.Name, &b.Slug, &b.LogoURL, &b.Color, &b.VerificationSource, &b.Status, &b.CreatedAt, &b.UpdatedAt,
			&b.ListingCount,
		); err != nil {
			return nil, fmt.Errorf("scan brand with count: %w", err)
		}
		brands = append(brands, b)
	}
	return brands, rows.Err()
}

func (r *BrandRepository) Update(ctx context.Context, b *entity.Brand) error {
	b.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE brands SET name=$1, slug=$2, logo_url=$3, verification_source=$4, status=$5, updated_at=$6
		WHERE id=$7`,
		b.Name, b.Slug, b.LogoURL, b.VerificationSource, b.Status, b.UpdatedAt, b.ID,
	)
	return err
}
