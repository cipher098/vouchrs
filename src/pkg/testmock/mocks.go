// Package testmock provides testify mock implementations of all domain ports.
// Use these in use-case unit tests to avoid hitting real infrastructure.
package testmock

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

// --- UserRepo ---

type UserRepo struct{ mock.Mock }

func (m *UserRepo) Create(ctx context.Context, u *entity.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *UserRepo) FindByPhone(ctx context.Context, phone string) (*entity.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *UserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}
func (m *UserRepo) Update(ctx context.Context, u *entity.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *UserRepo) IncrListingCount(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *UserRepo) List(ctx context.Context, p pagination.Params) ([]*entity.User, int, error) {
	args := m.Called(ctx, p)
	return args.Get(0).([]*entity.User), args.Int(1), args.Error(2)
}
func (m *UserRepo) Ban(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- TokenSvc ---

type TokenSvc struct{ mock.Mock }

func (m *TokenSvc) GenerateAccessToken(c port.TokenClaims) (string, error) {
	args := m.Called(c)
	return args.String(0), args.Error(1)
}
func (m *TokenSvc) GenerateRefreshToken(id uuid.UUID) (string, error) {
	args := m.Called(id)
	return args.String(0), args.Error(1)
}
func (m *TokenSvc) ValidateAccessToken(t string) (*port.TokenClaims, error) {
	args := m.Called(t)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.TokenClaims), args.Error(1)
}
func (m *TokenSvc) ValidateRefreshToken(t string) (uuid.UUID, error) {
	args := m.Called(t)
	return args.Get(0).(uuid.UUID), args.Error(1)
}
func (m *TokenSvc) RevokeToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}
func (m *TokenSvc) IsRevoked(ctx context.Context, token string) (bool, error) {
	args := m.Called(ctx, token)
	return args.Bool(0), args.Error(1)
}

// --- CacheSvc ---

type CacheSvc struct{ mock.Mock }

func (m *CacheSvc) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}
func (m *CacheSvc) Get(ctx context.Context, key string, dest interface{}) error {
	return m.Called(ctx, key, dest).Error(0)
}
func (m *CacheSvc) Delete(ctx context.Context, keys ...string) error {
	args := []interface{}{ctx}
	for _, k := range keys {
		args = append(args, k)
	}
	return m.Called(args...).Error(0)
}
func (m *CacheSvc) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}
func (m *CacheSvc) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, ttl)
	return args.Bool(0), args.Error(1)
}

// --- SMSSvc ---

type SMSSvc struct{ mock.Mock }

func (m *SMSSvc) SendOTP(ctx context.Context, phone string) (string, error) {
	args := m.Called(ctx, phone)
	return args.String(0), args.Error(1)
}

func (m *SMSSvc) VerifyOTP(ctx context.Context, sessionID, otp string) error {
	return m.Called(ctx, sessionID, otp).Error(0)
}

// --- EmailSvc ---

type EmailSvc struct{ mock.Mock }

func (m *EmailSvc) SendOTP(ctx context.Context, email, otp string) error {
	return m.Called(ctx, email, otp).Error(0)
}
func (m *EmailSvc) SendCardCode(ctx context.Context, email, brand string, value float64, code string) error {
	return m.Called(ctx, email, brand, value, code).Error(0)
}
func (m *EmailSvc) SendPurchaseReceipt(ctx context.Context, email string, txn *entity.Transaction) error {
	return m.Called(ctx, email, txn).Error(0)
}
func (m *EmailSvc) SendCardRequestUpdate(ctx context.Context, email string, req *entity.CardRequest) error {
	return m.Called(ctx, email, req).Error(0)
}
func (m *EmailSvc) SendBuyRequestAlert(ctx context.Context, email string, l *entity.Listing, brand string) error {
	return m.Called(ctx, email, l, brand).Error(0)
}
func (m *EmailSvc) SendAdminCardRequestNotification(ctx context.Context, admins []string, req *entity.CardRequest) error {
	return m.Called(ctx, admins, req).Error(0)
}

// --- OAuthSvc ---

type OAuthSvc struct{ mock.Mock }

func (m *OAuthSvc) GetAuthURL(state string) string {
	return m.Called(state).String(0)
}
func (m *OAuthSvc) ExchangeCode(ctx context.Context, code string) (*port.OAuthUser, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.OAuthUser), args.Error(1)
}

// --- VerificationSvc ---

type VerificationSvc struct{ mock.Mock }

func (m *VerificationSvc) Verify(ctx context.Context, brand, code string) (*port.VerificationResult, error) {
	args := m.Called(ctx, brand, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.VerificationResult), args.Error(1)
}

// --- CipherSvc ---

type CipherSvc struct{ mock.Mock }

func (m *CipherSvc) Encrypt(s string) (string, error) {
	args := m.Called(s)
	return args.String(0), args.Error(1)
}
func (m *CipherSvc) Decrypt(s string) (string, error) {
	args := m.Called(s)
	return args.String(0), args.Error(1)
}
func (m *CipherSvc) Hash(s string) string {
	return m.Called(s).String(0)
}

// --- JobQueue ---

type JobQueue struct{ mock.Mock }

func (m *JobQueue) EnqueueLockExpiry(ctx context.Context, id uuid.UUID, d time.Duration) error {
	return m.Called(ctx, id, d).Error(0)
}
func (m *JobQueue) CancelLockExpiry(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *JobQueue) EnqueueMatchBuyRequests(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *JobQueue) EnqueuePayout(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- PaymentGW ---

type PaymentGW struct{ mock.Mock }

func (m *PaymentGW) CreateOrder(ctx context.Context, in port.PaymentOrderInput) (*port.PaymentOrderResult, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.PaymentOrderResult), args.Error(1)
}
func (m *PaymentGW) VerifyWebhook(ctx context.Context, body []byte, h map[string]string) (*port.PaymentWebhookEvent, error) {
	args := m.Called(ctx, body, h)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.PaymentWebhookEvent), args.Error(1)
}
func (m *PaymentGW) GetPaymentStatus(ctx context.Context, txnID string) (*port.PaymentWebhookEvent, error) {
	args := m.Called(ctx, txnID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*port.PaymentWebhookEvent), args.Error(1)
}

// --- StorageSvc ---

type StorageSvc struct{ mock.Mock }

func (m *StorageSvc) Upload(ctx context.Context, key string, data io.Reader, ct string) (string, error) {
	args := m.Called(ctx, key, data, ct)
	return args.String(0), args.Error(1)
}
func (m *StorageSvc) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
func (m *StorageSvc) PresignedURL(ctx context.Context, key string, exp time.Duration) (string, error) {
	args := m.Called(ctx, key, exp)
	return args.String(0), args.Error(1)
}

// ensure interfaces are satisfied at compile time
var _ port.UserRepository = (*UserRepo)(nil)
var _ port.TokenService = (*TokenSvc)(nil)
var _ port.CacheService = (*CacheSvc)(nil)
var _ port.SMSService = (*SMSSvc)(nil)
var _ port.EmailService = (*EmailSvc)(nil)
var _ port.OAuthService = (*OAuthSvc)(nil)
var _ port.VerificationService = (*VerificationSvc)(nil)
var _ port.CipherService = (*CipherSvc)(nil)
var _ port.JobQueue = (*JobQueue)(nil)
var _ port.PaymentGateway = (*PaymentGW)(nil)
var _ port.StorageService = (*StorageSvc)(nil)
