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

type PoolGroupRepository struct {
	db *pgxpool.Pool
}

func NewPoolGroupRepository(db *pgxpool.Pool) *PoolGroupRepository {
	return &PoolGroupRepository{db: db}
}

const poolCols = `id, brand_id, face_value, recommended_price, buyer_price, discount_pct, active_count, avg_sell_time_mins, created_at, updated_at`

func scanPoolGroup(row pgx.Row) (*entity.PoolGroup, error) {
	p := &entity.PoolGroup{}
	err := row.Scan(
		&p.ID, &p.BrandID, &p.FaceValue, &p.RecommendedPrice, &p.BuyerPrice,
		&p.DiscountPct, &p.ActiveCount, &p.AvgSellTimeMins, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

// Upsert creates a pool group if one doesn't exist for this brand+value, or returns the existing one.
func (r *PoolGroupRepository) Upsert(ctx context.Context, brandID uuid.UUID, faceValue, buyerPrice, sellerPayout, discountPct float64) (*entity.PoolGroup, error) {
	now := time.Now().UTC()
	id := uuid.New()
	row := r.db.QueryRow(ctx, `
		INSERT INTO pool_groups (id, brand_id, face_value, recommended_price, buyer_price, discount_pct, active_count, avg_sell_time_mins, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,0,0,$7,$8)
		ON CONFLICT (brand_id, face_value) DO UPDATE
		SET buyer_price=EXCLUDED.buyer_price, recommended_price=EXCLUDED.recommended_price,
		    discount_pct=EXCLUDED.discount_pct, updated_at=EXCLUDED.updated_at
		RETURNING `+poolCols,
		id, brandID, faceValue, sellerPayout, buyerPrice, discountPct, now, now,
	)
	return scanPoolGroup(row)
}

func (r *PoolGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.PoolGroup, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+poolCols+` FROM pool_groups WHERE id=$1`, id)
	p, err := scanPoolGroup(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return p, err
}

func (r *PoolGroupRepository) FindByBrandAndValue(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.PoolGroup, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+poolCols+` FROM pool_groups WHERE brand_id=$1 AND face_value=$2`,
		brandID, faceValue)
	p, err := scanPoolGroup(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return p, err
}

func (r *PoolGroupRepository) IncrCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE pool_groups SET active_count=active_count+1, updated_at=now() WHERE id=$1`, id)
	return fmt.Errorf("incr pool count: %w", err)
}

func (r *PoolGroupRepository) DecrCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE pool_groups SET active_count=GREATEST(0, active_count-1), updated_at=now() WHERE id=$1`, id)
	return err
}

func (r *PoolGroupRepository) List(ctx context.Context) ([]*entity.PoolGroup, error) {
	// active_count is overwritten with the real-time count of LIVE + unlocked listings
	// so buyers never see stale or locked-item counts.
	rows, err := r.db.Query(ctx, `
		SELECT pg.id, pg.brand_id, pg.face_value, pg.recommended_price, pg.buyer_price,
		       pg.discount_pct,
		       COUNT(l.id) FILTER (WHERE l.status = 'LIVE' AND l.lock_buyer_id IS NULL) AS active_count,
		       pg.avg_sell_time_mins, pg.created_at, pg.updated_at
		FROM pool_groups pg
		LEFT JOIN listings l ON l.brand_id = pg.brand_id
		                     AND l.face_value = pg.face_value
		                     AND l.is_pool = true
		GROUP BY pg.id
		HAVING COUNT(l.id) FILTER (WHERE l.status = 'LIVE' AND l.lock_buyer_id IS NULL) > 0
		ORDER BY pg.brand_id, pg.face_value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*entity.PoolGroup
	for rows.Next() {
		p, err := scanPoolGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, p)
	}
	return groups, rows.Err()
}
