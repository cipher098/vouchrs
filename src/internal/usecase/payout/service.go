package payout

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type Service struct {
	transactions port.TransactionRepository
	users        port.UserRepository
	payout       port.PayoutService
	logger       *slog.Logger
}

func NewService(
	transactions port.TransactionRepository,
	users port.UserRepository,
	payout port.PayoutService,
	logger *slog.Logger,
) port.PayoutUsecase {
	return &payoutUsecase{
		transactions: transactions,
		users:        users,
		payout:       payout,
		logger:       logger,
	}
}

type payoutUsecase struct {
	transactions port.TransactionRepository
	users        port.UserRepository
	payout       port.PayoutService
	logger       *slog.Logger
}

// ProcessPayout sends the seller's UPI payout for a completed transaction.
func (s *payoutUsecase) ProcessPayout(ctx context.Context, transactionID uuid.UUID) error {
	txn, err := s.transactions.FindByID(ctx, transactionID)
	if err != nil {
		return err
	}

	if txn.Status == entity.TxnStatusRefunded {
		return nil // skip if already refunded
	}

	if txn.PayoutRef != "" {
		return nil // already paid out
	}

	seller, err := s.users.FindByID(ctx, txn.SellerID)
	if err != nil {
		return err
	}

	if seller.UPIID == "" {
		return apperror.New(apperror.ErrBadRequest, "seller has no UPI ID configured")
	}

	result, err := s.payout.CreatePayout(ctx, port.CreatePayoutInput{
		UPIID:       seller.UPIID,
		Amount:      txn.SellerPayout,
		Purpose:     "payout",
		ReferenceID: "CS_PAYOUT_" + txn.ID.String()[:8],
		Narration:   fmt.Sprintf("CardSwap sale payout — txn %s", txn.ID),
	})
	if err != nil {
		s.logger.Error("razorpay payout failed", "txn_id", txn.ID, "seller_id", seller.ID, "error", err)
		return fmt.Errorf("create payout: %w", err)
	}

	if err := s.transactions.SetPayoutRef(ctx, txn.ID, result.PayoutID); err != nil {
		return err
	}

	s.logger.Info("payout queued", "txn_id", txn.ID, "payout_id", result.PayoutID, "amount", txn.SellerPayout)
	return nil
}

var _ port.PayoutUsecase = (*payoutUsecase)(nil)

// Silence unused variable
var _ = &Service{}
