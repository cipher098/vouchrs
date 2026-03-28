package entity

import (
	"time"

	"github.com/google/uuid"
)

type BuyRequestStatus string

const (
	BuyRequestStatusActive    BuyRequestStatus = "active"
	BuyRequestStatusFulfilled BuyRequestStatus = "fulfilled"
	BuyRequestStatusExpired   BuyRequestStatus = "expired"
	BuyRequestStatusCancelled BuyRequestStatus = "cancelled"
)

// BuyRequest is a buyer's notification request —
// alert me when a card matching these criteria becomes available.
type BuyRequest struct {
	ID           uuid.UUID        `db:"id"`
	UserID       uuid.UUID        `db:"user_id"`
	BrandID      uuid.UUID        `db:"brand_id"`
	MinValue     float64          `db:"min_value"`
	MaxValue     float64          `db:"max_value"`
	MaxPrice     float64          `db:"max_price"`
	Status       BuyRequestStatus `db:"status"`
	AlertedCount int              `db:"alerted_count"`
	ExpiresAt    time.Time        `db:"expires_at"`
	CreatedAt    time.Time        `db:"created_at"`
}
