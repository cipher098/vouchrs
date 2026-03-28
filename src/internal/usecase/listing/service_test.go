package listing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/internal/usecase/listing"
	"github.com/gothi/vouchrs/src/pkg/testmock"
)

type listingDeps struct {
	svc        port.ListingService
	listings   *testmock.ListingRepo
	users      *testmock.UserRepo
	brands     *testmock.BrandRepo
	verifyLog  *testmock.VerifyLogRepo
	fraudFlags *testmock.FraudFlagRepo
	verifySvc  *testmock.VerificationSvc
	cipher     *testmock.CipherSvc
	jobs       *testmock.JobQueue
}

func buildListingService(t *testing.T) listingDeps {
	t.Helper()
	d := listingDeps{
		listings:   &testmock.ListingRepo{},
		users:      &testmock.UserRepo{},
		brands:     &testmock.BrandRepo{},
		verifyLog:  &testmock.VerifyLogRepo{},
		fraudFlags: &testmock.FraudFlagRepo{},
		cipher:     &testmock.CipherSvc{},
		verifySvc:  &testmock.VerificationSvc{},
		jobs:       &testmock.JobQueue{},
	}
	poolRepo := &testmock.PoolGroupRepo{}
	d.svc = listing.NewService(d.listings, d.users, d.brands, poolRepo, d.verifyLog, d.fraudFlags, d.cipher, d.verifySvc, d.jobs, nil)
	return d
}

func TestCreateListing_Gate1Fail(t *testing.T) {
	d := buildListingService(t)
	ctx := context.Background()
	sellerID := uuid.New()
	brandID := uuid.New()

	d.users.On("FindByID", ctx, sellerID).Return(&entity.User{
		ID: sellerID, Role: entity.UserRoleSeller,
		ListingCountToday: 0,
	}, nil)
	d.brands.On("FindByID", ctx, brandID).Return(&entity.Brand{
		ID: brandID, Slug: "amazon", Status: entity.BrandStatusActive,
	}, nil)
	d.cipher.On("Hash", "INVALID-CODE").Return("abc123hash")
	d.listings.On("FindByCodeHash", ctx, "abc123hash").Return(nil, apperror.ErrNotFound)
	d.verifySvc.On("Verify", ctx, "amazon", "INVALID-CODE").Return(&port.VerificationResult{
		IsValid:    false,
		FailReason: "card has expired",
	}, nil)
	d.verifyLog.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := d.svc.CreateListing(ctx, sellerID, port.CreateListingInput{
		BrandID:    brandID,
		FaceValue:  1000,
		CardCode:   "INVALID-CODE",
		AcceptPool: true,
	})

	assert.ErrorIs(t, err, apperror.ErrVerificationFailed)
}

func TestCreateListing_DuplicateCard(t *testing.T) {
	d := buildListingService(t)
	ctx := context.Background()
	sellerID := uuid.New()
	brandID := uuid.New()

	d.users.On("FindByID", ctx, sellerID).Return(&entity.User{ID: sellerID, Role: entity.UserRoleSeller}, nil)
	d.brands.On("FindByID", ctx, brandID).Return(&entity.Brand{ID: brandID, Slug: "amazon", Status: entity.BrandStatusActive}, nil)
	d.cipher.On("Hash", "DUPE-CODE").Return("dupehash")
	d.listings.On("FindByCodeHash", ctx, "dupehash").Return(&entity.Listing{ID: uuid.New()}, nil)

	_, err := d.svc.CreateListing(ctx, sellerID, port.CreateListingInput{
		BrandID: brandID, FaceValue: 500, CardCode: "DUPE-CODE", AcceptPool: true,
	})
	assert.ErrorIs(t, err, apperror.ErrDuplicateCard)
}

func TestCreateListing_DailyLimitReached(t *testing.T) {
	d := buildListingService(t)
	ctx := context.Background()
	sellerID := uuid.New()
	brandID := uuid.New()

	d.users.On("FindByID", ctx, sellerID).Return(&entity.User{
		ID:                sellerID,
		ListingCountToday: 5, // at limit
		ListingCountDate:  time.Now().UTC(),
	}, nil)

	_, err := d.svc.CreateListing(ctx, sellerID, port.CreateListingInput{
		BrandID: brandID, FaceValue: 1000, CardCode: "SOME-CODE", AcceptPool: true,
	})
	assert.ErrorIs(t, err, apperror.ErrListingLimitReached)
}

func TestCancelListing_NotOwner(t *testing.T) {
	d := buildListingService(t)
	ctx := context.Background()
	listingID := uuid.New()
	ownerID := uuid.New()
	otherUser := uuid.New()

	d.listings.On("FindByID", ctx, listingID).Return(&entity.Listing{
		ID: listingID, SellerID: ownerID, Status: entity.ListingStatusLive,
	}, nil)

	err := d.svc.CancelListing(ctx, otherUser, listingID)
	assert.ErrorIs(t, err, apperror.ErrNotListingOwner)
}
