package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

// --- User ---

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByPhone(ctx context.Context, phone string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	// IncrListingCount increments today's listing count, resetting if the date has changed.
	IncrListingCount(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, p pagination.Params) ([]*entity.User, int, error)
	Ban(ctx context.Context, id uuid.UUID) error
}

// --- Brand ---

type BrandRepository interface {
	Create(ctx context.Context, brand *entity.Brand) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Brand, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Brand, error)
	ListActive(ctx context.Context) ([]*entity.Brand, error)
	Update(ctx context.Context, brand *entity.Brand) error
}

// --- Listing ---

type MarketplaceFilter struct {
	BrandID      *uuid.UUID
	MinValue     *float64
	MaxValue     *float64
	MinDiscount  *float64
	MaxDiscount  *float64
	IsPool       *bool // nil = both, true = pool only, false = individual only
	SortBy       string // "discount_desc", "price_asc", "value_desc", "newest"
	Pagination   pagination.Params
}

type ListingRepository interface {
	Create(ctx context.Context, listing *entity.Listing) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Listing, error)
	FindByCodeHash(ctx context.Context, hash string) (*entity.Listing, error)
	ListMarketplace(ctx context.Context, f MarketplaceFilter) ([]*entity.Listing, int, error)
	ListBySeller(ctx context.Context, sellerID uuid.UUID, p pagination.Params) ([]*entity.Listing, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ListingStatus) error
	// Lock atomically sets status=LOCKED, lock_buyer_id, lock_expires_at.
	// Returns ErrListingNotLive if the listing is not LIVE.
	Lock(ctx context.Context, id uuid.UUID, buyerID uuid.UUID, expiresAt time.Time) error
	// Unlock sets status=LIVE and clears lock fields.
	Unlock(ctx context.Context, id uuid.UUID) error
	MarkSold(ctx context.Context, id uuid.UUID) error
	// OldestLiveInPool returns the oldest LIVE listing in the CardSwap pool
	// for the given brand + face value (FIFO selection).
	OldestLiveInPool(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.Listing, error)
	// FindExpiredLocks returns all LOCKED listings whose lock_expires_at is in the past.
	FindExpiredLocks(ctx context.Context) ([]*entity.Listing, error)
	ListAll(ctx context.Context, p pagination.Params) ([]*entity.Listing, int, error)
}

// --- PoolGroup ---

type PoolGroupRepository interface {
	// Upsert creates or returns the pool group for a brand+value combination.
	Upsert(ctx context.Context, brandID uuid.UUID, faceValue, buyerPrice, sellerPayout, discountPct float64) (*entity.PoolGroup, error)
	FindByBrandAndValue(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.PoolGroup, error)
	IncrCount(ctx context.Context, id uuid.UUID) error
	DecrCount(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]*entity.PoolGroup, error)
}

// --- Transaction ---

type TransactionRepository interface {
	Create(ctx context.Context, txn *entity.Transaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)
	FindByListingID(ctx context.Context, listingID uuid.UUID) (*entity.Transaction, error)
	FindByPaymentRef(ctx context.Context, paymentRef string) (*entity.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.TransactionStatus) error
	SetPaymentRef(ctx context.Context, id uuid.UUID, ref string) error
	SetPayoutRef(ctx context.Context, id uuid.UUID, ref string) error
	SetPaidAt(ctx context.Context, id uuid.UUID) error
	SetCodeRevealedAt(ctx context.Context, id uuid.UUID) error
	SetCompletedAt(ctx context.Context, id uuid.UUID) error
	ListByBuyer(ctx context.Context, buyerID uuid.UUID, p pagination.Params) ([]*entity.Transaction, int, error)
	ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int, error)
}

// --- BuyRequest ---

type BuyRequestRepository interface {
	Create(ctx context.Context, req *entity.BuyRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.BuyRequest, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.BuyRequest, error)
	// FindMatchingForListing returns active buy requests that match the listing's brand + value.
	FindMatchingForListing(ctx context.Context, listing *entity.Listing) ([]*entity.BuyRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.BuyRequestStatus) error
	IncrAlertCount(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// --- CardRequest ---

type CardRequestRepository interface {
	Create(ctx context.Context, req *entity.CardRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.CardRequest, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.CardRequest, error)
	ListPending(ctx context.Context) ([]*entity.CardRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.CardRequestStatus, adminNotes string) error
}

// --- VerificationLog ---

type VerificationLogRepository interface {
	Create(ctx context.Context, log *entity.VerificationLog) error
	FindLatest(ctx context.Context, listingID uuid.UUID, gate int) (*entity.VerificationLog, error)
}

// --- FraudFlag ---

type FraudFlagRepository interface {
	Create(ctx context.Context, flag *entity.FraudFlag) error
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*entity.FraudFlag, error)
	FindByListing(ctx context.Context, listingID uuid.UUID) ([]*entity.FraudFlag, error)
	ListUnresolved(ctx context.Context) ([]*entity.FraudFlag, error)
	Resolve(ctx context.Context, id uuid.UUID) error
}
