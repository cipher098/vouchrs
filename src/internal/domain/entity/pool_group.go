package entity

import (
	"time"

	"github.com/google/uuid"
)

// PoolGroup aggregates all CardSwap-pool listings for a given brand + face value.
// It is the single "CardSwap" listing that buyers see on the marketplace.
type PoolGroup struct {
	ID               uuid.UUID `db:"id"`
	BrandID          uuid.UUID `db:"brand_id"`
	FaceValue        float64   `db:"face_value"`
	RecommendedPrice float64   `db:"recommended_price"` // seller receives this
	BuyerPrice       float64   `db:"buyer_price"`
	DiscountPct      float64   `db:"discount_pct"`
	ActiveCount      int       `db:"active_count"` // LIVE listings in pool
	AvgSellTimeMins  float64   `db:"avg_sell_time_mins"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}
