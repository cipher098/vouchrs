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
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type TransactionRepository struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{db: db}
}

const txnCols = `id, listing_id, buyer_id, seller_id, buyer_amount, seller_payout,
	payment_ref, payout_ref, status, lock_started_at, paid_at, code_revealed_at, completed_at,
	created_at, updated_at`

func scanTxn(row pgx.Row) (*entity.Transaction, error) {
	t := &entity.Transaction{}
	err := row.Scan(
		&t.ID, &t.ListingID, &t.BuyerID, &t.SellerID, &t.BuyerAmount, &t.SellerPayout,
		&t.PaymentRef, &t.PayoutRef, &t.Status, &t.LockStartedAt, &t.PaidAt,
		&t.CodeRevealedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

func (r *TransactionRepository) Create(ctx context.Context, t *entity.Transaction) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO transactions (`+txnCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		t.ID, t.ListingID, t.BuyerID, t.SellerID, t.BuyerAmount, t.SellerPayout,
		t.PaymentRef, t.PayoutRef, t.Status, t.LockStartedAt, t.PaidAt,
		t.CodeRevealedAt, t.CompletedAt, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *TransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	row := r.db.QueryRow(ctx, `SELECT `+txnCols+` FROM transactions WHERE id=$1`, id)
	t, err := scanTxn(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return t, err
}

func (r *TransactionRepository) FindByListingID(ctx context.Context, listingID uuid.UUID) (*entity.Transaction, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+txnCols+` FROM transactions WHERE listing_id=$1 ORDER BY created_at DESC LIMIT 1`, listingID)
	t, err := scanTxn(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return t, err
}

func (r *TransactionRepository) FindByPaymentRef(ctx context.Context, ref string) (*entity.Transaction, error) {
	row := r.db.QueryRow(ctx, `SELECT `+txnCols+` FROM transactions WHERE payment_ref=$1`, ref)
	t, err := scanTxn(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return t, err
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.TransactionStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE transactions SET status=$1, updated_at=now() WHERE id=$2`, status, id)
	return err
}

func (r *TransactionRepository) SetPaymentRef(ctx context.Context, id uuid.UUID, ref string) error {
	_, err := r.db.Exec(ctx, `UPDATE transactions SET payment_ref=$1, updated_at=now() WHERE id=$2`, ref, id)
	return err
}

func (r *TransactionRepository) SetPayoutRef(ctx context.Context, id uuid.UUID, ref string) error {
	_, err := r.db.Exec(ctx, `UPDATE transactions SET payout_ref=$1, updated_at=now() WHERE id=$2`, ref, id)
	return err
}

func (r *TransactionRepository) SetPaidAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE transactions SET paid_at=$1, status=$2, updated_at=now() WHERE id=$3`,
		now, entity.TxnStatusPaid, id)
	return err
}

func (r *TransactionRepository) SetCodeRevealedAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE transactions SET code_revealed_at=$1, updated_at=now() WHERE id=$2`, now, id)
	return err
}

func (r *TransactionRepository) SetCompletedAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE transactions SET completed_at=$1, status=$2, updated_at=now() WHERE id=$3`,
		now, entity.TxnStatusCompleted, id)
	return err
}

func (r *TransactionRepository) ListByBuyer(ctx context.Context, buyerID uuid.UUID, p pagination.Params) ([]*entity.Transaction, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE buyer_id=$1`, buyerID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count buyer transactions: %w", err)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+txnCols+` FROM transactions WHERE buyer_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		buyerID, p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var txns []*entity.Transaction
	for rows.Next() {
		t, err := scanTxn(rows)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}

func (r *TransactionRepository) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM transactions`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+txnCols+` FROM transactions ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.Limit, p.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var txns []*entity.Transaction
	for rows.Next() {
		t, err := scanTxn(rows)
		if err != nil {
			return nil, 0, err
		}
		txns = append(txns, t)
	}
	return txns, total, rows.Err()
}
