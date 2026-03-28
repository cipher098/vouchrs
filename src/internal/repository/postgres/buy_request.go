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

type BuyRequestRepository struct {
	db *pgxpool.Pool
}

func NewBuyRequestRepository(db *pgxpool.Pool) *BuyRequestRepository {
	return &BuyRequestRepository{db: db}
}

const buyReqCols = `id, user_id, brand_id, min_value, max_value, max_price, status, alerted_count, expires_at, created_at`

func scanBuyRequest(row pgx.Row) (*entity.BuyRequest, error) {
	r := &entity.BuyRequest{}
	err := row.Scan(
		&r.ID, &r.UserID, &r.BrandID, &r.MinValue, &r.MaxValue,
		&r.MaxPrice, &r.Status, &r.AlertedCount, &r.ExpiresAt, &r.CreatedAt,
	)
	return r, err
}

func (r *BuyRequestRepository) Create(ctx context.Context, req *entity.BuyRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	req.CreatedAt = time.Now().UTC()
	req.Status = entity.BuyRequestStatusActive

	_, err := r.db.Exec(ctx, `
		INSERT INTO buy_requests (`+buyReqCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.UserID, req.BrandID, req.MinValue, req.MaxValue,
		req.MaxPrice, req.Status, req.AlertedCount, req.ExpiresAt, req.CreatedAt,
	)
	return err
}

func (r *BuyRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.BuyRequest, error) {
	row := r.db.QueryRow(ctx, `SELECT `+buyReqCols+` FROM buy_requests WHERE id=$1`, id)
	req, err := scanBuyRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return req, err
}

func (r *BuyRequestRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.BuyRequest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+buyReqCols+` FROM buy_requests WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list buy requests: %w", err)
	}
	defer rows.Close()
	var reqs []*entity.BuyRequest
	for rows.Next() {
		req, err := scanBuyRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *BuyRequestRepository) FindMatchingForListing(ctx context.Context, l *entity.Listing) ([]*entity.BuyRequest, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+buyReqCols+` FROM buy_requests
		WHERE brand_id=$1
		  AND min_value <= $2 AND max_value >= $2
		  AND max_price >= $3
		  AND status=$4
		  AND expires_at > now()`,
		l.BrandID, l.FaceValue, l.BuyerPrice, entity.BuyRequestStatusActive)
	if err != nil {
		return nil, fmt.Errorf("find matching buy requests: %w", err)
	}
	defer rows.Close()
	var reqs []*entity.BuyRequest
	for rows.Next() {
		req, err := scanBuyRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *BuyRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.BuyRequestStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE buy_requests SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *BuyRequestRepository) IncrAlertCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE buy_requests SET alerted_count=alerted_count+1 WHERE id=$1`, id)
	return err
}

func (r *BuyRequestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM buy_requests WHERE id=$1`, id)
	return err
}
