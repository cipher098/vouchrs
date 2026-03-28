package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/internal/usecase/auth"
	"github.com/gothi/vouchrs/src/pkg/testmock"
)

func newService(t *testing.T) (port.AuthService, *testmock.UserRepo, *testmock.TokenSvc, *testmock.CacheSvc) {
	t.Helper()
	users := &testmock.UserRepo{}
	tokens := &testmock.TokenSvc{}
	cache := &testmock.CacheSvc{}
	sms := &testmock.SMSSvc{}
	email := &testmock.EmailSvc{}
	oauth := &testmock.OAuthSvc{}

	svc := auth.NewService(users, tokens, cache, sms, email, oauth, 6, []string{"admin@test.com"}, nil)
	return svc, users, tokens, cache
}

func TestRequestOTP_SendsSMS(t *testing.T) {
	_, users, _, cache := newService(t)
	_ = users // not called for phone
	cache.On("Get", mock.Anything, "otp_attempts:+919999999999", mock.Anything).Return(apperror.ErrCacheMiss)
	cache.On("Set", mock.Anything, mock.MatchedBy(func(k string) bool { return k[:4] == "otp:" }), mock.Anything, mock.Anything).Return(nil)
	cache.On("Set", mock.Anything, "otp_attempts:+919999999999", mock.Anything, mock.Anything).Return(nil)

	// Use a dedicated SMS mock
	smsM := &testmock.SMSSvc{}
	smsM.On("SendOTP", mock.Anything, "+919999999999", mock.AnythingOfType("string")).Return(nil)
	emailM := &testmock.EmailSvc{}
	oauthM := &testmock.OAuthSvc{}
	usersM := &testmock.UserRepo{}
	tokensM := &testmock.TokenSvc{}

	svc := auth.NewService(usersM, tokensM, cache, smsM, emailM, oauthM, 6, []string{"admin@test.com"}, nil)
	err := svc.RequestOTP(context.Background(), "+919999999999")
	assert.NoError(t, err)
	smsM.AssertExpectations(t)
}

func TestVerifyOTP_InvalidOTP(t *testing.T) {
	svc, _, _, cache := newService(t)
	cache.On("Get", mock.Anything, "otp:+919999999999", mock.Anything).Return(apperror.ErrCacheMiss)

	_, _, err := svc.VerifyOTP(context.Background(), "+919999999999", "000000")
	assert.True(t, errors.Is(err, apperror.ErrOTPInvalid))
}

func TestVerifyOTP_BannedUser(t *testing.T) {
	svc, users, tokens, cache := newService(t)

	otp := "123456"
	userID := uuid.New()
	bannedUser := &entity.User{ID: userID, Phone: "+919999999999", IsBanned: true}

	cache.On("Get", mock.Anything, "otp:+919999999999", mock.AnythingOfType("*string")).
		Run(func(args mock.Arguments) {
			*args[2].(*string) = otp
		}).Return(nil)
	cache.On("Delete", mock.Anything, "otp:+919999999999", "otp_attempts:+919999999999").Return(nil)
	users.On("FindByPhone", mock.Anything, "+919999999999").Return(bannedUser, nil)
	_ = tokens // not called for banned user

	_, _, err := svc.VerifyOTP(context.Background(), "+919999999999", otp)
	assert.True(t, errors.Is(err, apperror.ErrForbidden))
}

func TestVerifyOTP_NewUserCreated(t *testing.T) {
	_, users, tokens, cache := newService(t)
	otp := "123456"
	smsM := &testmock.SMSSvc{}
	emailM := &testmock.EmailSvc{}
	oauthM := &testmock.OAuthSvc{}

	cache.On("Get", mock.Anything, "otp:+919999999999", mock.AnythingOfType("*string")).
		Run(func(args mock.Arguments) { *args[2].(*string) = otp }).Return(nil)
	cache.On("Delete", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	users.On("FindByPhone", mock.Anything, "+919999999999").Return(nil, apperror.ErrNotFound)
	users.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)
	tokens.On("GenerateAccessToken", mock.Anything).Return("access-token", nil)
	tokens.On("GenerateRefreshToken", mock.Anything).Return("refresh-token", nil)

	svc := auth.NewService(users, tokens, cache, smsM, emailM, oauthM, 6, []string{"admin@test.com"}, nil)
	pair, user, err := svc.VerifyOTP(context.Background(), "+919999999999", otp)
	assert.NoError(t, err)
	assert.Equal(t, "access-token", pair.AccessToken)
	assert.Equal(t, entity.UserRoleBuyer, user.Role)
	users.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}
