package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

// Task type names — keep in sync with worker handlers.
const (
	TaskLockExpiry      = "listing:lock_expiry"
	TaskMatchBuyRequests = "listing:match_buy_requests"
	TaskProcessPayout   = "transaction:payout"
)

type asynqJobQueue struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

// NewAsynqJobQueue creates a Redis-backed job queue using asynq.
func NewAsynqJobQueue(redisAddr, redisPassword string, redisDB int) (port.JobQueue, *asynq.Client) {
	opt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	}
	client := asynq.NewClient(opt)
	inspector := asynq.NewInspector(opt)
	return &asynqJobQueue{client: client, inspector: inspector}, client
}

func (q *asynqJobQueue) EnqueueLockExpiry(ctx context.Context, listingID uuid.UUID, delay time.Duration) error {
	payload, _ := json.Marshal(map[string]string{"listing_id": listingID.String()})
	task := asynq.NewTask(TaskLockExpiry, payload,
		asynq.TaskID("lock_expiry:"+listingID.String()), // idempotent — one job per listing
		asynq.ProcessIn(delay),
		asynq.MaxRetry(0), // do not retry lock expiry
	)
	_, err := q.client.EnqueueContext(ctx, task)
	if err != nil && err != asynq.ErrTaskIDConflict {
		return fmt.Errorf("enqueue lock expiry: %w", err)
	}
	return nil
}

func (q *asynqJobQueue) CancelLockExpiry(_ context.Context, listingID uuid.UUID) error {
	// Delete the scheduled task from the default queue. Errors are non-fatal —
	// if the task already fired it won't exist; if it fires after SOLD it's a no-op.
	err := q.inspector.DeleteTask("default", "lock_expiry:"+listingID.String())
	if err != nil && err.Error() != "asynq: task not found" {
		return err
	}
	return nil
}

func (q *asynqJobQueue) EnqueueMatchBuyRequests(ctx context.Context, listingID uuid.UUID) error {
	payload, _ := json.Marshal(map[string]string{"listing_id": listingID.String()})
	task := asynq.NewTask(TaskMatchBuyRequests, payload,
		asynq.MaxRetry(3),
	)
	_, err := q.client.EnqueueContext(ctx, task)
	return err
}

func (q *asynqJobQueue) EnqueuePayout(ctx context.Context, transactionID uuid.UUID) error {
	payload, _ := json.Marshal(map[string]string{"transaction_id": transactionID.String()})
	task := asynq.NewTask(TaskProcessPayout, payload,
		asynq.TaskID("payout:"+transactionID.String()),
		asynq.MaxRetry(5),
	)
	_, err := q.client.EnqueueContext(ctx, task)
	if err != nil && err != asynq.ErrTaskIDConflict {
		return fmt.Errorf("enqueue payout: %w", err)
	}
	return nil
}
