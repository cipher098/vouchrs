package entity

import (
	"time"

	"github.com/google/uuid"
)

// VerificationLog records every Gate 1 and Gate 2 Qwikcilver check.
type VerificationLog struct {
	ID           uuid.UUID `db:"id"`
	ListingID    uuid.UUID `db:"listing_id"`
	Gate         int       `db:"gate"`          // 1 or 2
	Result       string    `db:"result"`        // "pass" or "fail"
	BalanceFound float64   `db:"balance_found"`
	FailReason   string    `db:"fail_reason"`
	ResponseHash string    `db:"response_hash"` // SHA-256 of raw response for audit
	CheckedAt    time.Time `db:"checked_at"`
}
