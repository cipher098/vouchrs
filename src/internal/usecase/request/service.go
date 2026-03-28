package request

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

type Service struct {
	buyRequests  port.BuyRequestRepository
	cardRequests port.CardRequestRepository
	listings     port.ListingRepository
	brands       port.BrandRepository
	users        port.UserRepository
	email        port.EmailService
	adminEmails  []string
	logger       *slog.Logger
}

func NewService(
	buyRequests port.BuyRequestRepository,
	cardRequests port.CardRequestRepository,
	listings port.ListingRepository,
	brands port.BrandRepository,
	users port.UserRepository,
	email port.EmailService,
	adminEmails []string,
	logger *slog.Logger,
) port.RequestService {
	return &Service{
		buyRequests:  buyRequests,
		cardRequests: cardRequests,
		listings:     listings,
		brands:       brands,
		users:        users,
		email:        email,
		adminEmails:  adminEmails,
		logger:       logger,
	}
}

func (s *Service) CreateBuyRequest(ctx context.Context, userID uuid.UUID, input port.CreateBuyRequestInput) (*entity.BuyRequest, error) {
	req := &entity.BuyRequest{
		UserID:    userID,
		BrandID:   input.BrandID,
		MinValue:  input.MinValue,
		MaxValue:  input.MaxValue,
		MaxPrice:  input.MaxPrice,
		ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour), // 30-day expiry
	}
	if err := s.buyRequests.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) DeleteBuyRequest(ctx context.Context, userID, requestID uuid.UUID) error {
	req, err := s.buyRequests.FindByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req.UserID != userID {
		return apperror.ErrForbidden
	}
	return s.buyRequests.Delete(ctx, requestID)
}

func (s *Service) ListMyBuyRequests(ctx context.Context, userID uuid.UUID) ([]*entity.BuyRequest, error) {
	return s.buyRequests.ListByUser(ctx, userID)
}

func (s *Service) CreateCardRequest(ctx context.Context, userID uuid.UUID, input port.CreateCardRequestInput) (*entity.CardRequest, error) {
	req := &entity.CardRequest{
		UserID:       userID,
		Brand:        input.Brand,
		DesiredValue: input.DesiredValue,
		Urgency:      input.Urgency,
	}
	if err := s.cardRequests.Create(ctx, req); err != nil {
		return nil, err
	}

	// Notify admin
	if err := s.email.SendAdminCardRequestNotification(ctx, s.adminEmails, req); err != nil {
		s.logger.Warn("send admin card request notification failed", "error", err)
	}

	return req, nil
}

func (s *Service) ListMyCardRequests(ctx context.Context, userID uuid.UUID) ([]*entity.CardRequest, error) {
	return s.cardRequests.ListByUser(ctx, userID)
}

// MatchAndNotify finds active buy requests matching the listing and sends email alerts.
// This is called by the job queue worker after a new listing goes LIVE.
func (s *Service) MatchAndNotify(ctx context.Context, listingID uuid.UUID) error {
	listing, err := s.listings.FindByID(ctx, listingID)
	if err != nil {
		return fmt.Errorf("find listing: %w", err)
	}

	brand, err := s.brands.FindByID(ctx, listing.BrandID)
	if err != nil {
		return fmt.Errorf("find brand: %w", err)
	}

	matches, err := s.buyRequests.FindMatchingForListing(ctx, listing)
	if err != nil {
		return fmt.Errorf("find matching buy requests: %w", err)
	}

	for _, req := range matches {
		user, err := s.users.FindByID(ctx, req.UserID)
		if err != nil {
			s.logger.Warn("find user for buy request alert failed", "user_id", req.UserID, "error", err)
			continue
		}
		if user.Email == "" {
			continue
		}
		if err := s.email.SendBuyRequestAlert(ctx, user.Email, listing, brand.Name); err != nil {
			s.logger.Warn("send buy request alert failed", "user_id", req.UserID, "error", err)
			continue
		}
		_ = s.buyRequests.IncrAlertCount(ctx, req.ID)
	}
	return nil
}

var _ port.RequestService = (*Service)(nil)
var _ = errors.New
