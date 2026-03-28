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
	ID           uuid.UUID        `db:"id"            json:"id"`
	UserID       uuid.UUID        `db:"user_id"       json:"user_id"`
	BrandID      uuid.UUID        `db:"brand_id"      json:"brand_id"`
	MinValue     float64          `db:"min_value"     json:"min_value"`
	MaxValue     float64          `db:"max_value"     json:"max_value"`
	MaxPrice     float64          `db:"max_price"     json:"max_price"`
	Status       BuyRequestStatus `db:"status"        json:"status"`
	AlertedCount int              `db:"alerted_count" json:"alerted_count"`
	ExpiresAt    time.Time        `db:"expires_at"    json:"expires_at"`
	CreatedAt    time.Time        `db:"created_at"    json:"created_at"`
}
