package entity

import (
	"time"

	"github.com/google/uuid"
)

type FraudSeverity string

const (
	FraudSeverityLow    FraudSeverity = "low"
	FraudSeverityMedium FraudSeverity = "medium"
	FraudSeverityHigh   FraudSeverity = "high"
)

type FraudFlag struct {
	ID         uuid.UUID     `db:"id"`
	UserID     uuid.UUID     `db:"user_id"`
	ListingID  *uuid.UUID    `db:"listing_id"`
	Reason     string        `db:"reason"`
	Severity   FraudSeverity `db:"severity"`
	IsResolved bool          `db:"is_resolved"`
	CreatedAt  time.Time     `db:"created_at"`
	ResolvedAt *time.Time    `db:"resolved_at"`
}
