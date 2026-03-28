package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type ListingRepository struct {
	db *pgxpool.Pool
}

func NewListingRepository(db *pgxpool.Pool) *ListingRepository {
	return &ListingRepository{db: db}
}

const listingCols = `id, seller_id, brand_id, face_value, buyer_price, seller_payout, discount_pct,
	is_pool, expiry_date, code_encrypted, code_hash, pin_encrypted, status, lock_buyer_id, lock_expires_at,
	gate1_at, sold_at, verified_balance, created_at, updated_at`

func scanListing(row pgx.Row) (*entity.Listing, error) {
	l := &entity.Listing{}
	err := row.Scan(
		&l.ID, &l.SellerID, &l.BrandID, &l.FaceValue, &l.BuyerPrice, &l.SellerPayout, &l.DiscountPct,
		&l.IsPool, &l.ExpiryDate, &l.CodeEncrypted, &l.CodeHash, &l.PinEncrypted, &l.Status, &l.LockBuyerID, &l.LockExpiresAt,
		&l.Gate1At, &l.SoldAt, &l.VerifiedBalance, &l.CreatedAt, &l.UpdatedAt,
	)
	return l, err
}

func (r *ListingRepository) Create(ctx context.Context, l *entity.Listing) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	now := time.Now().UTC()
	l.CreatedAt = now
	l.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO listings (`+listingCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		l.ID, l.SellerID, l.BrandID, l.FaceValue, l.BuyerPrice, l.SellerPayout, l.DiscountPct,
		l.IsPool, l.ExpiryDate, l.CodeEncrypted, l.CodeHash, l.PinEncrypted, l.Status, l.LockBuyerID, l.LockExpiresAt,
		l.Gate1At, l.SoldAt, l.VerifiedBalance, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create listing: %w", err)
	}
	return nil
}

func (r *ListingRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Listing, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+listingCols+` FROM listings WHERE id = $1`, id)
	l, err := scanListing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find listing by id: %w", err)
	}
	return l, nil
}

func (r *ListingRepository) FindByCodeHash(ctx context.Context, hash string) (*entity.Listing, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+listingCols+` FROM listings WHERE code_hash = $1 AND status != $2`,
		hash, entity.ListingStatusCancelled)
	l, err := scanListing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find listing by code hash: %w", err)
	}
	return l, nil
}

func (r *ListingRepository) ListMarketplace(ctx context.Context, f port.MarketplaceFilter) ([]*entity.Listing, int, error) {
	where := []string{"status = 'LIVE'"}
	args := []interface{}{}
	idx := 1

	if f.BrandID != nil {
		where = append(where, fmt.Sprintf("brand_id = $%d", idx))
		args = append(args, *f.BrandID)
		idx++
	}
	if f.MinValue != nil {
		where = append(where, fmt.Sprintf("face_value >= $%d", idx))
		args = append(args, *f.MinValue)
		idx++
	}
	if f.MaxValue != nil {
		where = append(where, fmt.Sprintf("face_value <= $%d", idx))
		args = append(args, *f.MaxValue)
		idx++
	}
	if f.IsPool != nil {
		where = append(where, fmt.Sprintf("is_pool = $%d", idx))
		args = append(args, *f.IsPool)
		idx++
	}

	orderBy := "created_at DESC"
	switch f.SortBy {
	case "discount_desc":
		orderBy = "discount_pct DESC"
	case "price_asc":
		orderBy = "buyer_price ASC"
	case "value_desc":
		orderBy = "face_value DESC"
	}

	whereClause := strings.Join(where, " AND ")
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM listings WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count marketplace listings: %w", err)
	}

	args = append(args, f.Pagination.Limit, f.Pagination.Offset)
	query := fmt.Sprintf(`SELECT `+listingCols+` FROM listings WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		whereClause, orderBy, idx, idx+1)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list marketplace: %w", err)
	}
	defer rows.Close()

	var listings []*entity.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan listing: %w", err)
		}
		listings = append(listings, l)
	}
	return listings, total, rows.Err()
}

func (r *ListingRepository) ListBySeller(ctx context.Context, sellerID uuid.UUID, p pagination.Params) ([]*entity.Listing, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM listings WHERE seller_id=$1`, sellerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+listingCols+` FROM listings WHERE seller_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		sellerID, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var listings []*entity.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, 0, err
		}
		listings = append(listings, l)
	}
	return listings, total, rows.Err()
}

func (r *ListingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ListingStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE listings SET status=$1, updated_at=now() WHERE id=$2`, status, id)
	return err
}

// Lock atomically sets LOCKED status — fails if not currently LIVE.
func (r *ListingRepository) Lock(ctx context.Context, id uuid.UUID, buyerID uuid.UUID, expiresAt time.Time) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE listings
		SET status=$1, lock_buyer_id=$2, lock_expires_at=$3, updated_at=now()
		WHERE id=$4 AND status=$5`,
		entity.ListingStatusLocked, buyerID, expiresAt, id, entity.ListingStatusLive,
	)
	if err != nil {
		return fmt.Errorf("lock listing: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return apperror.ErrListingNotLive
	}
	return nil
}

func (r *ListingRepository) Unlock(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE listings
		SET status=$1, lock_buyer_id=NULL, lock_expires_at=NULL, updated_at=now()
		WHERE id=$2`,
		entity.ListingStatusLive, id)
	return err
}

func (r *ListingRepository) MarkSold(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE listings SET status=$1, sold_at=$2, lock_buyer_id=NULL, lock_expires_at=NULL, updated_at=now()
		WHERE id=$3`,
		entity.ListingStatusSold, now, id)
	return err
}

// OldestLiveInPool returns the oldest available pool listing for a brand+value (FIFO).
// Only returns listings that are LIVE and not locked (lock_buyer_id IS NULL).
func (r *ListingRepository) OldestLiveInPool(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.Listing, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+listingCols+` FROM listings
		WHERE brand_id=$1 AND face_value=$2 AND is_pool=true
		  AND status=$3 AND lock_buyer_id IS NULL
		ORDER BY created_at ASC LIMIT 1`,
		brandID, faceValue, entity.ListingStatusLive)
	l, err := scanListing(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return l, err
}

func (r *ListingRepository) FindExpiredLocks(ctx context.Context) ([]*entity.Listing, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+listingCols+` FROM listings
		WHERE status=$1 AND lock_expires_at < now()`,
		entity.ListingStatusLocked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []*entity.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

func (r *ListingRepository) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Listing, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM listings`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+listingCols+` FROM listings ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var listings []*entity.Listing
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, 0, err
		}
		listings = append(listings, l)
	}
	return listings, total, rows.Err()
}
