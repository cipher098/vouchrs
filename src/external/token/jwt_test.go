package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gothi/vouchrs/src/external/token"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/testmock"
)

func newTokenSvc(t *testing.T, cache *testmock.CacheSvc) port.TokenService {
	t.Helper()
	return token.NewJWTService("access-secret", "refresh-secret", 15, 7, cache)
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	svc := newTokenSvc(t, &testmock.CacheSvc{})
	userID := uuid.New()
	claims := port.TokenClaims{UserID: userID, Role: "buyer", Email: "u@test.com"}

	tok, err := svc.GenerateAccessToken(claims)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	got, err := svc.ValidateAccessToken(tok)
	require.NoError(t, err)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, "buyer", got.Role)
	assert.Equal(t, "u@test.com", got.Email)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	svc := newTokenSvc(t, &testmock.CacheSvc{})
	other := token.NewJWTService("different-secret", "refresh-secret", 15, 7, &testmock.CacheSvc{})

	tok, _ := other.GenerateAccessToken(port.TokenClaims{UserID: uuid.New(), Role: "buyer"})
	_, err := svc.ValidateAccessToken(tok)
	assert.Error(t, err)
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	svc := newTokenSvc(t, &testmock.CacheSvc{})
	userID := uuid.New()

	tok, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	got, err := svc.ValidateRefreshToken(tok)
	require.NoError(t, err)
	assert.Equal(t, userID, got)
}

func TestRevokeToken(t *testing.T) {
	cache := &testmock.CacheSvc{}
	svc := newTokenSvc(t, cache)
	ctx := context.Background()

	tok, _ := svc.GenerateAccessToken(port.TokenClaims{UserID: uuid.New(), Role: "seller"})

	cache.On("Set", ctx, "revoked:"+tok[len(tok)-16:], "1", 15*time.Minute).Return(nil)

	err := svc.RevokeToken(ctx, tok)
	assert.NoError(t, err)
	cache.AssertExpectations(t)
}
