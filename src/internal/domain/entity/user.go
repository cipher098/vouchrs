package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleBuyer  UserRole = "buyer"
	UserRoleSeller UserRole = "seller"
	UserRoleAdmin  UserRole = "admin"
)

type User struct {
	ID                uuid.UUID `db:"id"`
	Phone             string    `db:"phone"`
	Email             string    `db:"email"`
	FullName          string    `db:"full_name"`
	Role              UserRole  `db:"role"`
	IsVerified        bool      `db:"is_verified"`
	IsBanned          bool      `db:"is_banned"`
	IsFlagged         bool      `db:"is_flagged"`
	ListingCountToday int       `db:"listing_count_today"`
	ListingCountDate  time.Time `db:"listing_count_date"`
	UPIID             string    `db:"upi_id"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
