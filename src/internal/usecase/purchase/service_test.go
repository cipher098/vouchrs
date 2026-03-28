package purchase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/internal/usecase/purchase"
	"github.com/gothi/vouchrs/src/pkg/testmock"
)

type purchaseDeps struct {
	svc          port.PurchaseService
	listings     *testmock.ListingRepo
	transactions *testmock.TransactionRepo
	users        *testmock.UserRepo
	brands       *testmock.BrandRepo
	poolGroups   *testmock.PoolGroupRepo
	verifyLog    *testmock.VerifyLogRepo
	fraudFlags   *testmock.FraudFlagRepo
	verifySvc    *testmock.VerificationSvc
	payment      *testmock.PaymentGW
	cipher       *testmock.CipherSvc
	email        *testmock.EmailSvc
	jobs         *testmock.JobQueue
}

func buildPurchaseService(t *testing.T) purchaseDeps {
	t.Helper()
	d := purchaseDeps{
		listings:     &testmock.ListingRepo{},
		transactions: &testmock.TransactionRepo{},
		users:        &testmock.UserRepo{},
		brands:       &testmock.BrandRepo{},
		poolGroups:   &testmock.PoolGroupRepo{},
		verifyLog:    &testmock.VerifyLogRepo{},
		fraudFlags:   &testmock.FraudFlagRepo{},
		verifySvc:    &testmock.VerificationSvc{},
		payment:      &testmock.PaymentGW{},
		cipher:       &testmock.CipherSvc{},
		email:        &testmock.EmailSvc{},
		jobs:         &testmock.JobQueue{},
	}
	d.svc = purchase.NewService(
		d.listings, d.transactions, d.users, d.brands, d.poolGroups,
		d.verifyLog, d.fraudFlags, d.verifySvc, d.payment, d.cipher, d.email, d.jobs,
		"https://callback.example.com", "https://redirect.example.com", nil,
	)
	return d
}

func TestInitiateBuy_BannedBuyer(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	buyerID := uuid.New()
	listingID := uuid.New()

	d.users.On("FindByID", ctx, buyerID).Return(&entity.User{ID: buyerID, IsBanned: true}, nil)

	_, err := d.svc.InitiateBuy(ctx, buyerID, listingID)
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestInitiateBuy_Gate2Fail_SetsfraudHold(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	buyerID := uuid.New()
	sellerID := uuid.New()
	brandID := uuid.New()
	listingID := uuid.New()

	d.users.On("FindByID", ctx, buyerID).Return(&entity.User{ID: buyerID}, nil)
	d.listings.On("FindByID", ctx, listingID).Return(&entity.Listing{
		ID:            listingID,
		SellerID:      sellerID,
		BrandID:       brandID,
		Status:        entity.ListingStatusLive,
		FaceValue:     1000,
		CodeEncrypted: "enc-code",
	}, nil)
	d.brands.On("FindByID", ctx, brandID).Return(&entity.Brand{
		ID: brandID, Slug: "amazon",
	}, nil)
	d.cipher.On("Decrypt", "enc-code").Return("PLAIN-CODE", nil)
	d.verifySvc.On("Verify", ctx, "amazon", "PLAIN-CODE").Return(&port.VerificationResult{
		IsValid:    false,
		FailReason: "card drained",
	}, nil)
	d.verifyLog.On("Create", mock.Anything, mock.Anything).Return(nil)
	d.listings.On("UpdateStatus", ctx, listingID, entity.ListingStatusFraudHold).Return(nil)
	d.fraudFlags.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := d.svc.InitiateBuy(ctx, buyerID, listingID)
	assert.ErrorIs(t, err, apperror.ErrCardTampered)
	d.listings.AssertCalled(t, "UpdateStatus", ctx, listingID, entity.ListingStatusFraudHold)
}

func TestInitiateBuy_SellerCannotBuyOwn(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	sellerID := uuid.New()
	brandID := uuid.New()
	listingID := uuid.New()

	d.users.On("FindByID", ctx, sellerID).Return(&entity.User{ID: sellerID}, nil)
	d.listings.On("FindByID", ctx, listingID).Return(&entity.Listing{
		ID:       listingID,
		SellerID: sellerID, // same as buyer
		BrandID:  brandID,
		Status:   entity.ListingStatusLive,
	}, nil)

	_, err := d.svc.InitiateBuy(ctx, sellerID, listingID)
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestHandlePaymentSuccess_EmailsCodeAndQueuesJob(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	txnID := uuid.New()
	listingID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	brandID := uuid.New()
	payRef := "CS-test-ref"

	txn := &entity.Transaction{
		ID:          txnID,
		ListingID:   listingID,
		BuyerID:     buyerID,
		SellerID:    sellerID,
		BuyerAmount: 910,
		Status:      entity.TxnStatusPending,
		PaymentRef:  payRef,
	}
	listing := &entity.Listing{
		ID:            listingID,
		SellerID:      sellerID,
		BrandID:       brandID,
		FaceValue:     1000,
		CodeEncrypted: "enc-code",
		IsPool:        false,
	}
	buyer := &entity.User{ID: buyerID, Email: "buyer@example.com"}
	brand := &entity.Brand{ID: brandID, Name: "Amazon India"}

	d.transactions.On("FindByPaymentRef", ctx, payRef).Return(txn, nil)
	d.listings.On("FindByID", ctx, listingID).Return(listing, nil)
	d.users.On("FindByID", ctx, buyerID).Return(buyer, nil)
	d.brands.On("FindByID", ctx, brandID).Return(brand, nil)
	d.listings.On("MarkSold", ctx, listingID).Return(nil)
	d.transactions.On("SetPaidAt", ctx, txnID).Return(nil)
	d.jobs.On("CancelLockExpiry", ctx, listingID).Return(nil)
	d.cipher.On("Decrypt", "enc-code").Return("PLAIN-1234", nil)
	d.email.On("SendCardCode", ctx, "buyer@example.com", "Amazon India", float64(1000), "PLAIN-1234").Return(nil)
	d.transactions.On("SetCodeRevealedAt", ctx, txnID).Return(nil)
	d.jobs.On("EnqueuePayout", ctx, txnID).Return(nil)

	err := d.svc.HandlePaymentSuccess(ctx, payRef)
	assert.NoError(t, err)
	d.email.AssertCalled(t, "SendCardCode", ctx, "buyer@example.com", "Amazon India", float64(1000), "PLAIN-1234")
	d.jobs.AssertCalled(t, "EnqueuePayout", ctx, txnID)
}

func TestHandlePaymentFailure_UnlocksListing(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	txnID := uuid.New()
	listingID := uuid.New()
	payRef := "CS-fail-ref"

	txn := &entity.Transaction{
		ID:         txnID,
		ListingID:  listingID,
		Status:     entity.TxnStatusPending,
		PaymentRef: payRef,
	}

	d.transactions.On("FindByPaymentRef", ctx, payRef).Return(txn, nil)
	d.transactions.On("UpdateStatus", ctx, txnID, entity.TxnStatusCancelled).Return(nil)
	d.listings.On("Unlock", ctx, listingID).Return(nil)

	err := d.svc.HandlePaymentFailure(ctx, payRef)
	assert.NoError(t, err)
	d.listings.AssertCalled(t, "Unlock", ctx, listingID)
}

func TestConfirmRedemption_WrongBuyer(t *testing.T) {
	d := buildPurchaseService(t)
	ctx := context.Background()
	txnID := uuid.New()
	realBuyer := uuid.New()
	otherUser := uuid.New()

	d.transactions.On("FindByID", ctx, txnID).Return(&entity.Transaction{
		ID:      txnID,
		BuyerID: realBuyer,
		Status:  entity.TxnStatusPaid,
	}, nil)

	err := d.svc.ConfirmRedemption(ctx, otherUser, txnID)
	assert.ErrorIs(t, err, apperror.ErrNotTransactionParty)
}
