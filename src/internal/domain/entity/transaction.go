package entity

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	TxnStatusPending   TransactionStatus = "pending"   // lock acquired, waiting for payment
	TxnStatusPaid      TransactionStatus = "paid"       // payment confirmed, code sent by email
	TxnStatusCompleted TransactionStatus = "completed"  // buyer confirmed redemption
	TxnStatusCancelled TransactionStatus = "cancelled"
	TxnStatusRefunded  TransactionStatus = "refunded"
)

type Transaction struct {
	ID             uuid.UUID         `db:"id"               json:"id"`
	ListingID      uuid.UUID         `db:"listing_id"       json:"listing_id"`
	BuyerID        uuid.UUID         `db:"buyer_id"         json:"buyer_id"`
	SellerID       uuid.UUID         `db:"seller_id"        json:"seller_id"`
	BuyerAmount    float64           `db:"buyer_amount"     json:"buyer_amount"`
	SellerPayout   float64           `db:"seller_payout"    json:"seller_payout"`
	PaymentRef     string            `db:"payment_ref"      json:"payment_ref"`
	PayoutRef      string            `db:"payout_ref"       json:"payout_ref,omitempty"`
	Status         TransactionStatus `db:"status"           json:"status"`
	LockStartedAt  *time.Time        `db:"lock_started_at"  json:"lock_started_at,omitempty"`
	PaidAt         *time.Time        `db:"paid_at"          json:"paid_at,omitempty"`
	CodeRevealedAt *time.Time        `db:"code_revealed_at" json:"code_revealed_at,omitempty"`
	CompletedAt    *time.Time        `db:"completed_at"     json:"completed_at,omitempty"`
	CreatedAt      time.Time         `db:"created_at"       json:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"       json:"updated_at"`
}
