package listing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const (
	platformDiscountPct  = 9.0  // buyer pays 9% off face value
	sellerPayoutPct      = 90.0 // seller receives 90% of face value
	platformFeePct       = 0.5  // 0.5% fee per side
	avgSellTimeMinsV1    = 45.0 // fixed V1 value
	maxListingsPerDay    = 5
)

type Service struct {
	listings     port.ListingRepository
	users        port.UserRepository
	brands       port.BrandRepository
	poolGroups   port.PoolGroupRepository
	verifyLogs   port.VerificationLogRepository
	fraudFlags   port.FraudFlagRepository
	cipher       port.CipherService
	verification port.VerificationService
	jobs         port.JobQueue
	logger       *slog.Logger
}

func NewService(
	listings port.ListingRepository,
	users port.UserRepository,
	brands port.BrandRepository,
	poolGroups port.PoolGroupRepository,
	verifyLogs port.VerificationLogRepository,
	fraudFlags port.FraudFlagRepository,
	cipher port.CipherService,
	verification port.VerificationService,
	jobs port.JobQueue,
	logger *slog.Logger,
) port.ListingService {
	return &Service{
		listings:     listings,
		users:        users,
		brands:       brands,
		poolGroups:   poolGroups,
		verifyLogs:   verifyLogs,
		fraudFlags:   fraudFlags,
		cipher:       cipher,
		verification: verification,
		jobs:         jobs,
		logger:       logger,
	}
}

// CreateListing runs Gate 1 verification, encrypts the card code, and creates the listing.
func (s *Service) CreateListing(ctx context.Context, sellerID uuid.UUID, input port.CreateListingInput) (*entity.Listing, error) {
	// 1. Check user
	user, err := s.users.FindByID(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	if user.IsBanned {
		return nil, apperror.ErrForbidden
	}

	// 2. Enforce daily listing limit
	today := time.Now().UTC().Truncate(24 * time.Hour)
	if user.ListingCountDate.Truncate(24*time.Hour).Equal(today) && user.ListingCountToday >= maxListingsPerDay {
		return nil, apperror.ErrListingLimitReached
	}

	// 3. Check brand exists and is active
	brand, err := s.brands.FindByID(ctx, input.BrandID)
	if err != nil {
		return nil, err
	}
	if brand.Status != entity.BrandStatusActive {
		return nil, apperror.New(apperror.ErrBadRequest, fmt.Sprintf("brand %q is not currently accepting listings", brand.Name))
	}
	if brand.RequiresPin && input.CardPin == "" {
		return nil, apperror.New(apperror.ErrBadRequest, fmt.Sprintf("brand %q requires a card PIN", brand.Name))
	}

	// 4. Duplicate check via code hash
	codeHash := s.cipher.Hash(input.CardCode)
	if existing, err := s.listings.FindByCodeHash(ctx, codeHash); err == nil && existing != nil {
		return nil, apperror.ErrDuplicateCard
	}

	// 5. Gate 1 — Qwikcilver verification
	result, err := s.verification.Verify(ctx, brand.Slug, input.CardCode)
	if err != nil {
		return nil, fmt.Errorf("verification service: %w", err)
	}

	// Log the verification attempt regardless of outcome
	now := time.Now().UTC()
	_ = s.verifyLogs.Create(ctx, &entity.VerificationLog{
		ListingID:    uuid.Nil, // will update after listing is created
		Gate:         1,
		Result:       resultString(result.IsValid),
		BalanceFound: result.Balance,
		FailReason:   result.FailReason,
		ResponseHash: result.ResponseHash,
	})

	if !result.IsValid {
		return nil, apperror.New(apperror.ErrVerificationFailed, result.FailReason)
	}

	// 6. Encrypt card code (and PIN if provided)
	encrypted, err := s.cipher.Encrypt(input.CardCode)
	if err != nil {
		return nil, fmt.Errorf("encrypt card code: %w", err)
	}
	var pinEncrypted string
	if input.CardPin != "" {
		pinEncrypted, err = s.cipher.Encrypt(input.CardPin)
		if err != nil {
			return nil, fmt.Errorf("encrypt card pin: %w", err)
		}
	}

	// 7. Calculate pricing
	buyerPrice, sellerPayout, discountPct := calculatePricing(input.FaceValue, input.AcceptPool, input.CustomDiscount)

	// 8. Create listing
	listing := &entity.Listing{
		ID:              uuid.New(),
		SellerID:        sellerID,
		BrandID:         input.BrandID,
		FaceValue:       input.FaceValue,
		BuyerPrice:      buyerPrice,
		SellerPayout:    sellerPayout,
		DiscountPct:     discountPct,
		IsPool:          input.AcceptPool,
		ExpiryDate:      input.ExpiryDate,
		CodeEncrypted:   encrypted,
		CodeHash:        codeHash,
		PinEncrypted:    pinEncrypted,
		Status:          entity.ListingStatusLive,
		Gate1At:         &now,
		VerifiedBalance: result.Balance,
	}

	if err := s.listings.Create(ctx, listing); err != nil {
		return nil, fmt.Errorf("create listing: %w", err)
	}

	// 9. If pool listing, upsert pool group
	if input.AcceptPool {
		pg, err := s.poolGroups.Upsert(ctx, input.BrandID, input.FaceValue, buyerPrice, sellerPayout, discountPct)
		if err != nil {
			s.logger.Warn("upsert pool group failed", "error", err)
		} else {
			_ = s.poolGroups.IncrCount(ctx, pg.ID)
		}
	}

	// 10. Increment seller's daily listing count
	_ = s.users.IncrListingCount(ctx, sellerID)

	// 11. Enqueue buy-request matching job
	_ = s.jobs.EnqueueMatchBuyRequests(ctx, listing.ID)

	return listing, nil
}

// CancelListing cancels a LIVE listing owned by sellerID.
func (s *Service) CancelListing(ctx context.Context, sellerID, listingID uuid.UUID) error {
	listing, err := s.listings.FindByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.SellerID != sellerID {
		return apperror.ErrNotListingOwner
	}
	if listing.Status != entity.ListingStatusLive {
		return apperror.New(apperror.ErrUnprocessable, "only LIVE listings can be cancelled")
	}

	if err := s.listings.UpdateStatus(ctx, listingID, entity.ListingStatusCancelled); err != nil {
		return err
	}

	// Decrement pool group count if applicable
	if listing.IsPool {
		pg, err := s.poolGroups.FindByBrandAndValue(ctx, listing.BrandID, listing.FaceValue)
		if err == nil {
			_ = s.poolGroups.DecrCount(ctx, pg.ID)
		}
	}
	return nil
}

func (s *Service) GetListing(ctx context.Context, id uuid.UUID) (*entity.Listing, error) {
	return s.listings.FindByID(ctx, id)
}

// GetMarketplace returns pool groups (top) and individual listings (below).
func (s *Service) GetMarketplace(ctx context.Context, f port.MarketplaceFilter) (*port.MarketplaceResult, error) {
	// Pool groups are fetched separately — they represent the CardSwap brand listing.
	poolGroups, err := s.poolGroups.List(ctx)
	if err != nil {
		return nil, err
	}

	// Individual listings: non-pool, LIVE
	notPool := false
	f.IsPool = &notPool
	individuals, total, err := s.listings.ListMarketplace(ctx, f)
	if err != nil {
		return nil, err
	}

	return &port.MarketplaceResult{
		PoolGroups:         poolGroups,
		IndividualListings: individuals,
		Total:              total,
	}, nil
}

func (s *Service) GetPoolGroup(ctx context.Context, id uuid.UUID) (*entity.PoolGroup, error) {
	return s.poolGroups.FindByID(ctx, id)
}

// GetRecommendedPrice returns the platform-recommended pricing breakdown for a brand+face value.
func (s *Service) GetRecommendedPrice(ctx context.Context, brandID uuid.UUID, faceValue float64) (*port.RecommendedPriceResult, error) {
	if _, err := s.brands.FindByID(ctx, brandID); err != nil {
		return nil, err
	}

	discountPct := platformDiscountPct
	sellerPrice := faceValue * (1 - discountPct/100)
	feeAmount := faceValue * (platformFeePct / 100)
	sellerPayout := sellerPrice - feeAmount
	buyerPrice := sellerPrice + feeAmount

	return &port.RecommendedPriceResult{
		RecommendedDiscountPct: discountPct,
		SellerPrice:            sellerPrice,
		SellerPayout:           sellerPayout,
		BuyerPrice:             buyerPrice,
		PlatformFeePerSide:     feeAmount,
		AvgSellTimeMins:        avgSellTimeMinsV1,
	}, nil
}

// --- helpers ---

func calculatePricing(faceValue float64, acceptPool bool, customDiscount *float64) (buyerPrice, sellerPayout, discountPct float64) {
	if acceptPool || customDiscount == nil {
		discountPct = platformDiscountPct
	} else {
		discountPct = *customDiscount
	}
	buyerPrice = faceValue * (1 - discountPct/100)
	sellerPayout = faceValue * (sellerPayoutPct / 100)
	return
}

func resultString(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

// ensure compile-time interface compliance
var _ port.ListingService = (*Service)(nil)

// suppress unused import warning for errors
var _ = errors.New
