package entity

import (
	"time"

	"github.com/google/uuid"
)

// PoolGroup aggregates all Vouchrs-pool listings for a given brand + face value.
// It is the single "Vouchrs" listing that buyers see on the marketplace.
type PoolGroup struct {
	ID               uuid.UUID `db:"id"                 json:"id"`
	BrandID          uuid.UUID `db:"brand_id"           json:"brand_id"`
	FaceValue        float64   `db:"face_value"         json:"face_value"`
	RecommendedPrice float64   `db:"recommended_price"  json:"recommended_price"`
	BuyerPrice       float64   `db:"buyer_price"        json:"buyer_price"`
	DiscountPct      float64   `db:"discount_pct"       json:"discount_pct"`
	ActiveCount      int       `db:"active_count"       json:"active_count"`
	AvgSellTimeMins  float64   `db:"avg_sell_time_mins" json:"avg_sell_time_mins"`
	CreatedAt        time.Time `db:"created_at"         json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"         json:"updated_at"`
}
