package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

// --- Auth ---

type AuthTokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthService interface {
	// RequestOTP sends a 6-digit OTP to phone (SMS) or email.
	RequestOTP(ctx context.Context, contact string) error
	// VerifyOTP validates the OTP and returns JWT tokens. Creates user on first login.
	VerifyOTP(ctx context.Context, contact, otp string) (*AuthTokenPair, *entity.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthTokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	// GetAdminOAuthURL returns the Google OAuth consent URL for admin login.
	GetAdminOAuthURL(state string) string
	// HandleAdminOAuthCallback exchanges the Google code and returns tokens.
	// Returns ErrForbidden if the email is not in the admin allowlist.
	HandleAdminOAuthCallback(ctx context.Context, code string) (*AuthTokenPair, *entity.User, error)
}

// --- Listing ---

type CreateListingInput struct {
	BrandID        uuid.UUID
	FaceValue      float64
	CardCode       string   // plaintext — will be encrypted
	ExpiryDate     string   // MM/YY or MM/YYYY as printed on card
	CardPin        string   // plaintext — will be encrypted; empty if brand doesn't require PIN
	AcceptPool     bool     // true = Vouchrs pool at recommended price
	CustomDiscount *float64 // used when AcceptPool=false
}

type MarketplaceResult struct {
	PoolGroups       []*entity.PoolGroup
	IndividualListings []*entity.Listing
	Total            int
}

type RecommendedPriceResult struct {
	RecommendedDiscountPct float64 `json:"recommended_discount_pct"`
	SellerPrice            float64 `json:"seller_price"`
	SellerPayout           float64 `json:"seller_payout"`
	BuyerPrice             float64 `json:"buyer_price"`
	PlatformFeePerSide     float64 `json:"platform_fee_per_side"`
	AvgSellTimeMins        float64 `json:"avg_sell_time_mins"`
}

type ListingService interface {
	// CreateListing runs Gate 1 verification, encrypts the code, and creates the listing.
	CreateListing(ctx context.Context, sellerID uuid.UUID, input CreateListingInput) (*entity.Listing, error)
	// CancelListing cancels a LIVE listing. Returns ErrNotListingOwner if not the seller.
	CancelListing(ctx context.Context, sellerID, listingID uuid.UUID) error
	GetListing(ctx context.Context, id uuid.UUID) (*entity.Listing, error)
	// GetMarketplace returns pool groups at the top and individual listings below.
	GetMarketplace(ctx context.Context, f MarketplaceFilter) (*MarketplaceResult, error)
	// GetRecommendedPrice returns the platform-recommended pricing for a given brand+face value.
	GetRecommendedPrice(ctx context.Context, brandID uuid.UUID, faceValue float64) (*RecommendedPriceResult, error)
	// GetPoolGroup returns a single pool group by ID.
	GetPoolGroup(ctx context.Context, id uuid.UUID) (*entity.PoolGroup, error)
}

// --- Purchase ---

type InitiateBuyResult struct {
	Transaction   *entity.Transaction
	PaymentURL    string // redirect buyer to this PhonePe URL
	LockExpiresAt string // ISO8601
	ReturnURL     string // PhonePe redirect URL (backend-constructed fallback)
}

type PurchaseService interface {
	// InitiateBuy runs Gate 2, locks the listing, creates a transaction,
	// and returns a PhonePe payment URL.
	InitiateBuy(ctx context.Context, buyerID, listingID uuid.UUID) (*InitiateBuyResult, error)
	// InitiateBuyFromPool resolves the oldest LIVE listing in the pool group (FIFO)
	// and delegates to InitiateBuy. Returns ErrNotFound if the pool is empty.
	InitiateBuyFromPool(ctx context.Context, buyerID, poolGroupID uuid.UUID) (*InitiateBuyResult, error)
	// HandlePaymentSuccess is called by the webhook handler on successful payment.
	// It marks the transaction paid and sends the card code by email.
	HandlePaymentSuccess(ctx context.Context, merchantTransactionID string) error
	// HandlePaymentFailure is called by the webhook on failure — unlocks the listing.
	HandlePaymentFailure(ctx context.Context, merchantTransactionID string) error
	// ConfirmRedemption is called by the buyer after they redeem the card.
	ConfirmRedemption(ctx context.Context, buyerID, transactionID uuid.UUID) error
	// GetTransaction returns a transaction. Validates that userID is buyer or seller.
	GetTransaction(ctx context.Context, userID, transactionID uuid.UUID) (*entity.Transaction, error)
}

// --- Request ---

type CreateBuyRequestInput struct {
	BrandID  uuid.UUID
	MinValue float64
	MaxValue float64
	MaxPrice float64
}

type CreateCardRequestInput struct {
	Brand        string
	DesiredValue float64
	Urgency      entity.CardRequestUrgency
}

type MyRequestsResult struct {
	BuyRequests  []*entity.BuyRequest
	CardRequests []*entity.CardRequest
}

type RequestService interface {
	CreateBuyRequest(ctx context.Context, userID uuid.UUID, input CreateBuyRequestInput) (*entity.BuyRequest, error)
	DeleteBuyRequest(ctx context.Context, userID, requestID uuid.UUID) error
	ListMyBuyRequests(ctx context.Context, userID uuid.UUID) ([]*entity.BuyRequest, error)
	CreateCardRequest(ctx context.Context, userID uuid.UUID, input CreateCardRequestInput) (*entity.CardRequest, error)
	ListMyCardRequests(ctx context.Context, userID uuid.UUID) ([]*entity.CardRequest, error)
	// MatchAndNotify finds active buy requests matching the listing and sends alerts.
	// Called by the job queue after a new listing goes LIVE.
	MatchAndNotify(ctx context.Context, listingID uuid.UUID) error
}

// --- Dashboard ---

type DashboardService interface {
	GetMyListings(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]*entity.Listing, *pagination.Meta, error)
	GetMyPurchases(ctx context.Context, userID uuid.UUID, p pagination.Params) ([]*entity.Transaction, *pagination.Meta, error)
	GetMyRequests(ctx context.Context, userID uuid.UUID) (*MyRequestsResult, error)
}

// --- Payout ---

// PayoutUsecase is the application-level payout service (drives PayoutService port).
type PayoutUsecase interface {
	// ProcessPayout sends the seller's UPI payout via Razorpay.
	// Called by the job queue after buyer confirms redemption.
	ProcessPayout(ctx context.Context, transactionID uuid.UUID) error
}

// --- Admin ---

type AdminCardRequestAction string

const (
	AdminActionApprove AdminCardRequestAction = "approve"
	AdminActionReject  AdminCardRequestAction = "reject"
	AdminActionDefer   AdminCardRequestAction = "defer"
)

type AdminStats struct {
	TotalUsers       int     `json:"total_users"`
	TotalListings    int     `json:"total_listings"`
	LiveListings     int     `json:"live_listings"`
	TotalTransactions int    `json:"total_transactions"`
	TotalGMV         float64 `json:"total_gmv"`
	PendingRequests  int     `json:"pending_card_requests"`
	OpenFraudFlags   int     `json:"open_fraud_flags"`
}

type AdminService interface {
	ListCardRequests(ctx context.Context) ([]*entity.CardRequest, error)
	ReviewCardRequest(ctx context.Context, reqID uuid.UUID, action AdminCardRequestAction, notes string) error
	ListFraudFlags(ctx context.Context) ([]*entity.FraudFlag, error)
	ResolveFraudFlag(ctx context.Context, flagID uuid.UUID) error
	BanUser(ctx context.Context, userID uuid.UUID) error
	GetStats(ctx context.Context) (*AdminStats, error)
	ListAllListings(ctx context.Context, p pagination.Params) ([]*entity.Listing, *pagination.Meta, error)
	ListAllTransactions(ctx context.Context, p pagination.Params) ([]*entity.Transaction, *pagination.Meta, error)
}
