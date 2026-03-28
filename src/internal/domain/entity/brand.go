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
	ID                 uuid.UUID   `db:"id"`
	Name               string      `db:"name"`
	Slug               string      `db:"slug"`
	LogoURL            string      `db:"logo_url"`
	VerificationSource string      `db:"verification_source"` // e.g. "qwikcilver", "manual"
	Status             BrandStatus `db:"status"`
	CreatedAt          time.Time   `db:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at"`
}
