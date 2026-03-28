package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
)

// Authenticate validates the JWT Bearer token and injects claims into context.
// Returns 401 if no token or invalid token.
func Authenticate(tokens port.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				response.Error(w, apperror.ErrUnauthorized)
				return
			}

			revoked, err := tokens.IsRevoked(r.Context(), token)
			if err == nil && revoked {
				response.Error(w, apperror.ErrUnauthorized)
				return
			}

			claims, err := tokens.ValidateAccessToken(token)
			if err != nil {
				response.Error(w, apperror.ErrUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextkey.UserID, claims.UserID)
			ctx = context.WithValue(ctx, contextkey.UserRole, claims.Role)
			ctx = context.WithValue(ctx, contextkey.UserEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that blocks requests from users without the required role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(contextkey.UserRole).(string)
			if !ok {
				response.Error(w, apperror.ErrUnauthorized)
				return
			}
			if _, permitted := allowed[role]; !permitted {
				response.Error(w, apperror.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
