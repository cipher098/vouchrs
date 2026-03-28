package testmock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

// --- ListingRepo ---

type ListingRepo struct{ mock.Mock }

func (m *ListingRepo) Create(ctx context.Context, l *entity.Listing) error {
	return m.Called(ctx, l).Error(0)
}
func (m *ListingRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Listing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Listing), args.Error(1)
}
func (m *ListingRepo) FindByCodeHash(ctx context.Context, hash string) (*entity.Listing, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Listing), args.Error(1)
}
func (m *ListingRepo) ListMarketplace(ctx context.Context, f port.MarketplaceFilter) ([]*entity.Listing, int, error) {
	args := m.Called(ctx, f)
	return args.Get(0).([]*entity.Listing), args.Int(1), args.Error(2)
}
func (m *ListingRepo) ListBySeller(ctx context.Context, id uuid.UUID, p pagination.Params) ([]*entity.Listing, int, error) {
	args := m.Called(ctx, id, p)
	return args.Get(0).([]*entity.Listing), args.Int(1), args.Error(2)
}
func (m *ListingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status entity.ListingStatus) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *ListingRepo) Lock(ctx context.Context, id uuid.UUID, buyerID uuid.UUID, exp time.Time) error {
	return m.Called(ctx, id, buyerID, exp).Error(0)
}
func (m *ListingRepo) Unlock(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *ListingRepo) MarkSold(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *ListingRepo) OldestLiveInPool(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.Listing, error) {
	args := m.Called(ctx, brandID, faceValue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Listing), args.Error(1)
}
func (m *ListingRepo) FindExpiredLocks(ctx context.Context) ([]*entity.Listing, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.Listing), args.Error(1)
}
func (m *ListingRepo) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Listing, int, error) {
	args := m.Called(ctx, p)
	return args.Get(0).([]*entity.Listing), args.Int(1), args.Error(2)
}

// --- BrandRepo ---

type BrandRepo struct{ mock.Mock }

func (m *BrandRepo) Create(ctx context.Context, b *entity.Brand) error {
	return m.Called(ctx, b).Error(0)
}
func (m *BrandRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Brand, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Brand), args.Error(1)
}
func (m *BrandRepo) FindBySlug(ctx context.Context, slug string) (*entity.Brand, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Brand), args.Error(1)
}
func (m *BrandRepo) ListActive(ctx context.Context) ([]*entity.Brand, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.Brand), args.Error(1)
}
func (m *BrandRepo) ListWithCount(ctx context.Context) ([]*entity.Brand, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.Brand), args.Error(1)
}
func (m *BrandRepo) Update(ctx context.Context, b *entity.Brand) error {
	return m.Called(ctx, b).Error(0)
}

// --- PoolGroupRepo ---

type PoolGroupRepo struct{ mock.Mock }

func (m *PoolGroupRepo) Upsert(ctx context.Context, brandID uuid.UUID, faceValue, buyerPrice, sellerPayout, discountPct float64) (*entity.PoolGroup, error) {
	args := m.Called(ctx, brandID, faceValue, buyerPrice, sellerPayout, discountPct)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PoolGroup), args.Error(1)
}
func (m *PoolGroupRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.PoolGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PoolGroup), args.Error(1)
}
func (m *PoolGroupRepo) FindByBrandAndValue(ctx context.Context, brandID uuid.UUID, faceValue float64) (*entity.PoolGroup, error) {
	args := m.Called(ctx, brandID, faceValue)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PoolGroup), args.Error(1)
}
func (m *PoolGroupRepo) IncrCount(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *PoolGroupRepo) DecrCount(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *PoolGroupRepo) List(ctx context.Context) ([]*entity.PoolGroup, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.PoolGroup), args.Error(1)
}

// --- VerifyLogRepo ---

type VerifyLogRepo struct{ mock.Mock }

func (m *VerifyLogRepo) Create(ctx context.Context, l *entity.VerificationLog) error {
	return m.Called(ctx, l).Error(0)
}
func (m *VerifyLogRepo) FindLatest(ctx context.Context, listingID uuid.UUID, gate int) (*entity.VerificationLog, error) {
	args := m.Called(ctx, listingID, gate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.VerificationLog), args.Error(1)
}

// --- FraudFlagRepo ---

type FraudFlagRepo struct{ mock.Mock }

func (m *FraudFlagRepo) Create(ctx context.Context, f *entity.FraudFlag) error {
	return m.Called(ctx, f).Error(0)
}
func (m *FraudFlagRepo) FindByUser(ctx context.Context, id uuid.UUID) ([]*entity.FraudFlag, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*entity.FraudFlag), args.Error(1)
}
func (m *FraudFlagRepo) FindByListing(ctx context.Context, id uuid.UUID) ([]*entity.FraudFlag, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]*entity.FraudFlag), args.Error(1)
}
func (m *FraudFlagRepo) ListUnresolved(ctx context.Context) ([]*entity.FraudFlag, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.FraudFlag), args.Error(1)
}
func (m *FraudFlagRepo) Resolve(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- TransactionRepo ---

type TransactionRepo struct{ mock.Mock }

func (m *TransactionRepo) Create(ctx context.Context, t *entity.Transaction) error {
	return m.Called(ctx, t).Error(0)
}
func (m *TransactionRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Transaction), args.Error(1)
}
func (m *TransactionRepo) FindByListingID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Transaction), args.Error(1)
}
func (m *TransactionRepo) FindByPaymentRef(ctx context.Context, ref string) (*entity.Transaction, error) {
	args := m.Called(ctx, ref)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Transaction), args.Error(1)
}
func (m *TransactionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, s entity.TransactionStatus) error {
	return m.Called(ctx, id, s).Error(0)
}
func (m *TransactionRepo) SetPaymentRef(ctx context.Context, id uuid.UUID, ref string) error {
	return m.Called(ctx, id, ref).Error(0)
}
func (m *TransactionRepo) SetPayoutRef(ctx context.Context, id uuid.UUID, ref string) error {
	return m.Called(ctx, id, ref).Error(0)
}
func (m *TransactionRepo) SetPaidAt(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *TransactionRepo) SetCodeRevealedAt(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *TransactionRepo) SetCompletedAt(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *TransactionRepo) ListByBuyer(ctx context.Context, id uuid.UUID, p pagination.Params) ([]*entity.Transaction, int, error) {
	args := m.Called(ctx, id, p)
	return args.Get(0).([]*entity.Transaction), args.Int(1), args.Error(2)
}
func (m *TransactionRepo) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Transaction, int, error) {
	args := m.Called(ctx, p)
	return args.Get(0).([]*entity.Transaction), args.Int(1), args.Error(2)
}

// Compile-time interface checks
var _ port.ListingRepository = (*ListingRepo)(nil)
var _ port.BrandRepository = (*BrandRepo)(nil)
var _ port.PoolGroupRepository = (*PoolGroupRepo)(nil)
var _ port.VerificationLogRepository = (*VerifyLogRepo)(nil)
var _ port.FraudFlagRepository = (*FraudFlagRepo)(nil)
var _ port.TransactionRepository = (*TransactionRepo)(nil)
