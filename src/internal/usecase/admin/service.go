package admin

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type Service struct {
	users        port.UserRepository
	listings     port.ListingRepository
	transactions port.TransactionRepository
	cardRequests port.CardRequestRepository
	fraudFlags   port.FraudFlagRepository
	email        port.EmailService
	logger       *slog.Logger
}

func NewService(
	users port.UserRepository,
	listings port.ListingRepository,
	transactions port.TransactionRepository,
	cardRequests port.CardRequestRepository,
	fraudFlags port.FraudFlagRepository,
	email port.EmailService,
	logger *slog.Logger,
) port.AdminService {
	return &Service{
		users:        users,
		listings:     listings,
		transactions: transactions,
		cardRequests: cardRequests,
		fraudFlags:   fraudFlags,
		email:        email,
		logger:       logger,
	}
}

func (s *Service) ListCardRequests(ctx context.Context) ([]*entity.CardRequest, error) {
	return s.cardRequests.ListPending(ctx)
}

func (s *Service) ReviewCardRequest(ctx context.Context, reqID uuid.UUID, action port.AdminCardRequestAction, notes string) error {
	req, err := s.cardRequests.FindByID(ctx, reqID)
	if err != nil {
		return err
	}

	var newStatus entity.CardRequestStatus
	switch action {
	case port.AdminActionApprove:
		newStatus = entity.CardRequestStatusUnderReview
	case port.AdminActionReject:
		newStatus = entity.CardRequestStatusRejected
	case port.AdminActionDefer:
		newStatus = entity.CardRequestStatusDeferred
	default:
		return apperror.New(apperror.ErrBadRequest, "invalid admin action")
	}

	if err := s.cardRequests.UpdateStatus(ctx, reqID, newStatus, notes); err != nil {
		return err
	}

	// Notify user of status update
	req.Status = newStatus
	req.AdminNotes = notes
	user, err := s.users.FindByID(ctx, req.UserID)
	if err != nil {
		s.logger.Warn("find user for card request notification", "error", err)
		return nil
	}
	if user.Email != "" {
		_ = s.email.SendCardRequestUpdate(ctx, user.Email, req)
	}
	return nil
}

func (s *Service) ListFraudFlags(ctx context.Context) ([]*entity.FraudFlag, error) {
	return s.fraudFlags.ListUnresolved(ctx)
}

func (s *Service) ResolveFraudFlag(ctx context.Context, flagID uuid.UUID) error {
	return s.fraudFlags.Resolve(ctx, flagID)
}

func (s *Service) BanUser(ctx context.Context, userID uuid.UUID) error {
	return s.users.Ban(ctx, userID)
}

func (s *Service) GetStats(ctx context.Context) (*port.AdminStats, error) {
	// These counts are rough — for a proper dashboard, use SQL aggregates.
	_, userTotal, err := s.users.List(ctx, pagination.Params{Limit: 1, Offset: 0})
	if err != nil {
		return nil, err
	}

	_, listingTotal, err := s.listings.ListAll(ctx, pagination.Params{Limit: 1, Offset: 0})
	if err != nil {
		return nil, err
	}

	_, txnTotal, err := s.transactions.ListAll(ctx, pagination.Params{Limit: 1, Offset: 0})
	if err != nil {
		return nil, err
	}

	pendingReqs, err := s.cardRequests.ListPending(ctx)
	if err != nil {
		return nil, err
	}

	openFlags, err := s.fraudFlags.ListUnresolved(ctx)
	if err != nil {
		return nil, err
	}

	return &port.AdminStats{
		TotalUsers:        userTotal,
		TotalListings:     listingTotal,
		TotalTransactions: txnTotal,
		PendingRequests:   len(pendingReqs),
		OpenFraudFlags:    len(openFlags),
	}, nil
}

func (s *Service) ListAllListings(ctx context.Context, p pagination.Params) ([]*entity.Listing, *pagination.Meta, error) {
	listings, total, err := s.listings.ListAll(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	meta := pagination.NewMeta(p, total)
	return listings, &meta, nil
}

func (s *Service) ListAllTransactions(ctx context.Context, p pagination.Params) ([]*entity.Transaction, *pagination.Meta, error) {
	txns, total, err := s.transactions.ListAll(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	meta := pagination.NewMeta(p, total)
	return txns, &meta, nil
}

var _ port.AdminService = (*Service)(nil)
var _ = errors.New
