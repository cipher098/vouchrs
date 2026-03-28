package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
)

type FraudFlagRepository struct {
	db *pgxpool.Pool
}

func NewFraudFlagRepository(db *pgxpool.Pool) *FraudFlagRepository {
	return &FraudFlagRepository{db: db}
}

const fraudFlagCols = `id, user_id, listing_id, reason, severity, is_resolved, created_at, resolved_at`

func scanFraudFlag(row pgx.Row) (*entity.FraudFlag, error) {
	f := &entity.FraudFlag{}
	err := row.Scan(
		&f.ID, &f.UserID, &f.ListingID, &f.Reason, &f.Severity,
		&f.IsResolved, &f.CreatedAt, &f.ResolvedAt,
	)
	return f, err
}

func (r *FraudFlagRepository) Create(ctx context.Context, f *entity.FraudFlag) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	f.CreatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		INSERT INTO fraud_flags (`+fraudFlagCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		f.ID, f.UserID, f.ListingID, f.Reason, f.Severity, f.IsResolved, f.CreatedAt, f.ResolvedAt,
	)
	return err
}

func (r *FraudFlagRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*entity.FraudFlag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+fraudFlagCols+` FROM fraud_flags WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flags []*entity.FraudFlag
	for rows.Next() {
		f, err := scanFraudFlag(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *FraudFlagRepository) FindByListing(ctx context.Context, listingID uuid.UUID) ([]*entity.FraudFlag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+fraudFlagCols+` FROM fraud_flags WHERE listing_id=$1 ORDER BY created_at DESC`, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flags []*entity.FraudFlag
	for rows.Next() {
		f, err := scanFraudFlag(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *FraudFlagRepository) ListUnresolved(ctx context.Context) ([]*entity.FraudFlag, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+fraudFlagCols+` FROM fraud_flags WHERE is_resolved=false ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var flags []*entity.FraudFlag
	for rows.Next() {
		f, err := scanFraudFlag(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}
	return flags, rows.Err()
}

func (r *FraudFlagRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx,
		`UPDATE fraud_flags SET is_resolved=true, resolved_at=$1 WHERE id=$2`, now, id)
	return err
}
