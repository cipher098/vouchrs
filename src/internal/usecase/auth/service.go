package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const (
	otpTTL          = 10 * time.Minute
	maxOTPAttempts  = 5
	otpAttemptsWindow = 15 * time.Minute
)

type Service struct {
	users       port.UserRepository
	tokens      port.TokenService
	cache       port.CacheService
	sms         port.SMSService
	email       port.EmailService
	oauth       port.OAuthService
	otpLen      int
	otpDevMode  bool
	adminEmails []string
	logger      *slog.Logger
}

func NewService(
	users port.UserRepository,
	tokens port.TokenService,
	cache port.CacheService,
	sms port.SMSService,
	email port.EmailService,
	oauth port.OAuthService,
	otpLen int,
	otpDevMode bool,
	adminEmails []string,
	logger *slog.Logger,
) port.AuthService {
	return &Service{
		users:       users,
		tokens:      tokens,
		cache:       cache,
		sms:         sms,
		email:       email,
		oauth:       oauth,
		otpLen:      otpLen,
		otpDevMode:  otpDevMode,
		adminEmails: adminEmails,
		logger:      logger,
	}
}

// RequestOTP generates and sends an OTP. contact is phone number or email.
func (s *Service) RequestOTP(ctx context.Context, contact string) error {
	// Rate-limit: block after maxOTPAttempts within the window
	attemptsKey := "otp_attempts:" + contact
	var attempts int
	if err := s.cache.Get(ctx, attemptsKey, &attempts); err == nil {
		if attempts >= maxOTPAttempts {
			return apperror.ErrOTPTooManyAttempts
		}
	}

	otp, err := generateOTP(s.otpLen)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}

	if err := s.cache.Set(ctx, "otp:"+contact, otp, otpTTL); err != nil {
		return fmt.Errorf("store otp: %w", err)
	}

	// Increment attempt counter
	newCount := attempts + 1
	_ = s.cache.Set(ctx, attemptsKey, newCount, otpAttemptsWindow)

	// In dev mode, print OTP to logs instead of sending externally.
	if s.otpDevMode {
		s.logger.Info("DEV MODE — OTP generated", "contact", contact, "otp", otp)
		return nil
	}

	// Send via appropriate channel
	if isEmail(contact) {
		if err := s.email.SendOTP(ctx, contact, otp); err != nil {
			return fmt.Errorf("send otp email: %w", err)
		}
	} else {
		if err := s.sms.SendOTP(ctx, contact, otp); err != nil {
			return fmt.Errorf("send otp sms: %w", err)
		}
	}
	return nil
}

// VerifyOTP validates the OTP and issues JWT tokens. Creates a new user on first login.
func (s *Service) VerifyOTP(ctx context.Context, contact, otp string) (*port.AuthTokenPair, *entity.User, error) {
	var storedOTP string
	if err := s.cache.Get(ctx, "otp:"+contact, &storedOTP); err != nil {
		return nil, nil, apperror.ErrOTPInvalid
	}
	if storedOTP != otp {
		return nil, nil, apperror.ErrOTPInvalid
	}
	// OTP is single-use
	_ = s.cache.Delete(ctx, "otp:"+contact, "otp_attempts:"+contact)

	user, err := s.findOrCreateUser(ctx, contact)
	if err != nil {
		return nil, nil, err
	}

	if user.IsBanned {
		return nil, nil, apperror.ErrForbidden
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*port.AuthTokenPair, error) {
	userID, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, err
	}
	if user.IsBanned {
		return nil, apperror.ErrForbidden
	}

	return s.issueTokens(user)
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	return s.tokens.RevokeToken(ctx, accessToken)
}

func (s *Service) GetAdminOAuthURL(state string) string {
	return s.oauth.GetAuthURL(state)
}

func (s *Service) HandleAdminOAuthCallback(ctx context.Context, code string) (*port.AuthTokenPair, *entity.User, error) {
	oauthUser, err := s.oauth.ExchangeCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth exchange: %w", err)
	}

	if !s.isAdminEmail(oauthUser.Email) {
		return nil, nil, apperror.ErrForbidden
	}

	user, err := s.users.FindByEmail(ctx, oauthUser.Email)
	if errors.Is(err, apperror.ErrNotFound) {
		user = &entity.User{
			ID:         uuid.New(),
			Email:      oauthUser.Email,
			FullName:   oauthUser.Name,
			Role:       entity.UserRoleAdmin,
			IsVerified: true,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("create admin user: %w", err)
		}
	} else if err != nil {
		return nil, nil, err
	}

	tokens, err := s.issueTokens(user)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

// --- helpers ---

func (s *Service) findOrCreateUser(ctx context.Context, contact string) (*entity.User, error) {
	var user *entity.User
	var err error

	if isEmail(contact) {
		user, err = s.users.FindByEmail(ctx, contact)
	} else {
		user, err = s.users.FindByPhone(ctx, contact)
	}

	if errors.Is(err, apperror.ErrNotFound) {
		// First login — create user
		user = &entity.User{
			ID:         uuid.New(),
			Role:       entity.UserRoleBuyer,
			IsVerified: true,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		if isEmail(contact) {
			user.Email = contact
		} else {
			user.Phone = contact
		}
		if err := s.users.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
		return user, nil
	}
	return user, err
}

func (s *Service) issueTokens(user *entity.User) (*port.AuthTokenPair, error) {
	access, err := s.tokens.GenerateAccessToken(port.TokenClaims{
		UserID: user.ID,
		Role:   string(user.Role),
		Email:  user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refresh, err := s.tokens.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	return &port.AuthTokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *Service) isAdminEmail(email string) bool {
	for _, e := range s.adminEmails {
		if strings.EqualFold(strings.TrimSpace(e), strings.TrimSpace(email)) {
			return true
		}
	}
	return false
}

func generateOTP(length int) (string, error) {
	digits := "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[n.Int64()]
	}
	return string(otp), nil
}

func isEmail(contact string) bool {
	return strings.Contains(contact, "@")
}
