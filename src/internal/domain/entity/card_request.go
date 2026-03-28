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

// CardRequest is a buyer or seller request for a brand/value not yet on platform.
// Goes into the admin queue for manual review.
type CardRequest struct {
	ID           uuid.UUID          `db:"id"`
	UserID       uuid.UUID          `db:"user_id"`
	Brand        string             `db:"brand"`        // free-text brand name
	DesiredValue float64            `db:"desired_value"`
	Urgency      CardRequestUrgency `db:"urgency"`
	Status       CardRequestStatus  `db:"status"`
	AdminNotes   string             `db:"admin_notes"`
	FulfilledAt  *time.Time         `db:"fulfilled_at"`
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
}
