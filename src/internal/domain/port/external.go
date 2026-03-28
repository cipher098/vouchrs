package port

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
)

// --- Token Service ---

type TokenClaims struct {
	UserID uuid.UUID
	Role   string
	Email  string
}

type TokenService interface {
	GenerateAccessToken(claims TokenClaims) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (uuid.UUID, error)
	RevokeToken(ctx context.Context, token string) error
	IsRevoked(ctx context.Context, token string) (bool, error)
}

// --- Cipher Service ---

type CipherService interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	// Hash returns a deterministic SHA-256 hex hash (for duplicate detection).
	Hash(plaintext string) string
}

// --- Cache ---

type CacheService interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	// SetNX sets key only if it does not exist. Returns true if set.
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
}

// --- Storage ---

type StorageService interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, key string) error
	PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// --- SMS ---

type SMSService interface {
	SendOTP(ctx context.Context, phone, otp string) error
}

// --- Email ---

type EmailService interface {
	SendOTP(ctx context.Context, email, otp string) error
	// SendCardCode sends the decrypted card code to the buyer's email.
	// Code is NEVER returned to the API — email only.
	SendCardCode(ctx context.Context, email, brandName string, faceValue float64, code string) error
	SendPurchaseReceipt(ctx context.Context, email string, txn *entity.Transaction) error
	SendCardRequestUpdate(ctx context.Context, email string, req *entity.CardRequest) error
	SendBuyRequestAlert(ctx context.Context, email string, listing *entity.Listing, brandName string) error
	SendAdminCardRequestNotification(ctx context.Context, adminEmails []string, req *entity.CardRequest) error
}

// --- Verification (Qwikcilver) ---

type VerificationResult struct {
	IsValid      bool
	Balance      float64
	Status       string // "active", "claimed", "expired"
	FailReason   string
	ResponseHash string // SHA-256 of raw response for audit log
}

type VerificationService interface {
	// Verify checks a gift card code via the appropriate verification source for the brand.
	Verify(ctx context.Context, brandSlug, cardCode string) (*VerificationResult, error)
}

// --- Payment Gateway (pluggable — PhonePe today, swappable) ---

type PaymentOrderInput struct {
	MerchantTransactionID string
	Amount                float64 // in INR
	UserID                uuid.UUID
	RedirectURL           string
	CallbackURL           string
}

type PaymentOrderResult struct {
	MerchantTransactionID string
	PaymentURL            string // redirect buyer here
	Raw                   map[string]interface{}
}

type PaymentWebhookEvent struct {
	MerchantTransactionID string
	Status                string // "SUCCESS", "FAILURE", "PENDING"
	Amount                float64
	Raw                   map[string]interface{}
}

// PaymentGateway is the port for buyer payment processing.
// Implement this interface to support any payment gateway (PhonePe, Razorpay, Paytm, etc.)
type PaymentGateway interface {
	// CreateOrder initiates a payment and returns a URL to redirect the buyer.
	CreateOrder(ctx context.Context, input PaymentOrderInput) (*PaymentOrderResult, error)
	// VerifyWebhook validates the webhook signature and parses the event.
	VerifyWebhook(ctx context.Context, body []byte, headers map[string]string) (*PaymentWebhookEvent, error)
	// GetPaymentStatus fetches the current status of a payment order.
	GetPaymentStatus(ctx context.Context, merchantTransactionID string) (*PaymentWebhookEvent, error)
}

// --- Payout Service (Razorpay) ---

type CreatePayoutInput struct {
	UPIID        string
	Amount       float64
	Purpose      string
	ReferenceID  string
	Narration    string
}

type PayoutResult struct {
	PayoutID string
	Status   string // "processing", "processed", "failed"
}

type PayoutService interface {
	CreatePayout(ctx context.Context, input CreatePayoutInput) (*PayoutResult, error)
	GetPayoutStatus(ctx context.Context, payoutID string) (*PayoutResult, error)
}

// --- Google OAuth (admin) ---

type OAuthUser struct {
	Email   string
	Name    string
	Picture string
}

type OAuthService interface {
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthUser, error)
}

// --- Job Queue ---

type JobQueue interface {
	// EnqueueLockExpiry schedules a job to unlock a listing after the lock window.
	// The job is idempotent — if the listing is already SOLD or LIVE, it's a no-op.
	EnqueueLockExpiry(ctx context.Context, listingID uuid.UUID, delay time.Duration) error
	// CancelLockExpiry removes the pending lock expiry job (called on payment success).
	CancelLockExpiry(ctx context.Context, listingID uuid.UUID) error
	// EnqueueMatchBuyRequests triggers buy-request matching after a new listing goes LIVE.
	EnqueueMatchBuyRequests(ctx context.Context, listingID uuid.UUID) error
	// EnqueuePayout queues a seller payout job.
	EnqueuePayout(ctx context.Context, transactionID uuid.UUID) error
}
