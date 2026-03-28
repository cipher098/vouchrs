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

type CardRequestRepository struct {
	db *pgxpool.Pool
}

func NewCardRequestRepository(db *pgxpool.Pool) *CardRequestRepository {
	return &CardRequestRepository{db: db}
}

const cardReqCols = `id, user_id, brand, desired_value, urgency, status, admin_notes, fulfilled_at, created_at, updated_at`

func scanCardRequest(row pgx.Row) (*entity.CardRequest, error) {
	r := &entity.CardRequest{}
	err := row.Scan(
		&r.ID, &r.UserID, &r.Brand, &r.DesiredValue, &r.Urgency,
		&r.Status, &r.AdminNotes, &r.FulfilledAt, &r.CreatedAt, &r.UpdatedAt,
	)
	return r, err
}

func (r *CardRequestRepository) Create(ctx context.Context, req *entity.CardRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	now := time.Now().UTC()
	req.CreatedAt = now
	req.UpdatedAt = now
	req.Status = entity.CardRequestStatusPendingReview

	_, err := r.db.Exec(ctx, `
		INSERT INTO card_requests (`+cardReqCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.UserID, req.Brand, req.DesiredValue, req.Urgency,
		req.Status, req.AdminNotes, req.FulfilledAt, req.CreatedAt, req.UpdatedAt,
	)
	return err
}

func (r *CardRequestRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.CardRequest, error) {
	row := r.db.QueryRow(ctx, `SELECT `+cardReqCols+` FROM card_requests WHERE id=$1`, id)
	req, err := scanCardRequest(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return req, err
}

func (r *CardRequestRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.CardRequest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cardReqCols+` FROM card_requests WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list card requests by user: %w", err)
	}
	defer rows.Close()
	var reqs []*entity.CardRequest
	for rows.Next() {
		req, err := scanCardRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *CardRequestRepository) ListPending(ctx context.Context) ([]*entity.CardRequest, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+cardReqCols+` FROM card_requests WHERE status IN ($1,$2) ORDER BY created_at ASC`,
		entity.CardRequestStatusPendingReview, entity.CardRequestStatusUnderReview)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []*entity.CardRequest
	for rows.Next() {
		req, err := scanCardRequest(rows)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, rows.Err()
}

func (r *CardRequestRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.CardRequestStatus, notes string) error {
	var fulfilledAt *time.Time
	if status == entity.CardRequestStatusFulfilled {
		now := time.Now().UTC()
		fulfilledAt = &now
	}
	_, err := r.db.Exec(ctx, `
		UPDATE card_requests SET status=$1, admin_notes=$2, fulfilled_at=$3, updated_at=now()
		WHERE id=$4`, status, notes, fulfilledAt, id)
	return err
}
