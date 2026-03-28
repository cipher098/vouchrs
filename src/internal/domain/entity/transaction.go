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
	ID              uuid.UUID         `db:"id"`
	ListingID       uuid.UUID         `db:"listing_id"`
	BuyerID         uuid.UUID         `db:"buyer_id"`
	SellerID        uuid.UUID         `db:"seller_id"`
	BuyerAmount     float64           `db:"buyer_amount"`
	SellerPayout    float64           `db:"seller_payout"`
	PaymentRef      string            `db:"payment_ref"`  // PG merchant transaction ID
	PayoutRef       string            `db:"payout_ref"`   // Razorpay payout ID
	Status          TransactionStatus `db:"status"`
	LockStartedAt   *time.Time        `db:"lock_started_at"`
	PaidAt          *time.Time        `db:"paid_at"`
	CodeRevealedAt  *time.Time        `db:"code_revealed_at"` // when email was sent
	CompletedAt     *time.Time        `db:"completed_at"`
	CreatedAt       time.Time         `db:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at"`
}
