// Package worker runs background job handlers using asynq.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/gothi/vouchrs/src/external/queue"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type Worker struct {
	server   *asynq.Server
	requests port.RequestService
	purchase port.PurchaseService
	payout   port.PayoutUsecase
	listings port.ListingRepository
	logger   *slog.Logger
}

func New(
	redisAddr, redisPassword string,
	redisDB, concurrency int,
	requests port.RequestService,
	purchase port.PurchaseService,
	payout port.PayoutUsecase,
	listings port.ListingRepository,
	logger *slog.Logger,
) *Worker {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: redisDB},
		asynq.Config{Concurrency: concurrency},
	)
	return &Worker{
		server:   srv,
		requests: requests,
		purchase: purchase,
		payout:   payout,
		listings: listings,
		logger:   logger,
	}
}

func (w *Worker) Start() error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskLockExpiry, w.handleLockExpiry)
	mux.HandleFunc(queue.TaskMatchBuyRequests, w.handleMatchBuyRequests)
	mux.HandleFunc(queue.TaskProcessPayout, w.handleProcessPayout)
	return w.server.Run(mux)
}

func (w *Worker) handleLockExpiry(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		ListingID string `json:"listing_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("parse lock expiry payload: %w", err)
	}
	listingID, err := uuid.Parse(payload.ListingID)
	if err != nil {
		return err
	}

	listing, err := w.listings.FindByID(ctx, listingID)
	if err != nil {
		return err
	}

	// Only unlock if still LOCKED (idempotent — if already SOLD, skip)
	if listing.Status != "LOCKED" {
		return nil
	}

	w.logger.Info("lock expired — unlocking listing", "listing_id", listingID)

	if err := w.listings.Unlock(ctx, listingID); err != nil {
		return fmt.Errorf("unlock listing: %w", err)
	}

	// Trigger payment failure to update transaction status
	if listing.LockBuyerID != nil {
		txn, err := w.listings.FindByID(ctx, listingID)
		if err == nil && txn != nil {
			// Handle as payment failure
			_ = w.purchase.HandlePaymentFailure(ctx, "lock_expired:"+listingID.String())
		}
	}

	return nil
}

func (w *Worker) handleMatchBuyRequests(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		ListingID string `json:"listing_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	listingID, err := uuid.Parse(payload.ListingID)
	if err != nil {
		return err
	}
	return w.requests.MatchAndNotify(ctx, listingID)
}

func (w *Worker) handleProcessPayout(ctx context.Context, t *asynq.Task) error {
	var payload struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}
	txnID, err := uuid.Parse(payload.TransactionID)
	if err != nil {
		return err
	}
	return w.payout.ProcessPayout(ctx, txnID)
}
