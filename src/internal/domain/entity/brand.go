package entity

import (
	"time"

	"github.com/google/uuid"
)

type BrandStatus string

const (
	BrandStatusActive   BrandStatus = "active"
	BrandStatusTesting  BrandStatus = "testing"
	BrandStatusInactive BrandStatus = "inactive"
)

type Brand struct {
	ID                 uuid.UUID   `db:"id"                  json:"id"`
	Name               string      `db:"name"                json:"name"`
	Slug               string      `db:"slug"                json:"slug"`
	LogoURL            string      `db:"logo_url"            json:"logo_url"`
	Color              string      `db:"color"               json:"color"`
	VerificationSource string      `db:"verification_source" json:"verification_source"`
	Status             BrandStatus `db:"status"              json:"status"`
	// ListingCount is populated by ListWithCount — not a DB column.
	ListingCount int       `db:"-"          json:"listing_count,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}
