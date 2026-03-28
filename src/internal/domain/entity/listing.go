package entity

import (
	"time"

	"github.com/google/uuid"
)

type ListingStatus string

const (
	ListingStatusLive      ListingStatus = "LIVE"
	ListingStatusLocked    ListingStatus = "LOCKED"
	ListingStatusSold      ListingStatus = "SOLD"
	ListingStatusExpired   ListingStatus = "EXPIRED"
	ListingStatusCancelled ListingStatus = "CANCELLED"
	ListingStatusFraudHold ListingStatus = "FRAUD_HOLD"
)

type Listing struct {
	ID             uuid.UUID     `db:"id"`
	SellerID       uuid.UUID     `db:"seller_id"`
	BrandID        uuid.UUID     `db:"brand_id"`
	FaceValue      float64       `db:"face_value"`       // e.g. 1000.00
	BuyerPrice     float64       `db:"buyer_price"`      // what buyer pays
	SellerPayout   float64       `db:"seller_payout"`    // what seller receives
	DiscountPct    float64       `db:"discount_pct"`     // e.g. 9.0 for 9% off
	IsPool         bool          `db:"is_pool"`          // true = CardSwap pool
	CodeEncrypted  string        `db:"code_encrypted"`   // AES-256-GCM encrypted
	CodeHash       string        `db:"code_hash"`        // SHA-256 for duplicate check
	Status         ListingStatus `db:"status"`
	LockBuyerID    *uuid.UUID    `db:"lock_buyer_id"`
	LockExpiresAt  *time.Time    `db:"lock_expires_at"`
	Gate1At        *time.Time    `db:"gate1_at"`
	SoldAt         *time.Time    `db:"sold_at"`
	VerifiedBalance float64      `db:"verified_balance"` // balance at Gate 1
	CreatedAt      time.Time     `db:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at"`
}

// IsAvailable returns true if the listing can be purchased.
func (l *Listing) IsAvailable() bool {
	return l.Status == ListingStatusLive
}
