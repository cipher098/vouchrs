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
	ID              uuid.UUID     `db:"id"               json:"id"`
	SellerID        uuid.UUID     `db:"seller_id"        json:"seller_id"`
	BrandID         uuid.UUID     `db:"brand_id"         json:"brand_id"`
	FaceValue       float64       `db:"face_value"       json:"face_value"`
	BuyerPrice      float64       `db:"buyer_price"      json:"buyer_price"`
	SellerPayout    float64       `db:"seller_payout"    json:"seller_payout"`
	DiscountPct     float64       `db:"discount_pct"     json:"discount_pct"`
	IsPool          bool          `db:"is_pool"          json:"is_pool"`
	ExpiryDate      string        `db:"expiry_date"      json:"expiry_date"`
	CodeEncrypted   string        `db:"code_encrypted"   json:"code_encrypted,omitempty"`
	CodeHash        string        `db:"code_hash"        json:"code_hash,omitempty"`
	PinEncrypted    string        `db:"pin_encrypted"    json:"pin_encrypted,omitempty"`
	Status          ListingStatus `db:"status"           json:"status"`
	LockBuyerID     *uuid.UUID    `db:"lock_buyer_id"    json:"lock_buyer_id,omitempty"`
	LockExpiresAt   *time.Time    `db:"lock_expires_at"  json:"lock_expires_at,omitempty"`
	Gate1At         *time.Time    `db:"gate1_at"         json:"gate1_at,omitempty"`
	SoldAt          *time.Time    `db:"sold_at"          json:"sold_at,omitempty"`
	VerifiedBalance float64       `db:"verified_balance" json:"verified_balance"`
	CreatedAt       time.Time     `db:"created_at"       json:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"       json:"updated_at"`
}

// IsAvailable returns true if the listing can be purchased.
func (l *Listing) IsAvailable() bool {
	return l.Status == ListingStatusLive
}
