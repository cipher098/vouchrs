package entity

import (
	"time"

	"github.com/google/uuid"
)

type CardRequestStatus string

const (
	CardRequestStatusPendingReview CardRequestStatus = "PENDING_ADMIN_REVIEW"
	CardRequestStatusUnderReview   CardRequestStatus = "UNDER_REVIEW"
	CardRequestStatusFulfilled     CardRequestStatus = "FULFILLED"
	CardRequestStatusRejected      CardRequestStatus = "REJECTED"
	CardRequestStatusDeferred      CardRequestStatus = "DEFERRED"
)

type CardRequestUrgency string

const (
	CardRequestUrgencyFlexible CardRequestUrgency = "flexible"
	CardRequestUrgency24h      CardRequestUrgency = "within_24h"
	CardRequestUrgency1Week    CardRequestUrgency = "within_1_week"
)

// CardRequest is a buyer request for a brand/value not yet on platform.
// Goes into the admin queue for manual review.
type CardRequest struct {
	ID           uuid.UUID          `db:"id"            json:"id"`
	UserID       uuid.UUID          `db:"user_id"       json:"user_id"`
	Brand        string             `db:"brand"         json:"brand"`
	DesiredValue float64            `db:"desired_value" json:"desired_value"`
	Urgency      CardRequestUrgency `db:"urgency"       json:"urgency"`
	Status       CardRequestStatus  `db:"status"        json:"status"`
	AdminNotes   string             `db:"admin_notes"   json:"admin_notes,omitempty"`
	FulfilledAt  *time.Time         `db:"fulfilled_at"  json:"fulfilled_at,omitempty"`
	CreatedAt    time.Time          `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"    json:"updated_at"`
}
