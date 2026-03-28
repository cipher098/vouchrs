package dashboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type Service struct {
	listings     port.ListingRepository
	transactions port.TransactionRepository
	buyRequests  port.BuyRequestRepository
	cardRequests port.CardRequestRepository
}

func NewService(
	listings port.ListingRepository,
	transactions port.TransactionRepository,
	buyRequests port.BuyRequestRepository,
	cardRequests port.CardRequestRepository,
) port.DashboardService {
	return &Service{
		listings:     listings,
		transactions: transactions,
		buyRequests:  buyRequests,
		cardRequests: cardRequests,
	}
}

func (s *Service) GetMyListings(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]*entity.Listing, *pagination.Meta, error) {
	listings, total, err := s.listings.ListBySeller(ctx, userID, p)
	if err != nil {
		return nil, nil, err
	}
	meta := pagination.NewMeta(p, total)
	return listings, &meta, nil
}

func (s *Service) GetMyPurchases(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]*entity.Transaction, *pagination.Meta, error) {
	txns, total, err := s.transactions.ListByBuyer(ctx, userID, p)
	if err != nil {
		return nil, nil, err
	}
	meta := pagination.NewMeta(p, total)
	return txns, &meta, nil
}

func (s *Service) GetMyRequests(ctx context.Context, userID uuid.UUID) (*port.MyRequestsResult, error) {
	buyReqs, err := s.buyRequests.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	cardReqs, err := s.cardRequests.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &port.MyRequestsResult{
		BuyRequests:  buyReqs,
		CardRequests: cardReqs,
	}, nil
}

var _ port.DashboardService = (*Service)(nil)
