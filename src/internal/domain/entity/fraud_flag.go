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
	ID         uuid.UUID     `db:"id"          json:"id"`
	UserID     uuid.UUID     `db:"user_id"     json:"user_id"`
	ListingID  *uuid.UUID    `db:"listing_id"  json:"listing_id,omitempty"`
	Reason     string        `db:"reason"      json:"reason"`
	Severity   FraudSeverity `db:"severity"    json:"severity"`
	IsResolved bool          `db:"is_resolved" json:"is_resolved"`
	CreatedAt  time.Time     `db:"created_at"  json:"created_at"`
	ResolvedAt *time.Time    `db:"resolved_at" json:"resolved_at,omitempty"`
}
