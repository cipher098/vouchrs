package token

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type jwtService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	cache         port.CacheService
}

type accessClaims struct {
	jwt.RegisteredClaims
	Role  string `json:"role"`
	Email string `json:"email"`
}

type refreshClaims struct {
	jwt.RegisteredClaims
}

func NewJWTService(accessSecret, refreshSecret string, accessTTLMin, refreshTTLDay int, cache port.CacheService) port.TokenService {
	return &jwtService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     time.Duration(accessTTLMin) * time.Minute,
		refreshTTL:    time.Duration(refreshTTLDay) * 24 * time.Hour,
		cache:         cache,
	}
}

func (s *jwtService) GenerateAccessToken(claims port.TokenClaims) (string, error) {
	now := time.Now()
	c := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
		Role:  claims.Role,
		Email: claims.Email,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(s.accessSecret)
}

func (s *jwtService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	c := refreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return token.SignedString(s.refreshSecret)
}

func (s *jwtService) ValidateAccessToken(tokenStr string) (*port.TokenClaims, error) {
	c := &accessClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid subject in token: %w", err)
	}
	return &port.TokenClaims{UserID: userID, Role: c.Role, Email: c.Email}, nil
}

func (s *jwtService) ValidateRefreshToken(tokenStr string) (uuid.UUID, error) {
	c := &refreshClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, c, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	return uuid.Parse(c.Subject)
}

func (s *jwtService) RevokeToken(ctx context.Context, token string) error {
	// Store in Redis with TTL matching token expiry — any non-empty value marks revocation.
	return s.cache.Set(ctx, revokedKey(token), "1", s.accessTTL)
}

func (s *jwtService) IsRevoked(ctx context.Context, token string) (bool, error) {
	return s.cache.Exists(ctx, revokedKey(token))
}

func revokedKey(token string) string {
	// Use only the last 16 chars to keep the key short; collisions are acceptable
	// since each key also carries its own TTL.
	if len(token) > 16 {
		return "revoked:" + token[len(token)-16:]
	}
	return "revoked:" + token
}
