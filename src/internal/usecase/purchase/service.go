package purchase

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
	phonepepkg "github.com/gothi/vouchrs/src/external/payment/phonepe"
)

const lockDuration = 10 * time.Minute

type Service struct {
	listings     port.ListingRepository
	transactions port.TransactionRepository
	users        port.UserRepository
	brands       port.BrandRepository
	poolGroups   port.PoolGroupRepository
	verifyLogs   port.VerificationLogRepository
	fraudFlags   port.FraudFlagRepository
	verification port.VerificationService
	payment      port.PaymentGateway
	cipher       port.CipherService
	email        port.EmailService
	jobs         port.JobQueue
	callbackURL  string
	redirectURL  string
	logger       *slog.Logger
}

func NewService(
	listings port.ListingRepository,
	transactions port.TransactionRepository,
	users port.UserRepository,
	brands port.BrandRepository,
	poolGroups port.PoolGroupRepository,
	verifyLogs port.VerificationLogRepository,
	fraudFlags port.FraudFlagRepository,
	verification port.VerificationService,
	payment port.PaymentGateway,
	cipher port.CipherService,
	email port.EmailService,
	jobs port.JobQueue,
	callbackURL, redirectURL string,
	logger *slog.Logger,
) port.PurchaseService {
	return &Service{
		listings:     listings,
		transactions: transactions,
		users:        users,
		brands:       brands,
		poolGroups:   poolGroups,
		verifyLogs:   verifyLogs,
		fraudFlags:   fraudFlags,
		verification: verification,
		payment:      payment,
		cipher:       cipher,
		email:        email,
		jobs:         jobs,
		callbackURL:  callbackURL,
		redirectURL:  redirectURL,
		logger:       logger,
	}
}

// InitiateBuy runs Gate 2 verification, locks the listing, creates a transaction,
// and returns a PhonePe payment URL.
func (s *Service) InitiateBuy(ctx context.Context, buyerID, listingID uuid.UUID) (*port.InitiateBuyResult, error) {
	buyer, err := s.users.FindByID(ctx, buyerID)
	if err != nil {
		return nil, err
	}
	if buyer.IsBanned {
		return nil, apperror.ErrForbidden
	}

	listing, err := s.listings.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}

	// If this is a pool listing, pick the actual oldest LIVE card (FIFO)
	if listing.IsPool {
		listing, err = s.listings.OldestLiveInPool(ctx, listing.BrandID, listing.FaceValue)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return nil, apperror.New(apperror.ErrListingNotLive, "no cards available in pool right now")
			}
			return nil, err
		}
	}

	if !listing.IsAvailable() {
		if listing.Status == entity.ListingStatusLocked {
			return nil, apperror.ErrListingLocked
		}
		return nil, apperror.ErrListingNotLive
	}

	// Prevent seller from buying their own listing
	if listing.SellerID == buyerID {
		return nil, apperror.New(apperror.ErrForbidden, "you cannot purchase your own listing")
	}

	// Gate 2: re-verify card before locking
	brand, err := s.brands.FindByID(ctx, listing.BrandID)
	if err != nil {
		return nil, err
	}

	// Decrypt to get plain code for re-verification
	plainCode, err := s.cipher.Decrypt(listing.CodeEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt card code for gate 2: %w", err)
	}

	verifyResult, verifyErr := s.verification.Verify(ctx, brand.Slug, plainCode)

	// Log gate 2 regardless of outcome
	_ = s.verifyLogs.Create(ctx, &entity.VerificationLog{
		ListingID:    listing.ID,
		Gate:         2,
		Result:       resultStr(verifyResult != nil && verifyResult.IsValid),
		BalanceFound: func() float64 {
			if verifyResult != nil {
				return verifyResult.Balance
			}
			return 0
		}(),
		FailReason:   func() string {
			if verifyResult != nil {
				return verifyResult.FailReason
			}
			if verifyErr != nil {
				return verifyErr.Error()
			}
			return ""
		}(),
		ResponseHash: func() string {
			if verifyResult != nil {
				return verifyResult.ResponseHash
			}
			return ""
		}(),
	})

	if verifyErr != nil || verifyResult == nil || !verifyResult.IsValid {
		// Card was tampered — set FRAUD_HOLD and flag the seller
		_ = s.listings.UpdateStatus(ctx, listing.ID, entity.ListingStatusFraudHold)
		_ = s.fraudFlags.Create(ctx, &entity.FraudFlag{
			UserID:    listing.SellerID,
			ListingID: &listing.ID,
			Reason:    "Gate 2 failed: " + func() string {
				if verifyResult != nil {
					return verifyResult.FailReason
				}
				return "verification error"
			}(),
			Severity: entity.FraudSeverityHigh,
		})
		return nil, apperror.ErrCardTampered
	}

	// Lock the listing
	lockExpiresAt := time.Now().UTC().Add(lockDuration)
	if err := s.listings.Lock(ctx, listing.ID, buyerID, lockExpiresAt); err != nil {
		return nil, err
	}

	// Create transaction in pending state
	merchantTxnID := phonepepkg.MerchantTransactionID()
	now := time.Now().UTC()
	txn := &entity.Transaction{
		ListingID:     listing.ID,
		BuyerID:       buyerID,
		SellerID:      listing.SellerID,
		BuyerAmount:   listing.BuyerPrice,
		SellerPayout:  listing.SellerPayout,
		PaymentRef:    merchantTxnID,
		Status:        entity.TxnStatusPending,
		LockStartedAt: &now,
	}
	if err := s.transactions.Create(ctx, txn); err != nil {
		// Rollback lock on failure
		_ = s.listings.Unlock(ctx, listing.ID)
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// Enqueue lock expiry job (auto-unlock after 10 min if no payment)
	_ = s.jobs.EnqueueLockExpiry(ctx, listing.ID, lockDuration+5*time.Second)

	// Create PhonePe payment order
	pgOrder, err := s.payment.CreateOrder(ctx, port.PaymentOrderInput{
		MerchantTransactionID: merchantTxnID,
		Amount:                listing.BuyerPrice,
		UserID:                buyerID,
		RedirectURL:           s.redirectURL,
		CallbackURL:           s.callbackURL,
	})
	if err != nil {
		s.logger.Error("create payment order failed", "txn_id", txn.ID, "error", err)
		// Don't fail the lock — buyer can retry or it will auto-expire
		return nil, fmt.Errorf("create payment order: %w", err)
	}

	return &port.InitiateBuyResult{
		Transaction:   txn,
		PaymentURL:    pgOrder.PaymentURL,
		LockExpiresAt: lockExpiresAt.Format(time.RFC3339),
	}, nil
}

// HandlePaymentSuccess is called by the webhook on successful payment.
func (s *Service) HandlePaymentSuccess(ctx context.Context, merchantTransactionID string) error {
	txn, err := s.transactions.FindByPaymentRef(ctx, merchantTransactionID)
	if err != nil {
		return err
	}
	if txn.Status != entity.TxnStatusPending {
		return nil // idempotent
	}

	listing, err := s.listings.FindByID(ctx, txn.ListingID)
	if err != nil {
		return err
	}
	buyer, err := s.users.FindByID(ctx, txn.BuyerID)
	if err != nil {
		return err
	}
	brand, err := s.brands.FindByID(ctx, listing.BrandID)
	if err != nil {
		return err
	}

	// Mark listing as sold
	if err := s.listings.MarkSold(ctx, listing.ID); err != nil {
		return fmt.Errorf("mark listing sold: %w", err)
	}

	// Decrement pool group if applicable
	if listing.IsPool {
		pg, err := s.poolGroups.FindByBrandAndValue(ctx, listing.BrandID, listing.FaceValue)
		if err == nil {
			_ = s.poolGroups.DecrCount(ctx, pg.ID)
		}
	}

	// Mark transaction as paid
	if err := s.transactions.SetPaidAt(ctx, txn.ID); err != nil {
		return err
	}

	// Cancel the lock expiry job — payment succeeded
	_ = s.jobs.CancelLockExpiry(ctx, listing.ID)

	// Decrypt card code and send via email (NEVER return to API)
	plainCode, err := s.cipher.Decrypt(listing.CodeEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt card code: %w", err)
	}

	if buyer.Email != "" {
		if err := s.email.SendCardCode(ctx, buyer.Email, brand.Name, listing.FaceValue, plainCode); err != nil {
			s.logger.Error("send card code email failed", "txn_id", txn.ID, "error", err)
			// Don't fail — log and retry manually
		}
	}

	// Record that code was sent
	_ = s.transactions.SetCodeRevealedAt(ctx, txn.ID)

	// Queue seller payout
	_ = s.jobs.EnqueuePayout(ctx, txn.ID)

	return nil
}

// HandlePaymentFailure unlocks the listing so it becomes available again.
func (s *Service) HandlePaymentFailure(ctx context.Context, merchantTransactionID string) error {
	txn, err := s.transactions.FindByPaymentRef(ctx, merchantTransactionID)
	if err != nil {
		return err
	}
	if txn.Status != entity.TxnStatusPending {
		return nil
	}

	if err := s.transactions.UpdateStatus(ctx, txn.ID, entity.TxnStatusCancelled); err != nil {
		return err
	}
	return s.listings.Unlock(ctx, txn.ListingID)
}

// ConfirmRedemption marks the transaction as completed.
func (s *Service) ConfirmRedemption(ctx context.Context, buyerID, transactionID uuid.UUID) error {
	txn, err := s.transactions.FindByID(ctx, transactionID)
	if err != nil {
		return err
	}
	if txn.BuyerID != buyerID {
		return apperror.ErrNotTransactionParty
	}
	if txn.Status != entity.TxnStatusPaid {
		return apperror.ErrAlreadyCompleted
	}
	return s.transactions.SetCompletedAt(ctx, txn.ID)
}

func (s *Service) GetTransaction(ctx context.Context, userID, transactionID uuid.UUID) (*entity.Transaction, error) {
	txn, err := s.transactions.FindByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if txn.BuyerID != userID && txn.SellerID != userID {
		return nil, apperror.ErrNotTransactionParty
	}
	return txn, nil
}

func resultStr(pass bool) string {
	if pass {
		return "pass"
	}
	return "fail"
}

var _ port.PurchaseService = (*Service)(nil)
