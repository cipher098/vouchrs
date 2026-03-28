package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
)

type VerificationLogRepository struct {
	db *pgxpool.Pool
}

func NewVerificationLogRepository(db *pgxpool.Pool) *VerificationLogRepository {
	return &VerificationLogRepository{db: db}
}

func (r *VerificationLogRepository) Create(ctx context.Context, l *entity.VerificationLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	l.CheckedAt = time.Now().UTC()

	_, err := r.db.Exec(ctx, `
		INSERT INTO verification_logs (id, listing_id, gate, result, balance_found, fail_reason, response_hash, checked_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		l.ID, l.ListingID, l.Gate, l.Result, l.BalanceFound, l.FailReason, l.ResponseHash, l.CheckedAt,
	)
	return err
}

func (r *VerificationLogRepository) FindLatest(ctx context.Context, listingID uuid.UUID, gate int) (*entity.VerificationLog, error) {
	l := &entity.VerificationLog{}
	err := r.db.QueryRow(ctx, `
		SELECT id, listing_id, gate, result, balance_found, fail_reason, response_hash, checked_at
		FROM verification_logs
		WHERE listing_id=$1 AND gate=$2
		ORDER BY checked_at DESC LIMIT 1`, listingID, gate).Scan(
		&l.ID, &l.ListingID, &l.Gate, &l.Result, &l.BalanceFound, &l.FailReason, &l.ResponseHash, &l.CheckedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return l, err
}
